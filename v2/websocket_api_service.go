package binance

import (
	"crypto/hmac"
	"crypto/sha256"
	"encoding/hex"
	stdjson "encoding/json"
	"fmt"
	"net/http"
	"net/url"
	"sync"
	"sync/atomic"
	"time"

	"github.com/gorilla/websocket"
	jsoniter "github.com/json-iterator/go"
)

var wsAPIJson = jsoniter.ConfigCompatibleWithStandardLibrary

const (
	// WebSocket API endpoint for Spot
	wsAPIBaseURL        = "wss://ws-api.binance.com:443/ws-api/v3"
	wsAPITestnetBaseURL = "wss://ws-api.testnet.binance.vision/ws-api/v3"

	// Ping/pong intervals for connection health monitoring.
	// We send a ping every 30s; if no pong (or any message) arrives within
	// 60s the read deadline fires and ReadMessage returns an error, which
	// surfaces through errHandler and lets the caller reconnect.
	wsAPIPingInterval = 30 * time.Second
	wsAPIReadDeadline = 60 * time.Second
)

// getWsAPIEndpoint returns the WebSocket API endpoint based on testnet flag
func getWsAPIEndpoint() string {
	if UseTestnet {
		return wsAPITestnetBaseURL
	}
	return wsAPIBaseURL
}

// WsAPIClient represents a WebSocket API client for Binance
type WsAPIClient struct {
	apiKey         string
	secretKey      string
	conn           *websocket.Conn
	mu             sync.Mutex
	requestID      int64
	handlers       map[string]chan []byte
	handlerMu      sync.RWMutex
	done           chan struct{}
	closeOnce      sync.Once
	errHandler     ErrHandler
	userDataChan   chan *WsUserDataEvent
	subscriptionID int
	// lastMessageTime is updated on every received message or pong frame,
	// allowing callers to detect stale connections via LastMessageTime().
	lastMessageTime atomic.Value // stores time.Time
}

// NewWsAPIClient creates a new WebSocket API client with built-in ping/pong
// health monitoring. If the connection goes silent for longer than
// wsAPIReadDeadline the read will error out and errHandler is called.
// It connects directly (no proxy) via websocket.DefaultDialer.
func NewWsAPIClient(apiKey, secretKey string, errHandler ErrHandler) (*WsAPIClient, error) {
	return NewWsAPIClientWithProxy(apiKey, secretKey, "", errHandler)
}

// NewWsAPIClientWithProxy is like NewWsAPIClient but routes the WebSocket
// handshake through the given HTTP proxy (format: http://user:pass@host:port).
// This matters in production: opening every account's WS directly from one pod
// IP trips Binance's per-IP connection limits (mass "use of closed network
// connection" disconnects) and the API key IP-whitelist check (ip_forbidden).
// Routing through the rotating proxy spreads connections across exit IPs and
// exits from whitelisted IPs. A WSS handshake goes over an HTTP CONNECT tunnel,
// which keeps a single upstream TCP connection alive for the socket's lifetime,
// so the exit IP stays stable — the proxy does not rotate it mid-connection.
// An empty proxyURL falls back to a direct connection via DefaultDialer.
func NewWsAPIClientWithProxy(apiKey, secretKey, proxyURL string, errHandler ErrHandler) (*WsAPIClient, error) {
	endpoint := getWsAPIEndpoint()

	dialer := websocket.DefaultDialer
	if proxyURL != "" {
		parsed, err := url.Parse(proxyURL)
		if err != nil {
			return nil, fmt.Errorf("failed to parse proxy URL: %w", err)
		}
		dialer = &websocket.Dialer{
			Proxy:            http.ProxyURL(parsed),
			HandshakeTimeout: 45 * time.Second,
		}
	}

	conn, _, err := dialer.Dial(endpoint, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to connect to WebSocket API: %w", err)
	}

	client := &WsAPIClient{
		apiKey:       apiKey,
		secretKey:    secretKey,
		conn:         conn,
		handlers:     make(map[string]chan []byte),
		done:         make(chan struct{}),
		errHandler:   errHandler,
		userDataChan: make(chan *WsUserDataEvent, 500),
	}
	client.lastMessageTime.Store(time.Now())

	// Pong handler: every time the server responds to our ping, extend the
	// read deadline so that ReadMessage keeps working.
	client.conn.SetPongHandler(func(appData string) error {
		client.lastMessageTime.Store(time.Now())
		return client.conn.SetReadDeadline(time.Now().Add(wsAPIReadDeadline))
	})

	// Set initial read deadline — if no data arrives at all within this
	// window (e.g. silent TCP death before the first ping/pong cycle),
	// ReadMessage will return a timeout error.
	_ = client.conn.SetReadDeadline(time.Now().Add(wsAPIReadDeadline))

	go client.readMessages()
	go client.pingLoop()

	return client, nil
}

// UserDataChan returns channel for receiving user data events.
// The channel is closed when the underlying connection is lost or Close() is called.
func (c *WsAPIClient) UserDataChan() <-chan *WsUserDataEvent {
	return c.userDataChan
}

// LastMessageTime returns the time of the last received message or pong.
// Callers can use this as a secondary health check.
func (c *WsAPIClient) LastMessageTime() time.Time {
	if t, ok := c.lastMessageTime.Load().(time.Time); ok {
		return t
	}
	return time.Time{}
}

// Close closes the WebSocket connection. Safe to call multiple times.
func (c *WsAPIClient) Close() error {
	var err error
	c.closeOnce.Do(func() {
		close(c.done)
		err = c.conn.Close()
	})
	return err
}

// pingLoop sends WebSocket ping frames at regular intervals.
// If a ping write fails (connection dead), it reports through errHandler.
func (c *WsAPIClient) pingLoop() {
	ticker := time.NewTicker(wsAPIPingInterval)
	defer ticker.Stop()
	for {
		select {
		case <-c.done:
			return
		case <-ticker.C:
			c.mu.Lock()
			err := c.conn.WriteControl(
				websocket.PingMessage,
				[]byte{},
				time.Now().Add(10*time.Second),
			)
			c.mu.Unlock()
			if err != nil {
				// Don't report errors on deliberate shutdown.
				select {
				case <-c.done:
					return
				default:
				}
				if c.errHandler != nil {
					c.errHandler(fmt.Errorf("ping failed: %w", err))
				}
				return
			}
		}
	}
}

// readMessages reads messages from WebSocket and dispatches to handlers.
// When it exits (error or shutdown) it closes userDataChan so downstream
// readers unblock immediately instead of hanging forever.
func (c *WsAPIClient) readMessages() {
	defer close(c.userDataChan)
	for {
		select {
		case <-c.done:
			return
		default:
			_, message, err := c.conn.ReadMessage()
			if err != nil {
				// Don't fire errHandler on deliberate Close().
				select {
				case <-c.done:
					return
				default:
				}
				if c.errHandler != nil {
					c.errHandler(err)
				}
				return
			}

			// Every received message proves the connection is alive.
			c.lastMessageTime.Store(time.Now())
			_ = c.conn.SetReadDeadline(time.Now().Add(wsAPIReadDeadline))

			// First check if it's a subscription event (has "subscriptionId" and "event" fields)
			// Format: {"subscriptionId":0,"event":{"e":"executionReport",...}}
			var subscriptionEventCheck struct {
				SubscriptionID *int               `json:"subscriptionId"`
				Event          stdjson.RawMessage `json:"event"`
			}
			unmarshalErr := wsAPIJson.Unmarshal(message, &subscriptionEventCheck)
			hasSubscriptionID := subscriptionEventCheck.SubscriptionID != nil
			hasEvent := len(subscriptionEventCheck.Event) > 2
			if unmarshalErr == nil && hasSubscriptionID && hasEvent {
				// This is a subscription event, parse it
				// Use standard json for parsing to avoid json-iterator issues with mixed types
				var eventType struct {
					Event string `json:"e"`
				}
				stdjson.Unmarshal(subscriptionEventCheck.Event, &eventType)
				if eventType.Event != "" {
					// Create event and set type manually (avoid full unmarshal due to field conflicts)
					event := &WsUserDataEvent{
						Event: UserDataEventType(eventType.Event),
					}

					// Parse time field
					var timeField struct {
						Time int64 `json:"E"`
					}
					stdjson.Unmarshal(subscriptionEventCheck.Event, &timeField)
					event.Time = timeField.Time

					// Handle specific event types
					switch UserDataEventType(eventType.Event) {
					case UserDataEventTypeBalanceUpdate:
						stdjson.Unmarshal(subscriptionEventCheck.Event, &event.BalanceUpdate)
					case UserDataEventTypeExecutionReport:
						stdjson.Unmarshal(subscriptionEventCheck.Event, &event.OrderUpdate)
					case UserDataEventTypeOutboundAccountPosition:
						// Spot outboundAccountPosition has 'B' field for balances
						var temp struct {
							Balances []WsAccountUpdate `json:"B"`
						}
						stdjson.Unmarshal(subscriptionEventCheck.Event, &temp)
						event.AccountUpdate = temp.Balances
					case UserDataEventTypeListStatus:
						stdjson.Unmarshal(subscriptionEventCheck.Event, &event.OCOUpdate)
					}

					select {
					case c.userDataChan <- event:
					default:
						// Channel full, skip event
					}
				}
				continue
			}

			// Try to parse as API response (has "id" field)
			var resp struct {
				ID string `json:"id"`
			}
			if err := wsAPIJson.Unmarshal(message, &resp); err == nil && resp.ID != "" {
				c.handlerMu.RLock()
				if ch, ok := c.handlers[resp.ID]; ok {
					select {
					case ch <- message:
					default:
					}
				}
				c.handlerMu.RUnlock()
				continue
			}

			// Fallback: Try to parse as direct user data event (has "e" field for event type)
			var eventType struct {
				Event string `json:"e"`
			}
			if err := wsAPIJson.Unmarshal(message, &eventType); err == nil && eventType.Event != "" {
				event := new(WsUserDataEvent)
				if err := wsAPIJson.Unmarshal(message, event); err != nil {
					if c.errHandler != nil {
						c.errHandler(err)
					}
					continue
				}

				// Handle specific event types
				switch UserDataEventType(eventType.Event) {
				case UserDataEventTypeBalanceUpdate:
					wsAPIJson.Unmarshal(message, &event.BalanceUpdate)
				case UserDataEventTypeExecutionReport:
					wsAPIJson.Unmarshal(message, &event.OrderUpdate)
				case UserDataEventTypeOutboundAccountPosition:
					// Spot outboundAccountPosition has 'B' field for balances
					var temp struct {
						Balances []WsAccountUpdate `json:"B"`
					}
					wsAPIJson.Unmarshal(message, &temp)
					event.AccountUpdate = temp.Balances
				case UserDataEventTypeListStatus:
					wsAPIJson.Unmarshal(message, &event.OCOUpdate)
				}

				select {
				case c.userDataChan <- event:
				default:
					// Channel full, skip event
				}
			}
		}
	}
}

// generateSignature generates HMAC SHA256 signature
func (c *WsAPIClient) generateSignature(params string) string {
	mac := hmac.New(sha256.New, []byte(c.secretKey))
	mac.Write([]byte(params))
	return hex.EncodeToString(mac.Sum(nil))
}

// nextRequestID generates next request ID
func (c *WsAPIClient) nextRequestID() string {
	id := atomic.AddInt64(&c.requestID, 1)
	return fmt.Sprintf("req-%d", id)
}

// WsAPIRequest represents a WebSocket API request
type WsAPIRequest struct {
	ID     string                 `json:"id"`
	Method string                 `json:"method"`
	Params map[string]interface{} `json:"params,omitempty"`
}

// WsAPIResponse represents a WebSocket API response
type WsAPIResponse struct {
	ID     string             `json:"id"`
	Status int                `json:"status"`
	Result stdjson.RawMessage `json:"result,omitempty"`
	Error  *WsAPIError        `json:"error,omitempty"`
}

// WsAPIError represents a WebSocket API error
type WsAPIError struct {
	Code int    `json:"code"`
	Msg  string `json:"msg"`
}

func (e *WsAPIError) Error() string {
	return fmt.Sprintf("<APIError> code=%d, msg=%s", e.Code, e.Msg)
}

// SubscribeUserDataStream subscribes to user data stream using signature method
// This is the new way to get user data updates after Binance removed REST API endpoints
func (c *WsAPIClient) SubscribeUserDataStream() (subscriptionID int, err error) {
	requestID := c.nextRequestID()
	timestamp := time.Now().UnixMilli()

	// Build signature payload
	params := fmt.Sprintf("apiKey=%s&timestamp=%d", c.apiKey, timestamp)
	signature := c.generateSignature(params)

	req := WsAPIRequest{
		ID:     requestID,
		Method: "userDataStream.subscribe.signature",
		Params: map[string]interface{}{
			"apiKey":    c.apiKey,
			"timestamp": timestamp,
			"signature": signature,
		},
	}

	// Create response channel
	respChan := make(chan []byte, 1)
	c.handlerMu.Lock()
	c.handlers[requestID] = respChan
	c.handlerMu.Unlock()

	defer func() {
		c.handlerMu.Lock()
		delete(c.handlers, requestID)
		c.handlerMu.Unlock()
	}()

	// Send request
	c.mu.Lock()
	err = c.conn.WriteJSON(req)
	c.mu.Unlock()
	if err != nil {
		return 0, fmt.Errorf("failed to send subscribe request: %w", err)
	}

	// Wait for response with timeout
	select {
	case respData := <-respChan:
		var resp WsAPIResponse
		if err := wsAPIJson.Unmarshal(respData, &resp); err != nil {
			return 0, fmt.Errorf("failed to parse response: %w", err)
		}

		if resp.Error != nil {
			return 0, resp.Error
		}

		if resp.Status != 200 {
			return 0, fmt.Errorf("unexpected status: %d", resp.Status)
		}

		var result struct {
			SubscriptionID int `json:"subscriptionId"`
		}
		if err := wsAPIJson.Unmarshal(resp.Result, &result); err != nil {
			return 0, fmt.Errorf("failed to parse result: %w", err)
		}

		c.subscriptionID = result.SubscriptionID
		return result.SubscriptionID, nil

	case <-time.After(30 * time.Second):
		return 0, fmt.Errorf("timeout waiting for subscribe response")
	}
}

// GetSubscriptionID returns the current subscription ID
func (c *WsAPIClient) GetSubscriptionID() int {
	return c.subscriptionID
}

// UnsubscribeUserDataStream unsubscribes from user data stream
func (c *WsAPIClient) UnsubscribeUserDataStream(subscriptionID int) error {
	requestID := c.nextRequestID()

	req := WsAPIRequest{
		ID:     requestID,
		Method: "userDataStream.unsubscribe",
		Params: map[string]interface{}{
			"subscriptionId": subscriptionID,
		},
	}

	// Create response channel
	respChan := make(chan []byte, 1)
	c.handlerMu.Lock()
	c.handlers[requestID] = respChan
	c.handlerMu.Unlock()

	defer func() {
		c.handlerMu.Lock()
		delete(c.handlers, requestID)
		c.handlerMu.Unlock()
	}()

	// Send request
	c.mu.Lock()
	err := c.conn.WriteJSON(req)
	c.mu.Unlock()
	if err != nil {
		return fmt.Errorf("failed to send unsubscribe request: %w", err)
	}

	// Wait for response with timeout
	select {
	case respData := <-respChan:
		var resp WsAPIResponse
		if err := wsAPIJson.Unmarshal(respData, &resp); err != nil {
			return fmt.Errorf("failed to parse response: %w", err)
		}

		if resp.Error != nil {
			return resp.Error
		}

		return nil

	case <-time.After(30 * time.Second):
		return fmt.Errorf("timeout waiting for unsubscribe response")
	}
}
