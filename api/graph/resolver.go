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

// surpriseSubscriber tracks a WebSocket channel for surprise upvote results.
// Scoped to room so all clients see the upvote update.
type surpriseSubscriber struct {
	room string
	ch   chan *model.Message
}

// calorieSubscriber tracks a WebSocket channel for calorie query results.
type calorieSubscriber struct {
	ch chan *model.CalorieQuery
}

// Resolver uses Postgres for persistence, in-memory channels for
// real-time WebSocket push, and a function to enqueue async jobs.
type Resolver struct {
	DB                *db.DB
	InsertJob         func(messageID string) error
	InsertSurpriseJob func(messageID string) error
	InsertCalorieJob  func(queryID string) error

	subscribers         map[int]*subscriber
	jobSubscribers      map[int]*jobSubscriber
	surpriseSubscribers map[int]*surpriseSubscriber
	calorieSubscribers  map[int]*calorieSubscriber
	nextID              int
	mu                  sync.Mutex
}

func NewResolver(database *db.DB) *Resolver {
	return &Resolver{
		DB:                  database,
		subscribers:         make(map[int]*subscriber),
		jobSubscribers:      make(map[int]*jobSubscriber),
		surpriseSubscribers: make(map[int]*surpriseSubscriber),
		calorieSubscribers:  make(map[int]*calorieSubscriber),
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

// HandleSurpriseComplete is called by the surprise_upvote worker when it finishes.
// It broadcasts the updated message (with upvotes) to all clients in the room.
func (r *Resolver) HandleSurpriseComplete(msg *model.Message) {
	r.mu.Lock()
	for _, sub := range r.surpriseSubscribers {
		if sub.room == msg.Room {
			select {
			case sub.ch <- msg:
			default:
			}
		}
	}
	r.mu.Unlock()
}

// HandleCalorieComplete is called by the estimate_calories worker when it finishes.
// It broadcasts the result to all calorie page subscribers.
func (r *Resolver) HandleCalorieComplete(result *model.CalorieQuery) {
	r.mu.Lock()
	for _, sub := range r.calorieSubscribers {
		select {
		case sub.ch <- result:
		default:
		}
	}
	r.mu.Unlock()
}
