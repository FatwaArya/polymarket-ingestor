package internal

import (
	"encoding/json"
	"log"
	"sync"
	"sync/atomic"
	"time"

	"github.com/gorilla/websocket"
)

const (
	// WebSocket URL for Polymarket real-time data
	WsURL        = "wss://ws-live-data.polymarket.com"
	PingInterval = 5 * time.Second
)

// Topic constants
const (
	TopicActivity = "activity"
	TopicComments = "comments"
	TopicClobUser = "clob_user"
)

// Type constants
const (
	TypeTrades = "trades"
	TypeAll    = "*"
)

// Auth holds the authentication credentials for private topics
type Auth struct {
	APIKey     string `json:"key"`
	Secret     string `json:"secret"`
	Passphrase string `json:"passphrase"`
}

// ClobAuth is the auth structure for clob_user subscriptions
type ClobAuth struct {
	Key        string `json:"key"`
	Secret     string `json:"secret"`
	Passphrase string `json:"passphrase"`
}

// Subscription represents a single topic subscription
type Subscription struct {
	Topic    string    `json:"topic"`
	Type     string    `json:"type"`
	ClobAuth *ClobAuth `json:"clob_auth,omitempty"`
	Filters  string    `json:"filters,omitempty"`
}

// SubscriptionMessage is the message sent to subscribe/unsubscribe
type SubscriptionMessage struct {
	Action        string         `json:"action"`
	Subscriptions []Subscription `json:"subscriptions"`
}

// IncomingMessage represents the structure of messages received from the WebSocket
type IncomingMessage struct {
	Topic        string          `json:"topic"`
	Type         string          `json:"type"`
	Timestamp    int64           `json:"timestamp"`
	ConnectionID string          `json:"connection_id"`
	Payload      json.RawMessage `json:"payload"`
}

// MessageCallback is a function type for handling incoming messages
type MessageCallback func(message []byte)

// WebSocketClient manages the WebSocket connection to Polymarket
type WebSocketClient struct {
	url             string
	subscriptions   []Subscription
	messageCallback MessageCallback
	verbose         bool
	conn            *websocket.Conn
	mu              sync.RWMutex
	done            chan struct{}
	closed          atomic.Bool
}

// NewWebSocketClient creates a new WebSocket connection handler
func NewWebSocketClient(
	subscriptions []Subscription,
	messageCallback MessageCallback,
	verbose bool,
) *WebSocketClient {
	return &WebSocketClient{
		url:             WsURL,
		subscriptions:   subscriptions,
		messageCallback: messageCallback,
		verbose:         verbose,
		done:            make(chan struct{}),
	}
}

// Connect establishes the WebSocket connection
func (w *WebSocketClient) Connect() error {
	if w.verbose {
		log.Printf("Connecting to %s", w.url)
	}

	conn, _, err := websocket.DefaultDialer.Dial(w.url, nil)
	if err != nil {
		return err
	}
	w.mu.Lock()
	w.conn = conn
	w.mu.Unlock()

	return nil
}

// Subscribe sends the subscription message
func (w *WebSocketClient) Subscribe() error {
	msg := SubscriptionMessage{
		Action:        "subscribe",
		Subscriptions: w.subscriptions,
	}

	data, err := json.Marshal(msg)
	if err != nil {
		return err
	}

	if w.verbose {
		log.Printf("Sending subscription: %s", string(data))
	}

	w.mu.Lock()
	defer w.mu.Unlock()
	return w.conn.WriteMessage(websocket.TextMessage, data)
}

// Unsubscribe sends the unsubscribe message for specific subscriptions
func (w *WebSocketClient) Unsubscribe(subscriptions []Subscription) error {
	msg := SubscriptionMessage{
		Action:        "unsubscribe",
		Subscriptions: subscriptions,
	}

	data, err := json.Marshal(msg)
	if err != nil {
		return err
	}

	if w.verbose {
		log.Printf("Sending unsubscribe: %s", string(data))
	}

	w.mu.Lock()
	defer w.mu.Unlock()
	return w.conn.WriteMessage(websocket.TextMessage, data)
}

// startPing sends ping messages at regular intervals to keep connection alive
func (w *WebSocketClient) startPing() {
	ticker := time.NewTicker(PingInterval)
	defer ticker.Stop()

	for {
		select {
		case <-ticker.C:
			w.mu.Lock()
			if w.conn != nil {
				// Send lowercase "ping" as plain text per Polymarket spec
				if err := w.conn.WriteMessage(websocket.TextMessage, []byte("ping")); err != nil {
					log.Printf("Ping error: %v", err)
				} else if w.verbose {
					log.Println("Sent ping")
				}
			}
			w.mu.Unlock()
		case <-w.done:
			return
		}
	}
}

// Run starts the WebSocket connection and message handling loop
func (w *WebSocketClient) Run() error {
	if err := w.Connect(); err != nil {
		return err
	}

	// Start ping goroutine
	go w.startPing()

	// Subscribe to topics
	if err := w.Subscribe(); err != nil {
		w.Close()
		return err
	}

	// Message reading loop
	for {
		select {
		case <-w.done:
			return nil
		default:
			_, message, err := w.conn.ReadMessage()
			if err != nil {
				// Check if we're shutting down
				if w.closed.Load() {
					return nil
				}
				if websocket.IsCloseError(err, websocket.CloseNormalClosure, websocket.CloseGoingAway) {
					log.Println("Connection closed normally")
					return nil
				}
				log.Printf("Read error: %v", err)
				return err
			}

			// Check if it's a pong response (plain text)
			if string(message) == "pong" {
				if w.verbose {
					log.Println("Received pong")
				}
				continue
			}

			if w.verbose {
				log.Printf("Received: %s", string(message))
			}

			// Pass raw message to callback
			if w.messageCallback != nil {
				w.messageCallback(message)
			}
		}
	}
}

// Close gracefully closes the WebSocket connection
func (w *WebSocketClient) Close() {
	// Use atomic to prevent double-close panic
	if w.closed.Swap(true) {
		return // Already closed
	}

	close(w.done)
	w.mu.Lock()
	defer w.mu.Unlock()
	if w.conn != nil {
		w.conn.Close()
		w.conn = nil
	}
}

// Helper function to create an activity trades subscription
func NewActivityTradesSubscription() Subscription {
	return Subscription{
		Topic: TopicActivity,
		Type:  TypeTrades,
	}
}

// Helper function to create an activity subscription for all types
func NewActivityAllSubscription() Subscription {
	return Subscription{
		Topic: TopicActivity,
		Type:  TypeAll,
	}
}

func NewCommentsSubscription() Subscription {
	return Subscription{
		Topic: TopicComments,
		Type:  TypeAll,
	}
}

// Helper function to create a clob_user subscription with auth
func NewClobUserSubscription(auth *Auth) Subscription {
	return Subscription{
		Topic: TopicClobUser,
		Type:  TypeAll,
		ClobAuth: &ClobAuth{
			Key:        auth.APIKey,
			Secret:     auth.Secret,
			Passphrase: auth.Passphrase,
		},
	}
}
