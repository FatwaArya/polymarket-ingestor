package internal

import (
	"encoding/json"
	"log"
	"sync"
	"time"

	"github.com/gorilla/websocket"
)

const (
	MarketChannel = "market"
	UserChannel   = "user"
	WsURL         = "wss://ws-subscriptions-clob.polymarket.com"
	PingInterval  = 10 * time.Second
)

// Auth holds the authentication credentials for the WebSocket connection
type Auth struct {
	APIKey     string `json:"apiKey"`
	Secret     string `json:"secret"`
	Passphrase string `json:"passphrase"`
}

// WebSocketAuth is the auth payload for WebSocket subscription (uses raw credentials per docs)
type WebSocketAuth struct {
	APIKey     string `json:"apiKey"`
	Secret     string `json:"secret"`
	Passphrase string `json:"passphrase"`
}

// MarketSubscription is the message sent to subscribe to market channel
type MarketSubscription struct {
	AssetsIDs []string `json:"assets_ids"`
	Type      string   `json:"type"`
}

// UserSubscription is the message sent to subscribe to user channel
type UserSubscription struct {
	Markets []string       `json:"markets"`
	Type    string         `json:"type"`
	Auth    *WebSocketAuth `json:"auth"`
}

// MessageCallback is a function type for handling incoming messages
type MessageCallback func(message []byte)

// WebSocketOrderBook manages the WebSocket connection to Polymarket
type WebSocketOrderBook struct {
	channelType     string
	url             string
	data            []string
	auth            *Auth
	messageCallback MessageCallback
	verbose         bool
	conn            *websocket.Conn
	orderbooks      map[string]interface{}
	mu              sync.RWMutex
	done            chan struct{}
}

// NewWebSocketOrderBook creates a new WebSocket connection handler
func NewWebSocketOrderBook(
	channelType string,
	url string,
	data []string,
	auth *Auth,
	messageCallback MessageCallback,
	verbose bool,
) *WebSocketOrderBook {
	return &WebSocketOrderBook{
		channelType:     channelType,
		url:             url,
		data:            data,
		auth:            auth,
		messageCallback: messageCallback,
		verbose:         verbose,
		orderbooks:      make(map[string]interface{}),
		done:            make(chan struct{}),
	}
}

// Connect establishes the WebSocket connection
func (w *WebSocketOrderBook) Connect() error {
	fullURL := w.url + "/ws/" + w.channelType

	if w.verbose {
		log.Printf("Connecting to %s", fullURL)
	}

	conn, _, err := websocket.DefaultDialer.Dial(fullURL, nil)
	if err != nil {
		return err
	}
	w.conn = conn

	// Subscribe based on channel type
	if err := w.subscribe(); err != nil {
		return err
	}

	return nil
}

// subscribe sends the subscription message based on channel type
func (w *WebSocketOrderBook) subscribe() error {
	var msg []byte
	var err error

	switch w.channelType {
	case MarketChannel:
		subscription := MarketSubscription{
			AssetsIDs: w.data,
			Type:      MarketChannel,
		}
		msg, err = json.Marshal(subscription)
	case UserChannel:
		if w.auth == nil {
			log.Fatal("Auth required for user channel")
		}
		subscription := UserSubscription{
			Markets: []string{},
			Type:    UserChannel,
			Auth: &WebSocketAuth{
				APIKey:     w.auth.APIKey,
				Secret:     w.auth.Secret,
				Passphrase: w.auth.Passphrase,
			},
		}
		// Only include markets if there are specific ones to subscribe to
		if len(w.data) > 0 {
			subscription.Markets = w.data
		}
		msg, err = json.Marshal(subscription)
	default:
		log.Fatalf("Unknown channel type: %s", w.channelType)
	}

	if err != nil {
		return err
	}

	if w.verbose {
		log.Printf("Sending subscription: %s", string(msg))
	}

	return w.conn.WriteMessage(websocket.TextMessage, msg)
}

// startPing sends PING messages at regular intervals
func (w *WebSocketOrderBook) startPing() {
	ticker := time.NewTicker(PingInterval)
	defer ticker.Stop()

	for {
		select {
		case <-ticker.C:
			w.mu.Lock()
			if w.conn != nil {
				if err := w.conn.WriteMessage(websocket.TextMessage, []byte("PING")); err != nil {
					log.Printf("Ping error: %v", err)
				} else if w.verbose {
					log.Println("Sent PING")
				}
			}
			w.mu.Unlock()
		case <-w.done:
			return
		}
	}
}

// Run starts the WebSocket connection and message handling loop
func (w *WebSocketOrderBook) Run() error {
	if err := w.Connect(); err != nil {
		return err
	}
	defer w.Close()

	// Start ping goroutine
	go w.startPing()

	// Message reading loop
	for {
		_, message, err := w.conn.ReadMessage()
		if err != nil {
			if websocket.IsCloseError(err, websocket.CloseNormalClosure, websocket.CloseGoingAway) {
				log.Println("Connection closed normally")
				return nil
			}
			log.Printf("Read error: %v", err)
			return err
		}

		if w.verbose {
			log.Printf("Received: %s", string(message))
		}

		if w.messageCallback != nil {
			w.messageCallback(message)
		}
	}
}

// Close gracefully closes the WebSocket connection
func (w *WebSocketOrderBook) Close() {
	close(w.done)
	w.mu.Lock()
	defer w.mu.Unlock()
	if w.conn != nil {
		w.conn.Close()
		w.conn = nil
	}
}
