package jobs

import (
	"context"
	"fmt"
	"math/rand"
	"time"

	"github.com/riverqueue/river"

	"github.com/gotesting/api/db"
)

// ProcessMessageArgs stores only the message ID — the worker fetches
// the full message from the database when it runs.
type ProcessMessageArgs struct {
	MessageID string `json:"message_id"`
}

func (ProcessMessageArgs) Kind() string { return "process_message" }

// ProcessMessageWorker simulates async message processing.
// In a real app this might call an LLM, run moderation, etc.
type ProcessMessageWorker struct {
	river.WorkerDefaults[ProcessMessageArgs]

	DB *db.DB

	// OnComplete is called when the job finishes. The resolver uses this
	// to push results to WebSocket subscribers.
	OnComplete func(messageID, room, sender string, result int)
}

func (w *ProcessMessageWorker) Work(ctx context.Context, job *river.Job[ProcessMessageArgs]) error {
	// Look up the message from the database.
	msg, err := w.DB.GetMessage(ctx, job.Args.MessageID)
	if err != nil {
		return fmt.Errorf("failed to fetch message %s: %w", job.Args.MessageID, err)
	}

	// Simulate work — sleep 2-5 seconds.
	delay := 2 + rand.Intn(4)
	time.Sleep(time.Duration(delay) * time.Second)

	// Generate a random "result" (stand-in for an LLM response, score, etc.).
	result := rand.Intn(1000)

	if w.OnComplete != nil {
		w.OnComplete(msg.ID, msg.Room, msg.Sender, result)
	}

	return nil
}
