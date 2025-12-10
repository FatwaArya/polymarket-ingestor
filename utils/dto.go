package utils

import (
	"encoding/json"
	"fmt"
)

// MakerOrder represents a maker order in a trade
type MakerOrder struct {
	AssetID       string `json:"asset_id"`
	MatchedAmount string `json:"matched_amount"`
	OrderID       string `json:"order_id"`
	Outcome       string `json:"outcome"`
	Owner         string `json:"owner"`
	Price         string `json:"price"`
}

// TradeMessage is emitted when orders are matched or trade status changes
// Statuses: MATCHED, MINED, CONFIRMED, RETRYING, FAILED
type TradeMessage struct {
	AssetID      string       `json:"asset_id"`
	EventType    string       `json:"event_type"` // "trade"
	ID           string       `json:"id"`
	LastUpdate   string       `json:"last_update"`
	MakerOrders  []MakerOrder `json:"maker_orders"`
	Market       string       `json:"market"` // condition ID
	MatchTime    string       `json:"matchtime"`
	Outcome      string       `json:"outcome"`
	Owner        string       `json:"owner"`
	Price        string       `json:"price"`
	Side         string       `json:"side"` // BUY/SELL
	Size         string       `json:"size"`
	Status       string       `json:"status"` // MATCHED, MINED, CONFIRMED, RETRYING, FAILED
	TakerOrderID string       `json:"taker_order_id"`
	Timestamp    string       `json:"timestamp"`
	TradeOwner   string       `json:"trade_owner"`
	Type         string       `json:"type"` // "TRADE"
}

// OrderMessage is emitted for order placements, updates, and cancellations
// Types: PLACEMENT, UPDATE, CANCELLATION
type OrderMessage struct {
	AssetID         string   `json:"asset_id"`
	AssociateTrades []string `json:"associate_trades"`
	EventType       string   `json:"event_type"` // "order"
	ID              string   `json:"id"`
	Market          string   `json:"market"` // condition ID
	OrderOwner      string   `json:"order_owner"`
	OriginalSize    string   `json:"original_size"`
	Outcome         string   `json:"outcome"`
	Owner           string   `json:"owner"`
	Price           string   `json:"price"`
	Side            string   `json:"side"` // BUY/SELL
	SizeMatched     string   `json:"size_matched"`
	Timestamp       string   `json:"timestamp"`
	Type            string   `json:"type"` // PLACEMENT, UPDATE, CANCELLATION
}

// UserChannelMessage is a generic wrapper to determine message type
type UserChannelMessage struct {
	EventType string `json:"event_type"` // "trade" or "order"
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

// Event type constants
const (
	EventTypeTrade = "trade"
	EventTypeOrder = "order"
)

// ParseUserMessage parses a raw WebSocket message into the appropriate struct
func ParseUserMessage(data []byte) (interface{}, error) {
	// First, determine the event type
	var wrapper UserChannelMessage
	if err := json.Unmarshal(data, &wrapper); err != nil {
		return nil, err
	}

	switch wrapper.EventType {
	case EventTypeTrade:
		var trade TradeMessage
		if err := json.Unmarshal(data, &trade); err != nil {
			return nil, err
		}
		return &trade, nil
	case EventTypeOrder:
		var order OrderMessage
		if err := json.Unmarshal(data, &order); err != nil {
			return nil, err
		}
		return &order, nil
	default:
		return nil, fmt.Errorf("unknown event type: %s", wrapper.EventType)
	}
}
