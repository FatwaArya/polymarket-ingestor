package domain

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"sync"
	"time"

	"github.com/FatwaArya/pm-ingest/internal"
	internalkafka "github.com/FatwaArya/pm-ingest/internal/kafka"
	"github.com/twmb/franz-go/pkg/kgo"
)

// ConfidenceService calculates user confidence based on new bets and closed positions
type ConfidenceService struct {
	consumer       *internalkafka.Consumer
	apiClient      *internal.PolymarketAPIClient
	processedUsers map[string]time.Time // Track when we last processed each user
	mu             sync.RWMutex
	minInterval    time.Duration // Minimum time between confidence calculations for same user
}

// ConfidenceResult represents the calculated confidence for a user
type ConfidenceResult struct {
	UserAddress string                     `json:"userAddress"`
	Timestamp   int64                      `json:"timestamp"`
	Prediction  PredictionResult           `json:"prediction"`
	LatestBet   internalkafka.TradeMessage `json:"latestBet,omitempty"`
}

// NewConfidenceService creates a new confidence calculation service
func NewConfidenceService(brokers string, topic string, groupID string) (*ConfidenceService, error) {
	consumer, err := internalkafka.NewConsumer(brokers, topic, groupID)
	if err != nil {
		return nil, fmt.Errorf("failed to create kafka consumer: %w", err)
	}

	apiClient := internal.NewPolymarketAPIClient()

	return &ConfidenceService{
		consumer:       consumer,
		apiClient:      apiClient,
		processedUsers: make(map[string]time.Time),
		minInterval:    5 * time.Minute, // Don't recalculate for same user more than once per 5 minutes
	}, nil
}

// Run starts the confidence service
func (cs *ConfidenceService) Run(ctx context.Context) error {
	return cs.consumer.Run(ctx, cs.handleBet)
}

// handleBet processes a new bet from Kafka and calculates confidence
func (cs *ConfidenceService) handleBet(record *kgo.Record) {
	var tradeMsg internalkafka.TradeMessage
	if err := json.Unmarshal(record.Value, &tradeMsg); err != nil {
		log.Printf("Error unmarshaling trade message: %v", err)
		return
	}

	// Skip if no proxy wallet (can't calculate confidence without user)
	if tradeMsg.ProxyWallet == "" {
		return
	}

	// Check if we should process this user (rate limiting)
	cs.mu.RLock()
	lastProcessed, exists := cs.processedUsers[tradeMsg.ProxyWallet]
	cs.mu.RUnlock()

	if exists && time.Since(lastProcessed) < cs.minInterval {
		return // Skip if processed recently
	}

	// Update processed time
	cs.mu.Lock()
	cs.processedUsers[tradeMsg.ProxyWallet] = time.Now()
	cs.mu.Unlock()

	// Calculate confidence in a goroutine to avoid blocking
	go cs.calculateAndLogConfidence(context.Background(), tradeMsg)
}

// calculateAndLogConfidence fetches closed positions and calculates confidence
func (cs *ConfidenceService) calculateAndLogConfidence(ctx context.Context, bet internalkafka.TradeMessage) {
	userAddress := bet.ProxyWallet

	// Fetch closed positions for the user
	prediction, err := CalculateConfidenceForUser(ctx, cs.apiClient, userAddress, 50)
	if err != nil {
		log.Printf("Error calculating confidence for user %s: %v", userAddress, err)
		return
	}

	// Create confidence result
	result := ConfidenceResult{
		UserAddress: userAddress,
		Timestamp:   time.Now().Unix(),
		Prediction:  prediction,
		LatestBet:   bet,
	}

	// Log the confidence result
	cs.logConfidenceResult(result)
}

// logConfidenceResult logs the confidence calculation result
func (cs *ConfidenceService) logConfidenceResult(result ConfidenceResult) {
	log.Printf("Confidence calculated for user %s:", result.UserAddress)
	log.Printf("  Sample Size: %d", result.Prediction.SampleSize)
	log.Printf("  Win Rate: %.2f%%", result.Prediction.WinRate)
	log.Printf("  Avg Realized PnL: $%.2f", result.Prediction.AvgRealizedPnl)
	log.Printf("  Total Realized PnL: $%.2f", result.Prediction.TotalRealizedPnl)
	log.Printf("  Brier Score: %.4f (lower is better)", result.Prediction.BrierScore)
	log.Printf("  Calibration: %.2f%%", result.Prediction.Calibration)
	log.Printf("  Confidence Interval: ±$%.2f", result.Prediction.ConfidenceInterval)
	log.Printf("  Latest Bet: %s on %s at $%.4f", result.LatestBet.Side, result.LatestBet.Slug, result.LatestBet.Price)
}

// GetConfidenceForUser manually calculates confidence for a specific user
func (cs *ConfidenceService) GetConfidenceForUser(ctx context.Context, userAddress string) (PredictionResult, error) {
	return CalculateConfidenceForUser(ctx, cs.apiClient, userAddress, 50)
}

// Close closes the confidence service
func (cs *ConfidenceService) Close() {
	if cs.consumer != nil {
		cs.consumer.Close()
	}
}
