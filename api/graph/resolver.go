package graph

import (
	"sync"
	"time"

	"github.com/gotesting/api/db"
	"github.com/gotesting/api/graph/model"
)

// subscriber tracks a WebSocket channel for chat messages.
type subscriber struct {
	room string
	ch   chan *model.Message
}

// jobSubscriber tracks a WebSocket channel for job completion results.
// Scoped to both room and sender so results only reach the originator.
type jobSubscriber struct {
	room   string
	sender string
	ch     chan *model.JobResult
}

// Resolver uses Postgres for persistence, in-memory channels for
// real-time WebSocket push, and a function to enqueue async jobs.
type Resolver struct {
	DB        *db.DB
	InsertJob func(messageID string) error

	subscribers    map[int]*subscriber
	jobSubscribers map[int]*jobSubscriber
	nextID         int
	mu             sync.Mutex
}

func NewResolver(database *db.DB) *Resolver {
	return &Resolver{
		DB:             database,
		subscribers:    make(map[int]*subscriber),
		jobSubscribers: make(map[int]*jobSubscriber),
	}
}

// HandleJobComplete is called by the River worker when a job finishes.
// It delivers the result only to the subscriber who sent the original message.
func (r *Resolver) HandleJobComplete(messageID, room, sender string, result int) {
	jr := &model.JobResult{
		MessageID: messageID,
		Room:      room,
		Sender:    sender,
		Result:    result,
		Timestamp: time.Now().UTC().Format(time.RFC3339),
	}

	r.mu.Lock()
	for _, sub := range r.jobSubscribers {
		if sub.room == room && sub.sender == sender {
			select {
			case sub.ch <- jr:
			default:
			}
		}
	}
	r.mu.Unlock()
}
