package kafka

import (
	"context"
	"log"

	"github.com/twmb/franz-go/pkg/kgo"
)

// Consumer is a simple Kafka consumer wrapper.
// It is not wired into main yet; you can use it in a separate
// service for notifications, analytics, etc.
type Consumer struct {
	client *kgo.Client
}

// NewConsumer creates a new consumer subscribed to the given topic.
func NewConsumer(brokers string, topic string, groupID string) (*Consumer, error) {
	opts := []kgo.Opt{
		kgo.SeedBrokers(brokers),
		kgo.ConsumerGroup(groupID),
		kgo.ConsumeTopics(topic),
	}

	cl, err := kgo.NewClient(opts...)
	if err != nil {
		return nil, err
	}

	return &Consumer{client: cl}, nil
}

// Run starts a basic poll loop and passes records to the handler.
func (c *Consumer) Run(ctx context.Context, handler func(*kgo.Record)) error {
	for {
		fetches := c.client.PollFetches(ctx)
		if errs := fetches.Errors(); len(errs) > 0 {
			for _, e := range errs {
				log.Printf("Kafka fetch error: %v", e)
			}
		}
		fetches.EachRecord(func(r *kgo.Record) {
			if handler != nil {
				handler(r)
			}
		})
	}
}

// Close closes the consumer client.
func (c *Consumer) Close() {
	if c.client != nil {
		c.client.Close()
	}
}
