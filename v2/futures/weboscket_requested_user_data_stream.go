package futures

import (
	"crypto/hmac"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"net/http"
	"net/url"
	"sort"
	"strings"
	"sync"
	"sync/atomic"
	"time"

	"github.com/gorilla/websocket"
)

const (
	// WebSocket API methods for Futures
	wsAPIMethodAccountBalance  = "account.balance"
	wsAPIMethodAccountPosition = "account.position"

	// userDataStreamRateLimit represent number of messages that can be sent per second
	userDataStreamRateLimit = 4

	// userDataStreamReadDeadline is the maximum time we will wait on the User
	// Data Stream socket without receiving ANY frame (a real event or, far more
	// commonly on an idle account, a server ping) before we treat the socket as
	// dead. Binance sends a ping on this stream roughly every 3 minutes, so 6
	// minutes tolerates one fully missed ping as scheduling/network slack.
	//
	// Why this exists: without a read deadline, ReadMessage blocks forever when
	// the underlying TCP connection dies without a FIN/RST (the exact silent
	// death that froze account balances in prod). No error ever fired, so the
	// errHandler never ran, hasError was never set, and the supervisor never
	// reconnected. The deadline turns that silent hang into a timeout error
	// that propagates through errHandler → hasError → supervisor restart.
	userDataStreamReadDeadline = 6 * time.Minute
)

// WsRequestedUserDataStreamService handles both:
// 1. User Data Stream connection (for real-time events via listenKey)
// 2. WebSocket API connection (for request-response queries like balance/position)
type WsRequestedUserDataStreamService struct {
	listenKey string
	apiKey    string
	secretKey string
	keepAlive bool
	// proxyURL, when non-empty, routes BOTH the User Data Stream and the
	// WebSocket API handshakes through this HTTP proxy. Empty = direct dial.
	proxyURL string

	// User Data Stream connection (for events)
	streamConn      *websocket.Conn
	streamConnMutex sync.Mutex

	// WebSocket API connection (for requests)
	apiConn      *websocket.Conn
	apiConnMutex sync.Mutex

	rateLimiter *time.Ticker
	requestID   int64

	// lastActivity is updated on every frame read from the User Data Stream —
	// real events AND server pings. It lets callers/tests observe liveness on
	// an otherwise idle stream (which emits no events for hours but is kept
	// alive by ~3-min server pings). Stores time.Time.
	lastActivity atomic.Value

	doneC        chan struct{}
	stopC        chan struct{}
	eventHandler WsUserDataHandler
	errHandler   ErrHandler

	respHandlers      map[string]wsAPIResponseHandler
	respHandlersMutex sync.Mutex
}

type wsAPIResponseHandler struct {
	tag     string
	handler func(result json.RawMessage)
}

func NewWsRequestedUserDataStreamService(listenKey, apiKey, secretKey string, keepAlive bool, errHandler ErrHandler) (service *WsRequestedUserDataStreamService, doneC, stopC chan struct{}) {
	return NewWsRequestedUserDataStreamServiceWithProxy(listenKey, apiKey, secretKey, "", keepAlive, errHandler)
}

// NewWsRequestedUserDataStreamServiceWithProxy is like
// NewWsRequestedUserDataStreamService but routes both the User Data Stream and
// the WebSocket API handshakes through the given HTTP proxy
// (format: http://user:pass@host:port). In production, opening every futures
// account's user-data stream directly from one pod IP trips Binance's per-IP
// connection limits (mass "use of closed network connection") and the API-key
// IP-whitelist check (ip_forbidden). A WSS handshake runs over an HTTP CONNECT
// tunnel that pins a single upstream TCP connection for the socket's lifetime,
// so a rotating proxy gives each account its own stable exit IP. An empty
// proxyURL falls back to a direct connection via DefaultDialer.
func NewWsRequestedUserDataStreamServiceWithProxy(listenKey, apiKey, secretKey, proxyURL string, keepAlive bool, errHandler ErrHandler) (service *WsRequestedUserDataStreamService, doneC, stopC chan struct{}) {
	service = &WsRequestedUserDataStreamService{
		listenKey:    listenKey,
		apiKey:       apiKey,
		secretKey:    secretKey,
		keepAlive:    keepAlive,
		proxyURL:     proxyURL,
		rateLimiter:  time.NewTicker(time.Second / userDataStreamRateLimit),
		doneC:        make(chan struct{}),
		stopC:        make(chan struct{}),
		errHandler:   errHandler,
		respHandlers: make(map[string]wsAPIResponseHandler),
	}
	service.lastActivity.Store(time.Now())
	doneC = service.doneC
	stopC = service.stopC
	return
}

// LastActivity returns the time the User Data Stream last received any frame
// (event or server ping). A stalling value means the stream is going silent.
func (s *WsRequestedUserDataStreamService) LastActivity() time.Time {
	if v, ok := s.lastActivity.Load().(time.Time); ok {
		return v
	}
	return time.Time{}
}

// dialer returns a websocket dialer honoring s.proxyURL. When proxyURL is empty
// it returns the shared DefaultDialer (direct connection).
func (s *WsRequestedUserDataStreamService) dialer() (*websocket.Dialer, error) {
	if s.proxyURL == "" {
		return websocket.DefaultDialer, nil
	}
	parsed, err := url.Parse(s.proxyURL)
	if err != nil {
		return nil, fmt.Errorf("failed to parse proxy URL: %w", err)
	}
	return &websocket.Dialer{
		Proxy:            http.ProxyURL(parsed),
		HandshakeTimeout: 45 * time.Second,
	}, nil
}

func (s *WsRequestedUserDataStreamService) APIKey(apiKey string) *WsRequestedUserDataStreamService {
	s.apiKey = apiKey
	return s
}

func (s *WsRequestedUserDataStreamService) SecretKey(secretKey string) *WsRequestedUserDataStreamService {
	s.secretKey = secretKey
	return s
}

func (s *WsRequestedUserDataStreamService) KeepAlive(keepAlive bool) *WsRequestedUserDataStreamService {
	s.keepAlive = keepAlive
	return s
}

func (s *WsRequestedUserDataStreamService) ListenKey(listenKey string) *WsRequestedUserDataStreamService {
	s.listenKey = listenKey
	return s
}

// StartUserDataStream connects to User Data Stream for real-time events
func (s *WsRequestedUserDataStreamService) StartUserDataStream(handler WsUserDataHandler) error {
	if err := s.connectUserDataStream(); err != nil {
		return err
	}
	if err := s.connectWebSocketAPI(); err != nil {
		return err
	}
	s.eventHandler = handler
	return nil
}

// Response types for WebSocket API
type WsUserDataPositionResponse struct {
	Positions []WsUserDataPosition `json:"positions"`
}

type WsUserDataPosition struct {
	EntryPrice       string `json:"entryPrice"`
	MarginType       string `json:"marginType"`
	IsAutoAddMargin  string `json:"isAutoAddMargin"`
	IsolatedMargin   string `json:"isolatedMargin"`
	Leverage         string `json:"leverage"`
	LiquidationPrice string `json:"liquidationPrice"`
	MarkPrice        string `json:"markPrice"`
	MaxQty           string `json:"maxQty"`
	Amount           string `json:"positionAmt"`
	Symbol           string `json:"symbol"`
	UnrealizedPnL    string `json:"unRealizedProfit"`
	Side             string `json:"positionSide"`
	UpdateTime       int64  `json:"updateTime"`
}

type WsUserDataPositionResponseHandler func(response *WsUserDataPositionResponse)

// RequestUserPosition requests current position information via WebSocket API
func (s *WsRequestedUserDataStreamService) RequestUserPosition(handler WsUserDataPositionResponseHandler) error {
	return s.sendAPIRequest(wsAPIMethodAccountPosition, nil, func(result json.RawMessage) {
		if handler == nil {
			return
		}
		// account.position returns array of positions directly
		var positions []WsUserDataPosition
		if err := json.Unmarshal(result, &positions); err != nil {
			s.errHandler(fmt.Errorf("unable to unmarshal position response: %w", err))
			return
		}
		handler(&WsUserDataPositionResponse{Positions: positions})
	})
}

type WsUserDataBalanceResponse struct {
	AccountAlias string              `json:"accountAlias"`
	Balances     []WsUserDataBalance `json:"balances"`
}

type WsUserDataBalance struct {
	Asset              string `json:"asset"`
	Balance            string `json:"balance"`
	CrossWalletBalance string `json:"crossWalletBalance"`
	CrossUnPnl         string `json:"crossUnPnl"`
	AvailableBalance   string `json:"availableBalance"`
	MaxWithdrawAmount  string `json:"maxWithdrawAmount"`
	UpdateTime         int64  `json:"updateTime"`
}

type WsUserDataBalanceResponseHandler func(response *WsUserDataBalanceResponse)

// RequestUserBalance requests account balance via WebSocket API
func (s *WsRequestedUserDataStreamService) RequestUserBalance(handler WsUserDataBalanceResponseHandler) error {
	return s.sendAPIRequest(wsAPIMethodAccountBalance, nil, func(result json.RawMessage) {
		if handler == nil {
			return
		}
		// account.balance returns array of balances directly
		var balances []WsUserDataBalance
		if err := json.Unmarshal(result, &balances); err != nil {
			s.errHandler(fmt.Errorf("unable to unmarshal balance response: %w", err))
			return
		}
		handler(&WsUserDataBalanceResponse{Balances: balances})
	})
}

type WsUserDataBalanceAccountInfoResponse struct {
	FeeTier      int    `json:"feeTier"`
	CanTrade     bool   `json:"canTrade"`
	CanDeposit   bool   `json:"canDeposit"`
	CanWithdraw  bool   `json:"canWithdraw"`
	AccountAlias string `json:"accountAlias"`
}

type WsUserDataBalanceAccountInfoResponseHandler func(response *WsUserDataBalanceAccountInfoResponse)

func (s *WsRequestedUserDataStreamService) RequestAccountInformation(handler WsUserDataBalanceAccountInfoResponseHandler) error {
	return s.sendAPIRequest("account.status", nil, func(result json.RawMessage) {
		if handler == nil {
			return
		}
		response := new(WsUserDataBalanceAccountInfoResponse)
		if err := json.Unmarshal(result, response); err != nil {
			s.errHandler(fmt.Errorf("unable to unmarshal account info response: %w", err))
			return
		}
		handler(response)
	})
}

// connectUserDataStream connects to User Data Stream endpoint for real-time events
func (s *WsRequestedUserDataStreamService) connectUserDataStream() error {
	s.streamConnMutex.Lock()
	defer s.streamConnMutex.Unlock()

	if s.streamConn != nil {
		return fmt.Errorf("user data stream already connected")
	}

	// User Data Stream uses private endpoint with listenKey
	endpoint := fmt.Sprintf("%s/%s", getWsPrivateEndpoint(), s.listenKey)
	dialer, err := s.dialer()
	if err != nil {
		return err
	}
	conn, _, err := dialer.Dial(endpoint, nil)
	if err != nil {
		return fmt.Errorf("unable to dial user data stream, endpoint: %s: %w", endpoint, err)
	}
	// Arm the read deadline immediately so a connection that dies before the
	// first server ping is still detected.
	_ = conn.SetReadDeadline(time.Now().Add(userDataStreamReadDeadline))
	conn.SetPingHandler(func(pingPayload string) error {
		s.streamConnMutex.Lock()
		defer s.streamConnMutex.Unlock()
		// A server ping is proof the connection is alive — push the read
		// deadline forward so an otherwise idle (eventless) stream is not
		// killed by its own watchdog, and record the activity for observers.
		_ = conn.SetReadDeadline(time.Now().Add(userDataStreamReadDeadline))
		s.lastActivity.Store(time.Now())
		return conn.WriteControl(websocket.PongMessage, []byte(pingPayload), time.Now().Add(WebsocketTimeout))
	})
	s.streamConn = conn

	go s.readStreamMessages()
	return nil
}

// connectWebSocketAPI connects to WebSocket API endpoint for request-response queries
func (s *WsRequestedUserDataStreamService) connectWebSocketAPI() error {
	s.apiConnMutex.Lock()
	defer s.apiConnMutex.Unlock()

	if s.apiConn != nil {
		return fmt.Errorf("websocket API already connected")
	}

	// WebSocket API endpoint (no listenKey needed, uses signature auth)
	endpoint := getWsAPIFuturesEndpoint()
	dialer, err := s.dialer()
	if err != nil {
		return err
	}
	conn, _, err := dialer.Dial(endpoint, nil)
	if err != nil {
		return fmt.Errorf("unable to dial websocket API, endpoint: %s: %w", endpoint, err)
	}
	conn.SetPingHandler(func(pingPayload string) error {
		s.apiConnMutex.Lock()
		defer s.apiConnMutex.Unlock()
		return conn.WriteControl(websocket.PongMessage, []byte(pingPayload), time.Now().Add(WebsocketTimeout))
	})
	s.apiConn = conn

	go s.readAPIMessages()
	return nil
}

// readStreamMessages reads messages from User Data Stream
func (s *WsRequestedUserDataStreamService) readStreamMessages() {
	silent := false
	go func() {
		select {
		case <-s.stopC:
			silent = true
		case <-s.doneC:
		}
		s.streamConnMutex.Lock()
		if s.streamConn != nil {
			s.streamConn.Close()
		}
		s.streamConnMutex.Unlock()
	}()
	for {
		messageType, message, err := s.streamConn.ReadMessage()
		if err != nil {
			if !silent {
				s.errHandler(fmt.Errorf("unable to read stream message: %w", err))
			}
			return
		}
		// Any successfully read frame means the socket is alive; extend the
		// deadline so a busy stream never times out, and record activity.
		_ = s.streamConn.SetReadDeadline(time.Now().Add(userDataStreamReadDeadline))
		s.lastActivity.Store(time.Now())
		if messageType == websocket.TextMessage {
			s.handleStreamEvent(message)
		}
	}
}

// readAPIMessages reads messages from WebSocket API
func (s *WsRequestedUserDataStreamService) readAPIMessages() {
	silent := false
	go func() {
		select {
		case <-s.stopC:
			silent = true
		case <-s.doneC:
		}
		s.apiConnMutex.Lock()
		if s.apiConn != nil {
			s.apiConn.Close()
		}
		s.apiConnMutex.Unlock()
	}()
	for {
		messageType, message, err := s.apiConn.ReadMessage()
		if err != nil {
			if !silent {
				s.errHandler(fmt.Errorf("unable to read API message: %w", err))
			}
			return
		}
		if messageType == websocket.TextMessage {
			s.handleAPIResponse(message)
		}
	}
}

func (s *WsRequestedUserDataStreamService) handleStreamEvent(eventRaw []byte) {
	event := new(WsUserDataEvent)
	if err := json.Unmarshal(eventRaw, event); err != nil {
		s.errHandler(fmt.Errorf("unable to unmarshal stream event: %w", err))
		return
	}
	// For ALGO_UPDATE event, parse AlgoUpdate from "o" field
	if event.Event == UserDataEventTypeAlgoUpdate {
		var rawEvent struct {
			O WsAlgoUpdate `json:"o"`
		}
		if err := json.Unmarshal(eventRaw, &rawEvent); err == nil {
			event.AlgoUpdate = rawEvent.O
		}
	}
	if s.eventHandler != nil {
		s.eventHandler(event)
	}
}

func (s *WsRequestedUserDataStreamService) handleAPIResponse(responseRaw []byte) {
	// Check for error response
	var errResp struct {
		ID     string `json:"id"`
		Status int    `json:"status"`
		Error  *struct {
			Code int    `json:"code"`
			Msg  string `json:"msg"`
		} `json:"error"`
	}
	if err := json.Unmarshal(responseRaw, &errResp); err == nil && errResp.Error != nil {
		s.errHandler(fmt.Errorf("API error response: code: %d, message: %s", errResp.Error.Code, errResp.Error.Msg))
		return
	}

	// Parse successful response
	var response wsAPIResponse
	if err := json.Unmarshal(responseRaw, &response); err != nil {
		s.errHandler(fmt.Errorf("unable to unmarshal API response: %w", err))
		return
	}

	s.respHandlersMutex.Lock()
	handler, ok := s.respHandlers[response.ID]
	delete(s.respHandlers, response.ID)
	s.respHandlersMutex.Unlock()

	if !ok {
		// Response for unknown request, ignore
		return
	}

	if response.Status != 200 {
		s.errHandler(fmt.Errorf("API response status: %d for request %s", response.Status, response.ID))
		return
	}

	handler.handler(response.Result)
}

// WebSocket API request/response types
type wsAPIRequest struct {
	ID     string                 `json:"id"`
	Method string                 `json:"method"`
	Params map[string]interface{} `json:"params,omitempty"`
}

type wsAPIResponse struct {
	ID     string          `json:"id"`
	Status int             `json:"status"`
	Result json.RawMessage `json:"result"`
}

// generateSignature generates HMAC SHA256 signature for WebSocket API
func (s *WsRequestedUserDataStreamService) generateSignature(params map[string]interface{}) string {
	// Sort keys alphabetically
	keys := make([]string, 0, len(params))
	for k := range params {
		keys = append(keys, k)
	}
	sort.Strings(keys)

	// Build query string
	var parts []string
	for _, k := range keys {
		parts = append(parts, fmt.Sprintf("%s=%v", k, params[k]))
	}
	queryString := strings.Join(parts, "&")

	// Generate HMAC SHA256
	mac := hmac.New(sha256.New, []byte(s.secretKey))
	mac.Write([]byte(queryString))
	return hex.EncodeToString(mac.Sum(nil))
}

// nextRequestID generates next request ID
func (s *WsRequestedUserDataStreamService) nextRequestID() string {
	id := atomic.AddInt64(&s.requestID, 1)
	return fmt.Sprintf("req-%d", id)
}

// sendAPIRequest sends a signed request to WebSocket API
func (s *WsRequestedUserDataStreamService) sendAPIRequest(method string, extraParams map[string]interface{}, handler func(result json.RawMessage)) error {
	if s.apiConn == nil {
		return fmt.Errorf("websocket API is not connected")
	}

	requestID := s.nextRequestID()
	timestamp := time.Now().UnixMilli()

	// Build params with authentication
	params := map[string]interface{}{
		"apiKey":    s.apiKey,
		"timestamp": timestamp,
	}
	for k, v := range extraParams {
		params[k] = v
	}

	// Generate signature
	params["signature"] = s.generateSignature(params)

	req := wsAPIRequest{
		ID:     requestID,
		Method: method,
		Params: params,
	}

	// Register handler
	s.respHandlersMutex.Lock()
	s.respHandlers[requestID] = wsAPIResponseHandler{
		tag:     method,
		handler: handler,
	}
	s.respHandlersMutex.Unlock()

	// Rate limit
	<-s.rateLimiter.C

	// Send request
	s.apiConnMutex.Lock()
	defer s.apiConnMutex.Unlock()
	return s.apiConn.WriteJSON(req)
}
