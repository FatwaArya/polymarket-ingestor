package utils

import (
	"encoding/json"
	"fmt"
)

// ActivityTradePayload represents a trade from the activity topic
type ActivityTradePayload struct {
	ID                 string  `json:"id"`
	Market             string  `json:"market"`
	Asset              string  `json:"asset"`
	Side               string  `json:"side"`  // BUY/SELL
	Price              float64 `json:"price"` // Price in decimal (e.g., 0.55)
	Size               float64 `json:"size"`
	Fee                float64 `json:"fee"`
	Timestamp          int64   `json:"timestamp"`
	TransactionHash    string  `json:"transactionHash,omitempty"`
	Maker              string  `json:"maker,omitempty"`
	Taker              string  `json:"taker,omitempty"`
	MakerOrderID       string  `json:"makerOrderId,omitempty"`
	TakerOrderID       string  `json:"takerOrderId,omitempty"`
	ConditionID        string  `json:"conditionId,omitempty"`
	OutcomeIndex       int     `json:"outcomeIndex,omitempty"`
	QuestionID         string  `json:"questionId,omitempty"`
	MarketSlug         string  `json:"marketSlug,omitempty"`
	EventSlug          string  `json:"eventSlug,omitempty"`
	EventTitle         string  `json:"eventTitle,omitempty"`
	OutcomeTitle       string  `json:"outcomeTitle,omitempty"`
	ProxyWalletAddress string  `json:"proxyWalletAddress,omitempty"`
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

// ParseActivityTrade parses the payload from an activity trades message
func ParseActivityTrade(payload json.RawMessage) (*ActivityTradePayload, error) {
	var trade ActivityTradePayload
	if err := json.Unmarshal(payload, &trade); err != nil {
		return nil, fmt.Errorf("failed to parse activity trade: %w", err)
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
