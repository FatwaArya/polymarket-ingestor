package kafka

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"strings"

	"github.com/FatwaArya/pm-ingest/utils"
	"github.com/twmb/franz-go/pkg/kgo"
)

type Producer struct {
	client *kgo.Client
	topic  string
}

type TradeMessage struct {
	Side            string  `json:"side"`
	Outcome         string  `json:"outcome"`
	EventSlug       string  `json:"eventSlug"`
	Slug            string  `json:"slug"`
	ConditionId     string  `json:"conditionId"`
	TransactionHash string  `json:"transactionHash"`
	ProxyWallet     string  `json:"proxyWallet"`
	QuestionId      string  `json:"questionId"`
	Price           float64 `json:"price"`
	Size            float64 `json:"size"`
	Fee             float64 `json:"fee"`
	Timestamp       int64   `json:"timestamp"`
}

// NewProducer creates a Kafka producer for the given brokers and topic.
// brokers: comma-separated list, e.g. "localhost:19092"
func NewProducer(brokers string, topic string) (*Producer, error) {
	bs := strings.Split(brokers, ",")
	opts := []kgo.Opt{
		kgo.SeedBrokers(bs...),
		kgo.AllowAutoTopicCreation(),
	}

	cl, err := kgo.NewClient(opts...)
	if err != nil {
		return nil, fmt.Errorf("failed to create kafka client: %w", err)
	}

	return &Producer{
		client: cl,
		topic:  topic,
	}, nil
}

// ProduceTrade serializes the trade as JSON and sends it to Kafka.
func (p *Producer) ProduceTrade(ctx context.Context, trade *utils.ActivityTradePayload) error {
	if trade == nil {
		return nil
	}
	tradeMessage := TradeMessage{
		Side:            trade.Side,
		Outcome:         trade.OutcomeTitle,
		EventSlug:       trade.EventSlug,
		Slug:            trade.MarketSlug,
		ConditionId:     trade.ConditionID,
		TransactionHash: trade.TransactionHash,
		ProxyWallet:     trade.ProxyWalletAddress,
		QuestionId:      trade.QuestionID,
		Price:           trade.Price,
		Size:            trade.Size,
		Fee:             trade.Fee,
		Timestamp:       trade.Timestamp,
	}

	value, err := json.Marshal(tradeMessage)
	if err != nil {
		return fmt.Errorf("failed to marshal trade: %w", err)
	}

	// Use transaction hash as key when available to keep related records in the same partition.
	var key []byte
	if trade.TransactionHash != "" {
		key = []byte(trade.TransactionHash)
	}

	record := &kgo.Record{
		Topic: p.topic,
		Key:   key,
		Value: value,
	}

	// Asynchronous production with callback logging.
	p.client.Produce(ctx, record, func(record *kgo.Record, err error) {
		if err != nil {
			log.Printf("Kafka produce error: %v", err)
		}
	})

	return nil
}

// Close flushes pending records and closes the Kafka client.
func (p *Producer) Close() {
	if p.client != nil {
		p.client.Close()
	}
}
