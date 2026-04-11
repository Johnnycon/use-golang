package jobs

import (
	"context"
	"fmt"
	"math/rand"
	"time"

	"github.com/riverqueue/river"

	"github.com/gotesting/api/db"
	"github.com/gotesting/api/graph/model"
)

// SurpriseUpvoteArgs stores only the message ID — the worker fetches
// the full message from the database when it runs.
type SurpriseUpvoteArgs struct {
	MessageID string `json:"message_id"`
}

func (SurpriseUpvoteArgs) Kind() string { return "surprise_upvote" }

func (SurpriseUpvoteArgs) InsertOpts() river.InsertOpts {
	return river.InsertOpts{Queue: "surprises"}
}

// SurpriseUpvoteWorker assigns a random number of upvotes (1-100) to a message.
type SurpriseUpvoteWorker struct {
	river.WorkerDefaults[SurpriseUpvoteArgs]

	DB *db.DB

	// OnComplete is called with the fully-updated message after upvotes are assigned.
	OnComplete func(msg *model.Message)
}

func (w *SurpriseUpvoteWorker) Work(ctx context.Context, job *river.Job[SurpriseUpvoteArgs]) error {
	msg, err := w.DB.GetMessage(ctx, job.Args.MessageID)
	if err != nil {
		return fmt.Errorf("failed to fetch message %s: %w", job.Args.MessageID, err)
	}

	// Simulate a brief delay (1-3 seconds).
	delay := 1 + rand.Intn(3)
	time.Sleep(time.Duration(delay) * time.Second)

	// Assign random upvotes between 1 and 100.
	upvotes := 1 + rand.Intn(100)

	if err := w.DB.UpdateMessageUpvotes(ctx, msg.ID, upvotes); err != nil {
		return fmt.Errorf("failed to update upvotes for message %s: %w", msg.ID, err)
	}

	msg.Upvotes = upvotes

	if w.OnComplete != nil {
		w.OnComplete(msg)
	}

	return nil
}
