package domain

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/FatwaArya/pm-ingest/config"
	internalqdb "github.com/FatwaArya/pm-ingest/internal"
	internalkafka "github.com/FatwaArya/pm-ingest/internal/kafka"
	"github.com/twmb/franz-go/pkg/kgo"
)

const (
	MinimumTradeSize = 10000 // USD
)

// UserProfile represents a user profile fetched from Polymarket API
type UserProfile struct {
	Address      string    `json:"address"`
	Name         string    `json:"name,omitempty"`
	Pseudonym    string    `json:"pseudonym,omitempty"`
	Bio          string    `json:"bio,omitempty"`
	Icon         string    `json:"icon,omitempty"`
	ProfileImage string    `json:"profileImage,omitempty"`
	FirstSeen    time.Time `json:"firstSeen"`
	LastSeen     time.Time `json:"lastSeen"`
}

// DiscoveryService handles discovery of high-value traders
type DiscoveryService struct {
	consumer      *internalkafka.Consumer
	profileWriter *internalqdb.ProfileWriter
	seenAddresses map[string]bool
	mu            sync.RWMutex
}

// NewDiscoveryService creates a new discovery service
func NewDiscoveryService(brokers string, topic string, groupID string) (*DiscoveryService, error) {
	consumer, err := internalkafka.NewConsumer(brokers, topic, groupID)
	if err != nil {
		return nil, fmt.Errorf("failed to create kafka consumer: %w", err)
	}

	// Create QuestDB writer for profiles
	ctx := context.Background()
	host := config.AppConfig.QuestDBHost
	portStr := config.AppConfig.QuestDBILPPort
	if portStr == "" {
		portStr = "9009" // Default ILP port
	}
	port, err := strconv.Atoi(portStr)
	if err != nil {
		port = 9009 // Fallback to default
	}
	profileWriter, err := internalqdb.NewProfileWriter(ctx, host, port)
	if err != nil {
		return nil, fmt.Errorf("failed to create profile writer: %w", err)
	}

	return &DiscoveryService{
		consumer:      consumer,
		profileWriter: profileWriter,
		seenAddresses: make(map[string]bool),
	}, nil
}

// Run starts the discovery service
func (ds *DiscoveryService) Run(ctx context.Context) error {
	return ds.consumer.Run(ctx, ds.handleTrade)
}

// handleTrade processes a trade message from Kafka
func (ds *DiscoveryService) handleTrade(record *kgo.Record) {
	var tradeMsg internalkafka.TradeMessage
	var tradeSizeInUSD float64
	if err := json.Unmarshal(record.Value, &tradeMsg); err != nil {
		log.Printf("Error unmarshaling trade message: %v", err)
		return
	}

	tradeSizeInUSD = tradeMsg.Size * tradeMsg.Price
	// Filter trades with size >= 10k USD
	if tradeSizeInUSD < MinimumTradeSize {
		return
	}

	log.Printf("Processing high-value trade: size=%.2f, proxyWallet=%s",
		tradeMsg.Size, tradeMsg.ProxyWallet)

	// Process proxy wallet address
	if tradeMsg.ProxyWallet != "" {
		go ds.fetchAndSaveProfile(context.Background(), tradeMsg.ProxyWallet)
	}
}

// fetchAndSaveProfile saves a user profile to QuestDB
func (ds *DiscoveryService) fetchAndSaveProfile(ctx context.Context, address string) {
	// Check if we've already processed this address
	ds.mu.Lock()
	if ds.seenAddresses[strings.ToLower(address)] {
		ds.mu.Unlock()
		return
	}
	ds.seenAddresses[strings.ToLower(address)] = true
	ds.mu.Unlock()

	// Create profile with just the address
	profile := &internalqdb.UserProfile{
		Address: address,
	}

	// Write profile to QuestDB
	if err := ds.profileWriter.Write(ctx, profile); err != nil {
		log.Printf("Error writing profile to QuestDB for address %s: %v", address, err)
		return
	}

	// Flush to ensure data is written
	if err := ds.profileWriter.Flush(ctx); err != nil {
		log.Printf("Error flushing profile to QuestDB for address %s: %v", address, err)
		return
	}

	log.Printf("Saved profile for address: %s", address)
}

// Close closes the discovery service
func (ds *DiscoveryService) Close() {
	if ds.consumer != nil {
		ds.consumer.Close()
	}
	if ds.profileWriter != nil {
		ctx := context.Background()
		ds.profileWriter.Close(ctx)
	}
}
