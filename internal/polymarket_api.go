package internal

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"time"
)

const (
	PolymarketAPIURL = "https://data-api.polymarket.com/trades"
)

// Trade represents a trade from the Polymarket API
type PolymarketTrade struct {
	ID              string       `json:"id"`
	TakerOrderID    string       `json:"taker_order_id"`
	Market          string       `json:"market"`
	AssetID         string       `json:"asset_id"`
	Side            string       `json:"side"`
	Size            string       `json:"size"`
	FeeRateBps      string       `json:"fee_rate_bps"`
	Price           string       `json:"price"`
	Status          string       `json:"status"`
	MatchTime       string       `json:"match_time"`
	LastUpdate      string       `json:"last_update"`
	Outcome         string       `json:"outcome"`
	MakerAddress    string       `json:"maker_address"`
	Owner           string       `json:"owner"`
	TransactionHash string       `json:"transaction_hash"`
	BucketIndex     int          `json:"bucket_index"`
	MakerOrders     []MakerOrder `json:"maker_orders"`
	Type            string       `json:"type"`
}

// MakerOrder represents a maker order in a trade
type MakerOrder struct {
	OrderID       string `json:"order_id"`
	MakerAddress  string `json:"maker_address"`
	Owner         string `json:"owner"`
	MatchedAmount string `json:"matched_amount"`
	FeeRateBps    string `json:"fee_rate_bps"`
	Price         string `json:"price"`
	AssetID       string `json:"asset_id"`
	Outcome       string `json:"outcome"`
	Side          string `json:"side"`
}

// TradesQueryParams represents query parameters for fetching trades
type TradesQueryParams struct {
	ID     string // id of trade to fetch
	Taker  string // address to get trades for where it is included as a taker
	Maker  string // address to get trades for where it is included as a maker
	Market string // market for which to get the trades (condition ID)
	Before string // unix timestamp representing the cutoff up to which trades that happened before then can be included
	After  string // unix timestamp representing the cutoff for which trades that happened after can be included
}

// PolymarketAPIClient handles API calls to Polymarket
type PolymarketAPIClient struct {
	httpClient *http.Client
	baseURL    string
}

// NewPolymarketAPIClient creates a new Polymarket API client
func NewPolymarketAPIClient() *PolymarketAPIClient {
	return &PolymarketAPIClient{
		httpClient: &http.Client{
			Timeout: 10 * time.Second,
		},
		baseURL: PolymarketAPIURL,
	}
}

// GetTrades fetches trades from the Polymarket API based on query parameters
func (c *PolymarketAPIClient) GetTrades(ctx context.Context, params TradesQueryParams) ([]PolymarketTrade, error) {
	// Build the API URL with query parameters
	apiURL, err := url.Parse(c.baseURL)
	if err != nil {
		return nil, fmt.Errorf("failed to parse API URL: %w", err)
	}

	// Add query parameters
	q := url.Values{}
	if params.ID != "" {
		q.Add("id", params.ID)
	}
	if params.Taker != "" {
		q.Add("taker", params.Taker)
	}
	if params.Maker != "" {
		q.Add("maker", params.Maker)
	}
	if params.Market != "" {
		q.Add("market", params.Market)
	}
	if params.Before != "" {
		q.Add("before", params.Before)
	}
	if params.After != "" {
		q.Add("after", params.After)
	}
	apiURL.RawQuery = q.Encode()

	// Create request
	req, err := http.NewRequestWithContext(ctx, "GET", apiURL.String(), nil)
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}

	// Make the request
	resp, err := c.httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("failed to make request: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("API returned status %d: %s", resp.StatusCode, string(body))
	}

	// Parse response
	var trades []PolymarketTrade
	if err := json.NewDecoder(resp.Body).Decode(&trades); err != nil {
		return nil, fmt.Errorf("failed to decode response: %w", err)
	}

	return trades, nil
}

// GetTradesForAddress fetches trades for a specific address (tries both maker and taker)
func (c *PolymarketAPIClient) GetTradesForAddress(ctx context.Context, address string) ([]PolymarketTrade, error) {
	// Try fetching as taker first
	trades, err := c.GetTrades(ctx, TradesQueryParams{
		Taker: address,
	})
	if err != nil {
		// If taker fails, try as maker
		makerTrades, makerErr := c.GetTrades(ctx, TradesQueryParams{
			Maker: address,
		})
		if makerErr != nil {
			return nil, fmt.Errorf("failed to fetch trades as taker (%v) and maker (%v)", err, makerErr)
		}
		return makerTrades, nil
	}

	// If taker succeeded, also try maker to get all trades
	makerTrades, err := c.GetTrades(ctx, TradesQueryParams{
		Maker: address,
	})
	if err == nil && len(makerTrades) > 0 {
		// Combine both results (deduplication could be added if needed)
		trades = append(trades, makerTrades...)
	}

	return trades, nil
}
