package internal

import (
	"context"
	"fmt"
	"log"
	"sync"
	"time"

	qdb "github.com/questdb/go-questdb-client/v3"
)

// ProfileWriter writes user profiles to QuestDB
type ProfileWriter struct {
	sender    qdb.LineSender
	tableName string
	mu        sync.Mutex
}

// UserProfile represents a user profile to be written to QuestDB
type UserProfile struct {
	Address      string
	Name         string
	Pseudonym    string
	Bio          string
	Icon         string
	ProfileImage string
}

// NewProfileWriter creates a new QuestDB profile writer using ILP over TCP
func NewProfileWriter(ctx context.Context, host string, port int) (*ProfileWriter, error) {
	conf := fmt.Sprintf("tcp::addr=%s:%d;", host, port)

	sender, err := qdb.LineSenderFromConf(ctx, conf)
	if err != nil {
		return nil, err
	}

	return &ProfileWriter{
		sender:    sender,
		tableName: "user_profiles",
	}, nil
}

// Write writes a user profile to QuestDB
func (w *ProfileWriter) Write(ctx context.Context, profile *UserProfile) error {
	w.mu.Lock()
	defer w.mu.Unlock()

	return w.sender.
		Table(w.tableName).
		Symbol("address", profile.Address).
		StringColumn("name", profile.Name).
		StringColumn("pseudonym", profile.Pseudonym).
		StringColumn("bio", profile.Bio).
		StringColumn("icon", profile.Icon).
		StringColumn("profile_image", profile.ProfileImage).
		At(ctx, time.Now())
}

// Flush sends all buffered data to QuestDB
func (w *ProfileWriter) Flush(ctx context.Context) error {
	w.mu.Lock()
	defer w.mu.Unlock()
	return w.sender.Flush(ctx)
}

// Close flushes pending data and closes the connection to QuestDB
func (w *ProfileWriter) Close(ctx context.Context) error {
	w.mu.Lock()
	defer w.mu.Unlock()

	// Final flush before closing
	if err := w.sender.Flush(ctx); err != nil {
		log.Printf("QuestDB final flush error: %v", err)
	}

	return w.sender.Close(ctx)
}
