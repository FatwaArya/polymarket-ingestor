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
	PolymarketAPIURL = "https://data-api.polymarket.com/closed-positions"
)

// ClosedPosition represents a closed position from the Polymarket API
type ClosedPosition struct {
	ProxyWallet     string  `json:"proxyWallet"`
	Asset           string  `json:"asset"`
	ConditionID     string  `json:"conditionId"`
	AvgPrice        float64 `json:"avgPrice"`
	TotalBought     float64 `json:"totalBought"`
	RealizedPnl     float64 `json:"realizedPnl"`
	CurPrice        float64 `json:"curPrice"`
	Timestamp       int64   `json:"timestamp"`
	Title           string  `json:"title"`
	Slug            string  `json:"slug"`
	Icon            string  `json:"icon"`
	EventSlug       string  `json:"eventSlug"`
	Outcome         string  `json:"outcome"`
	OutcomeIndex    int     `json:"outcomeIndex"`
	OppositeOutcome string  `json:"oppositeOutcome"`
	OppositeAsset   string  `json:"oppositeAsset"`
	EndDate         string  `json:"endDate"`
}

// ClosedPositionsQueryParams represents query parameters for fetching closed positions
type ClosedPositionsQueryParams struct {
	User          string   // The address of the user (required)
	Market        []string // The conditionId of the market(s). Supports multiple values
	Title         string   // Filter by market title
	EventID       []int    // The event id(s). Supports multiple values. Cannot be used with Market param
	Limit         int      // The max number of positions to return (default: 10, max: 50)
	Offset        int      // The starting index for pagination (default: 0)
	SortBy        string   // Sort criteria: REALIZEDPNL, TITLE, PRICE, AVGPRICE, TIMESTAMP (default: REALIZEDPNL)
	SortDirection string   // Sort direction: ASC, DESC (default: DESC)
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

// GetClosedPositions fetches closed positions from the Polymarket API based on query parameters
func (c *PolymarketAPIClient) GetClosedPositions(ctx context.Context, params ClosedPositionsQueryParams) ([]ClosedPosition, error) {
	// Build the API URL with query parameters
	apiURL, err := url.Parse(c.baseURL)
	if err != nil {
		return nil, fmt.Errorf("failed to parse API URL: %w", err)
	}

	// Add query parameters
	q := url.Values{}
	if params.User == "" {
		return nil, fmt.Errorf("user parameter is required")
	}
	q.Add("user", params.User)

	if len(params.Market) > 0 {
		// Support multiple market values (comma-separated)
		for _, market := range params.Market {
			q.Add("market", market)
		}
	}

	if params.Title != "" {
		q.Add("title", params.Title)
	}

	if len(params.EventID) > 0 {
		// Support multiple eventId values (comma-separated)
		for _, eventID := range params.EventID {
			q.Add("eventId", fmt.Sprintf("%d", eventID))
		}
	}

	if params.Limit > 0 {
		q.Add("limit", fmt.Sprintf("%d", params.Limit))
	}

	if params.Offset > 0 {
		q.Add("offset", fmt.Sprintf("%d", params.Offset))
	}

	if params.SortBy != "" {
		q.Add("sortBy", params.SortBy)
	}

	if params.SortDirection != "" {
		q.Add("sortDirection", params.SortDirection)
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
	var positions []ClosedPosition
	if err := json.NewDecoder(resp.Body).Decode(&positions); err != nil {
		return nil, fmt.Errorf("failed to decode response: %w", err)
	}

	return positions, nil
}
