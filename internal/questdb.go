package internal

import (
	"context"
	"fmt"
	"sync"
	"time"

	"github.com/FatwaArya/pm-ingest/utils"
	qdb "github.com/questdb/go-questdb-client/v3"
)

type TradeWriter struct {
	sender        qdb.LineSender
	tableName     string
	flushInterval time.Duration
	done          chan struct{}
	mu            sync.Mutex
}

// NewTradeWriter creates a new QuestDB trade writer using ILP over TCP
// with periodic background flushing (auto-flush not supported for TCP)
func NewTradeWriter(ctx context.Context, host string, port int) (*TradeWriter, error) {
	conf := fmt.Sprintf("tcp::addr=%s:%d;", host, port)

	sender, err := qdb.LineSenderFromConf(ctx, conf)
	if err != nil {
		return nil, err
	}

	w := &TradeWriter{
		sender:        sender,
		tableName:     "polymarket_trades",
		flushInterval: time.Second, // Flush every 1 second
		done:          make(chan struct{}),
	}

	// Start background flusher for TCP
	go w.backgroundFlush(ctx)

	return w, nil
}

// NewTradeWriterHTTP creates a new QuestDB trade writer using HTTP protocol with auto-flush
func NewTradeWriterHTTP(ctx context.Context, host string, port int) (*TradeWriter, error) {
	// HTTP protocol supports auto-flush
	conf := fmt.Sprintf("http::addr=%s:%d;auto_flush_interval=1000;", host, port)

	sender, err := qdb.LineSenderFromConf(ctx, conf)
	if err != nil {
		return nil, err
	}
	return &TradeWriter{
		sender:    sender,
		tableName: "polymarket_trades",
	}, nil
}

// backgroundFlush periodically flushes data to QuestDB (for TCP client)
func (w *TradeWriter) backgroundFlush(ctx context.Context) {
	ticker := time.NewTicker(w.flushInterval)
	defer ticker.Stop()

	for {
		select {
		case <-ticker.C:
			w.mu.Lock()
			if err := w.sender.Flush(ctx); err != nil {
				// Log error but don't stop flushing
				fmt.Printf("QuestDB flush error: %v\n", err)
			}
			w.mu.Unlock()
		case <-w.done:
			return
		case <-ctx.Done():
			return
		}
	}
}

// Write writes a single trade to QuestDB
func (w *TradeWriter) Write(ctx context.Context, trade *utils.ActivityTradePayload) error {
	// Timestamp in the payload is in seconds, convert to time.Time
	ts := time.Unix(trade.Timestamp, 0)

	w.mu.Lock()
	defer w.mu.Unlock()

	return w.sender.
		Table(w.tableName).
		Symbol("side", trade.Side).
		Symbol("outcome", trade.OutcomeTitle).
		Symbol("event_slug", trade.EventSlug).
		StringColumn("asset", trade.Asset).
		Float64Column("price", trade.Price).
		Float64Column("size", trade.Size).
		StringColumn("transaction_hash", trade.TransactionHash).
		StringColumn("condition_id", trade.ConditionID).
		Int64Column("outcome_index", int64(trade.OutcomeIndex)).
		StringColumn("market_slug", trade.MarketSlug).
		StringColumn("event_title", trade.EventTitle).
		StringColumn("proxy_wallet", trade.ProxyWalletAddress).
		StringColumn("name", trade.Name).
		StringColumn("pseudonym", trade.Pseudonym).
		At(ctx, ts)
}

// WriteBatch writes multiple trades to QuestDB
func (w *TradeWriter) WriteBatch(ctx context.Context, trades []*utils.ActivityTradePayload) error {
	for _, trade := range trades {
		if err := w.Write(ctx, trade); err != nil {
			return err
		}
	}
	return w.Flush(ctx)
}

// Flush sends all buffered data to QuestDB
func (w *TradeWriter) Flush(ctx context.Context) error {
	w.mu.Lock()
	defer w.mu.Unlock()
	return w.sender.Flush(ctx)
}

// Close stops the background flusher and closes the connection to QuestDB
func (w *TradeWriter) Close(ctx context.Context) error {
	// Stop background flusher if running
	if w.done != nil {
		close(w.done)
	}

	w.mu.Lock()
	defer w.mu.Unlock()

	// Final flush before closing
	if err := w.sender.Flush(ctx); err != nil {
		fmt.Printf("QuestDB final flush error: %v\n", err)
	}

	return w.sender.Close(ctx)
}
