package futures

import (
	"encoding/json"
	"fmt"
	"strings"
	"sync"
	"time"

	"github.com/bitly/go-simplejson"
	"github.com/gorilla/websocket"

	"github.com/aiviaio/go-binance/v2/common"
)

// LiveStreamRateLimit is the rate limit for messages sent to the live stream WS connection
// In fact Binance has a limit of ten messages per second, but we are using nine to have a margin
const LiveStreamRateLimit = 9

// streamCategory identifies which Binance futures WS endpoint a stream lives on.
// After 2026-04-23 Binance splits futures WS traffic: high-frequency streams
// (@depth, @bookTicker) go to /public/stream; regular market data
// (@aggTrade, @kline, @ticker, @markPrice, etc.) goes to /market/stream.
type streamCategory int

const (
	categoryMarket streamCategory = iota
	categoryPublic
)

// categoryForStream returns the correct endpoint category for a given stream name.
// Examples: "btcusdt@depth" -> public, "btcusdt@aggTrade" -> market.
func categoryForStream(stream string) streamCategory {
	// Order matters: check longer/more specific suffixes first.
	switch {
	case strings.Contains(stream, "@depth"), // @depth, @depth@500ms, @depth5, @depth10...
		strings.Contains(stream, "@bookTicker"),
		stream == "!bookTicker":
		return categoryPublic
	default:
		return categoryMarket
	}
}

// liveStreamConn owns a single WS connection to one endpoint (market or public).
// WsLiveStreamsService keeps one of these per category.
type liveStreamConn struct {
	endpoint string
	category streamCategory

	conn        *websocket.Conn
	connMutex   sync.Mutex
	rateLimiter *time.Ticker

	doneC      chan struct{}
	errHandler ErrHandler

	wsHandlersMutex sync.Mutex
	wsHandlers      map[string]WsHandler

	ops      map[int64]*liveStreamOp
	opsMutex sync.Mutex
}

func newLiveStreamConn(endpoint string, category streamCategory, errHandler ErrHandler, doneC chan struct{}) *liveStreamConn {
	return &liveStreamConn{
		endpoint:    endpoint,
		category:    category,
		rateLimiter: time.NewTicker(time.Second / LiveStreamRateLimit),
		doneC:       doneC,
		errHandler:  errHandler,
		wsHandlers:  make(map[string]WsHandler),
		ops:         make(map[int64]*liveStreamOp),
	}
}

type WsLiveStreamsService struct {
	errHandler ErrHandler
	doneC      chan struct{}

	marketConn *liveStreamConn
	publicConn *liveStreamConn
}

type liveStreamOp struct {
	ID     int64    `json:"id"`
	Method string   `json:"method"`
	Params []string `json:"params"`
	// stopC is used for common.LiveMethodSubscribe to which a signal is sent when the stream should be stopped.
	// If stopC was closed, nothing will happen.
	// If stopC is nil, the stream represents a common.LiveMethodUnsubscribe operation.
	stopC chan struct{}
}

func NewWsLiveStreamsService(errHandler ErrHandler) (service *WsLiveStreamsService, doneC chan struct{}) {
	doneC = make(chan struct{})
	service = &WsLiveStreamsService{
		errHandler: errHandler,
		doneC:      doneC,
		marketConn: newLiveStreamConn(getCombinedMarketEndpoint(), categoryMarket, errHandler, doneC),
		publicConn: newLiveStreamConn(getCombinedPublicEndpoint(), categoryPublic, errHandler, doneC),
	}
	return
}

// connFor picks the right per-category connection for the given stream name.
func (s *WsLiveStreamsService) connFor(stream string) *liveStreamConn {
	if categoryForStream(stream) == categoryPublic {
		return s.publicConn
	}
	return s.marketConn
}

// WsAllMarketsStatServe serve websocket that push 24hr statistics for all market every second
func (s *WsLiveStreamsService) WsAllMarketsStatServe(handler WsAllMarketTickerHandler) (stopC chan<- struct{}, err error) {
	c := s.connFor(common.LiveStreamAllMarketTickers)
	if err = c.connectIfNotConnected(); err != nil {
		return
	}
	wsHandler := func(message []byte) {
		var event WsAllMarketTickerEvent
		err = json.Unmarshal(message, &event)
		if err != nil {
			s.errHandler(fmt.Errorf("unable to unmarshal event: %w", err))
			return
		}
		handler(event)
	}
	if err = c.addWsHandler(common.LiveStreamAllMarketTickers, wsHandler); err != nil {
		return
	}

	return c.subscribe(common.LiveStreamAllMarketTickers)
}

// WsAllMiniMarketsStatServe serve websocket that push mini 24hr statistics for all market every second
func (s *WsLiveStreamsService) WsAllMiniMarketsStatServe(handler WsAllMiniMarketTickerHandler) (stopC chan<- struct{}, err error) {
	c := s.connFor(common.LiveStreamAllMiniMarketTickers)
	if err = c.connectIfNotConnected(); err != nil {
		return
	}
	wsHandler := func(message []byte) {
		var event WsAllMiniMarketTickerEvent
		err = json.Unmarshal(message, &event)
		if err != nil {
			s.errHandler(fmt.Errorf("unable to unmarshal mini ticker event: %w", err))
			return
		}
		handler(event)
	}
	if err = c.addWsHandler(common.LiveStreamAllMiniMarketTickers, wsHandler); err != nil {
		return
	}

	return c.subscribe(common.LiveStreamAllMiniMarketTickers)
}

// WsAggTradeServe serve websocket aggregate handler with a symbol
func (s *WsLiveStreamsService) WsAggTradeServe(symbol string, handler WsAggTradeHandler) (stopC chan<- struct{}, err error) {
	streamName := strings.ToLower(symbol) + common.LiveStreamAggTrade
	c := s.connFor(streamName)
	if err = c.connectIfNotConnected(); err != nil {
		return
	}
	wsHandler := wsHandlerWrapper(handler, s.errHandler)
	if err = c.addWsHandler(streamName, wsHandler); err != nil {
		return
	}

	return c.subscribe(streamName)
}

// WsDepthServe serve websocket depth handler with a symbol, using 1sec updates
func (s *WsLiveStreamsService) WsDepthServe(symbol string, handler WsDepthHandler) (stopC chan<- struct{}, err error) {
	streamName := strings.ToLower(symbol) + common.LiveStreamDepth
	c := s.connFor(streamName)
	if err = c.connectIfNotConnected(); err != nil {
		return
	}
	wsHandler := wsDepthHandlerWrapper(handler, s.errHandler)
	if err = c.addWsHandler(streamName, wsHandler); err != nil {
		return
	}

	return c.subscribe(streamName)
}

// WsAvgPriceServe serve websocket that pushes 1m average price for a specified symbol.
func (s *WsLiveStreamsService) WsAvgPriceServe(symbol string, handler WsAvgPriceHandler) (stopC chan<- struct{}, err error) {
	streamName := strings.ToLower(symbol) + common.LiveStreamAvgPrice
	c := s.connFor(streamName)
	if err = c.connectIfNotConnected(); err != nil {
		return
	}
	wsHandler := wsHandlerWrapper(handler, s.errHandler)
	if err = c.addWsHandler(streamName, wsHandler); err != nil {
		return
	}

	return c.subscribe(streamName)
}

// WsKlineServe serve websocket kline handler with a symbol and interval like 15m, 30s
func (s *WsLiveStreamsService) WsKlineServe(symbol string, interval string, handler WsKlineHandler) (stopC chan struct{}, err error) {
	streamName := strings.ToLower(symbol) + common.LiveStreamKline + interval
	c := s.connFor(streamName)
	if err = c.connectIfNotConnected(); err != nil {
		return
	}
	wsHandler := wsHandlerWrapper(handler, s.errHandler)
	if err = c.addWsHandler(streamName, wsHandler); err != nil {
		return
	}

	stopC2, err := c.subscribe(streamName)
	if err != nil {
		return
	}
	return stopC2, nil
}

// WsMarkPriceServe serve websocket that pushes price and funding rate for a single symbol.
func (s *WsLiveStreamsService) WsMarkPriceServe(symbol string, handler WsMarkPriceHandler) (stopC chan struct{}, err error) {
	streamName := strings.ToLower(symbol) + common.LiveStreamMarkPrice
	c := s.connFor(streamName)
	if err = c.connectIfNotConnected(); err != nil {
		return
	}
	wsHandler := wsHandlerWrapper(handler, s.errHandler)
	if err = c.addWsHandler(streamName, wsHandler); err != nil {
		return
	}

	stopC2, err := c.subscribe(streamName)
	if err != nil {
		return
	}
	return stopC2, nil
}

// --- liveStreamConn methods ---

// subscribe sends a subscription message to the WebSocket
func (c *liveStreamConn) subscribe(streams ...string) (stopC chan struct{}, err error) {
	stopC = make(chan struct{})
	op := &liveStreamOp{ID: time.Now().UnixNano(), Method: common.LiveMethodSubscribe, Params: streams, stopC: stopC}
	c.opsMutex.Lock()
	c.ops[op.ID] = op
	c.opsMutex.Unlock()
	go c.waitForStop(op)

	<-c.rateLimiter.C
	c.connMutex.Lock()
	defer c.connMutex.Unlock()
	if err = c.conn.WriteJSON(op); err != nil {
		c.errHandler(fmt.Errorf("unable to write JSON: %w", err))
	}
	return
}

// unsubscribe sends an unsubscription message to the WebSocket
func (c *liveStreamConn) unsubscribe(streams ...string) error {
	op := &liveStreamOp{ID: time.Now().UnixMilli(), Method: common.LiveMethodUnsubscribe, Params: streams}
	c.opsMutex.Lock()
	c.ops[op.ID] = op
	c.opsMutex.Unlock()

	// Remove handlers for unsubscribed streams
	c.wsHandlersMutex.Lock()
	for _, stream := range streams {
		delete(c.wsHandlers, stream)
	}
	c.wsHandlersMutex.Unlock()

	<-c.rateLimiter.C
	c.connMutex.Lock()
	defer c.connMutex.Unlock()
	if err := c.conn.WriteJSON(op); err != nil {
		return fmt.Errorf("unable to write JSON: %w", err)
	}
	return nil
}

// readMessages listens for incoming messages and handles them
func (c *liveStreamConn) readMessages() {
	defer c.conn.Close()
	defer c.rateLimiter.Stop()

	for {
		select {
		case <-c.doneC:
			return
		default:
			messageType, message, err := c.conn.ReadMessage()
			if err != nil {
				c.errHandler(fmt.Errorf("unable to read message: %w", err))
				return
			}
			switch messageType {
			case websocket.TextMessage:
				c.handleMessage(message)
			case websocket.PongMessage:
				// ignore
			}
		}
	}
}

// handleMessage processes incoming WebSocket messages
func (c *liveStreamConn) handleMessage(message []byte) {
	j, err := newJSON(message)
	if err != nil {
		c.errHandler(fmt.Errorf("unable to unmarshal JSON response: %w", err))
		return
	}

	// Check message type and process accordingly
	if errorMsg, ok := j.CheckGet("error"); ok {
		// if 'error' is not nil, then the server has sent an error message
		id, _ := j.Get("id").Int64()
		c.opsMutex.Lock()
		op := c.ops[id]

		delete(c.ops, id) // delete the operation from the map as it was not successful
		c.opsMutex.Unlock()
		if op != nil {
			c.errHandler(fmt.Errorf("operation was not successful, opMethod: %s, opParams: %v, error: %v",
				op.Method, op.Params, errorMsg))
		} else {
			c.errHandler(fmt.Errorf("error was received: %v", errorMsg))
		}
	} else if _, ok := j.CheckGet("result"); ok {
		// if 'result' is nil, then the server has sent a message about successful operation
		id, _ := j.Get("id").Int64()
		c.opsMutex.Lock()
		if op, ok := c.ops[id]; ok && op.stopC == nil {
			delete(c.ops, id) // delete the operation from the map as it was successful, and we don't need it anymore
		}
		c.opsMutex.Unlock()
	} else if streamRaw, ok := j.CheckGet("stream"); ok {
		// if 'stream' is not nil, then the server has sent a stream message
		c.handlerStreamMessage(streamRaw.MustString(), j.Get("data"))
	} else {
		// if none of the above, then the server has sent an unexpected message
		c.errHandler(fmt.Errorf("unexpected message: %v", j))
	}
}

func (c *liveStreamConn) connectIfNotConnected() error {
	c.connMutex.Lock()
	defer c.connMutex.Unlock()
	if c.conn == nil {
		return c.connect()
	}

	return nil
}

func (c *liveStreamConn) connect() error {
	conn, _, err := websocket.DefaultDialer.Dial(c.endpoint, nil)
	if err != nil {
		return fmt.Errorf("unable to dial websocket, endpoint: %s: %w", c.endpoint, err)
	}
	conn.SetPingHandler(func(pingPayload string) error {
		c.connMutex.Lock()
		defer c.connMutex.Unlock()
		<-c.rateLimiter.C
		return conn.WriteControl(websocket.PongMessage, []byte(pingPayload), time.Now().Add(WebsocketTimeout))
	})
	c.conn = conn

	go c.readMessages()
	return nil
}

func (c *liveStreamConn) addWsHandler(streamName string, handler WsHandler) error {
	c.wsHandlersMutex.Lock()
	defer c.wsHandlersMutex.Unlock()
	if _, ok := c.wsHandlers[streamName]; ok {
		return fmt.Errorf("stream %s already connected", streamName)
	}
	c.wsHandlers[streamName] = handler
	return nil
}

func (c *liveStreamConn) handlerStreamMessage(stream string, data *simplejson.Json) {
	c.wsHandlersMutex.Lock()
	defer c.wsHandlersMutex.Unlock()
	if handler, ok := c.wsHandlers[stream]; ok {
		encodedData, err := data.Encode()
		if err != nil {
			c.errHandler(fmt.Errorf("unable to encode data: %w", err))
			return
		}
		handler(encodedData)
	} else {
		c.errHandler(fmt.Errorf("handler for stream %s not found", stream))
	}
}

func (c *liveStreamConn) waitForStop(op *liveStreamOp) {
	if op.Method != common.LiveMethodSubscribe {
		return
	}
	<-op.stopC
	if err := c.unsubscribe(op.Params...); err != nil {
		c.errHandler(fmt.Errorf("unable to unsubscribe: %w", err))
	}
}

func wsHandlerWrapper[T any](handler func(t *T), errHandler ErrHandler) WsHandler {
	return func(message []byte) {
		event := new(T)
		err := json.Unmarshal(message, event)
		if err != nil {
			errHandler(err)
			return
		}
		handler(event)
	}
}
