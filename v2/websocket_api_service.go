package binance

import (
	"crypto/hmac"
	"crypto/sha256"
	"encoding/hex"
	stdjson "encoding/json"
	"fmt"
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
	wsAPITestnetBaseURL = "wss://testnet.binance.vision/ws-api/v3"
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
	errHandler     ErrHandler
	userDataChan   chan *WsUserDataEvent
	subscriptionID int
}

// NewWsAPIClient creates a new WebSocket API client
func NewWsAPIClient(apiKey, secretKey string, errHandler ErrHandler) (*WsAPIClient, error) {
	endpoint := getWsAPIEndpoint()
	conn, _, err := websocket.DefaultDialer.Dial(endpoint, nil)
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

	go client.readMessages()

	return client, nil
}

// UserDataChan returns channel for receiving user data events
func (c *WsAPIClient) UserDataChan() <-chan *WsUserDataEvent {
	return c.userDataChan
}

// Close closes the WebSocket connection
func (c *WsAPIClient) Close() error {
	close(c.done)
	return c.conn.Close()
}

// readMessages reads messages from WebSocket and dispatches to handlers
func (c *WsAPIClient) readMessages() {
	for {
		select {
		case <-c.done:
			return
		default:
			_, message, err := c.conn.ReadMessage()
			if err != nil {
				if c.errHandler != nil {
					c.errHandler(err)
				}
				return
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

			// Try to parse as user data event (has "e" field for event type)
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
