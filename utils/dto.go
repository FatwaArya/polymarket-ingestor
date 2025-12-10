package utils

import (
	"encoding/json"
	"fmt"
)

// IncomingMessage represents the wrapper structure for WebSocket messages
type IncomingMessage struct {
	ConnectionID string          `json:"connection_id"`
	Payload      json.RawMessage `json:"payload"`
	Timestamp    int64           `json:"timestamp"`
	Topic        string          `json:"topic"`
	Type         string          `json:"type"`
}

// ActivityTradePayload represents a trade from the activity topic
type ActivityTradePayload struct {
	ID                 string  `json:"id,omitempty"`
	Market             string  `json:"market,omitempty"`
	Asset              string  `json:"asset"`
	Side               string  `json:"side"`  // BUY/SELL
	Price              float64 `json:"price"` // Price in decimal (e.g., 0.55)
	Size               float64 `json:"size"`
	Fee                float64 `json:"fee,omitempty"`
	Timestamp          int64   `json:"timestamp"`
	TransactionHash    string  `json:"transactionHash,omitempty"`
	Maker              string  `json:"maker,omitempty"`
	Taker              string  `json:"taker,omitempty"`
	MakerOrderID       string  `json:"makerOrderId,omitempty"`
	TakerOrderID       string  `json:"takerOrderId,omitempty"`
	ConditionID        string  `json:"conditionId,omitempty"`
	OutcomeIndex       int     `json:"outcomeIndex,omitempty"`
	QuestionID         string  `json:"questionId,omitempty"`
	MarketSlug         string  `json:"slug,omitempty"` // JSON field is "slug"
	EventSlug          string  `json:"eventSlug,omitempty"`
	EventTitle         string  `json:"title,omitempty"`       // JSON field is "title"
	OutcomeTitle       string  `json:"outcome,omitempty"`     // JSON field is "outcome"
	ProxyWalletAddress string  `json:"proxyWallet,omitempty"` // JSON field is "proxyWallet"
	// Additional fields from the actual payload
	Name         string `json:"name,omitempty"`
	Pseudonym    string `json:"pseudonym,omitempty"`
	Bio          string `json:"bio,omitempty"`
	Icon         string `json:"icon,omitempty"`
	ProfileImage string `json:"profileImage,omitempty"`
}

// ClobUserOrder represents an order update from clob_user topic
type ClobUserOrder struct {
	ID              string   `json:"id"`
	Market          string   `json:"market"`
	AssetID         string   `json:"asset_id"`
	Side            string   `json:"side"`
	Price           string   `json:"price"`
	OriginalSize    string   `json:"original_size"`
	SizeMatched     string   `json:"size_matched"`
	Type            string   `json:"type"` // PLACEMENT, UPDATE, CANCELLATION
	Outcome         string   `json:"outcome"`
	Owner           string   `json:"owner"`
	Timestamp       string   `json:"timestamp"`
	AssociateTrades []string `json:"associate_trades,omitempty"`
}

// ClobUserTrade represents a trade update from clob_user topic
type ClobUserTrade struct {
	ID           string       `json:"id"`
	Market       string       `json:"market"`
	AssetID      string       `json:"asset_id"`
	Side         string       `json:"side"`
	Price        string       `json:"price"`
	Size         string       `json:"size"`
	Status       string       `json:"status"` // MATCHED, MINED, CONFIRMED, RETRYING, FAILED
	Outcome      string       `json:"outcome"`
	Owner        string       `json:"owner"`
	TakerOrderID string       `json:"taker_order_id"`
	Timestamp    string       `json:"timestamp"`
	MatchTime    string       `json:"matchtime,omitempty"`
	LastUpdate   string       `json:"last_update,omitempty"`
	MakerOrders  []MakerOrder `json:"maker_orders,omitempty"`
}

// MakerOrder represents a maker order in a trade
type MakerOrder struct {
	AssetID       string `json:"asset_id"`
	MatchedAmount string `json:"matched_amount"`
	OrderID       string `json:"order_id"`
	Outcome       string `json:"outcome"`
	Owner         string `json:"owner"`
	Price         string `json:"price"`
}

// Trade status constants
const (
	TradeStatusMatched   = "MATCHED"
	TradeStatusMined     = "MINED"
	TradeStatusConfirmed = "CONFIRMED"
	TradeStatusRetrying  = "RETRYING"
	TradeStatusFailed    = "FAILED"
)

// Order type constants
const (
	OrderTypePlacement    = "PLACEMENT"
	OrderTypeUpdate       = "UPDATE"
	OrderTypeCancellation = "CANCELLATION"
)

// Side constants
const (
	SideBuy  = "BUY"
	SideSell = "SELL"
)

// Topic constants
const (
	TopicActivity = "activity"
	TopicClobUser = "clob_user"
	TopicComments = "comments"
)

// Type constants
const (
	TypeTrades = "trades"
	TypeOrders = "orders"
)

// ErrSkipMessage is returned when a message should be skipped (not a trade)
var ErrSkipMessage = fmt.Errorf("skip message")

// ParseActivityTrade parses the full WebSocket message and extracts the trade payload
func ParseActivityTrade(message []byte) (*ActivityTradePayload, error) {
	// Skip empty messages
	if len(message) == 0 {
		return nil, ErrSkipMessage
	}

	// Skip non-JSON messages (like "pong")
	if message[0] != '{' {
		return nil, ErrSkipMessage
	}

	// First, parse the wrapper message
	var incoming IncomingMessage
	if err := json.Unmarshal(message, &incoming); err != nil {
		return nil, fmt.Errorf("failed to parse incoming message: %w", err)
	}

	// Skip non-trade messages silently
	if incoming.Topic != TopicActivity || incoming.Type != TypeTrades {
		return nil, ErrSkipMessage
	}

	// Parse the actual trade payload
	var trade ActivityTradePayload
	if err := json.Unmarshal(incoming.Payload, &trade); err != nil {
		return nil, fmt.Errorf("failed to parse activity trade payload: %w", err)
	}

	return &trade, nil
}

// ParseClobUserOrder parses an order message from clob_user topic
func ParseClobUserOrder(payload json.RawMessage) (*ClobUserOrder, error) {
	var order ClobUserOrder
	if err := json.Unmarshal(payload, &order); err != nil {
		return nil, fmt.Errorf("failed to parse clob_user order: %w", err)
	}
	return &order, nil
}

// ParseClobUserTrade parses a trade message from clob_user topic
func ParseClobUserTrade(payload json.RawMessage) (*ClobUserTrade, error) {
	var trade ClobUserTrade
	if err := json.Unmarshal(payload, &trade); err != nil {
		return nil, fmt.Errorf("failed to parse clob_user trade: %w", err)
	}
	return &trade, nil
}
