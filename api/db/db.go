package db

import (
	"context"
	"embed"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/use-golang/api/graph/model"
)

//go:embed migrations/*.sql
var migrations embed.FS

// DB wraps a pgx connection pool.
type DB struct {
	Pool *pgxpool.Pool
}

// Connect creates a connection pool to Postgres.
func Connect(ctx context.Context, dsn string) (*DB, error) {
	pool, err := pgxpool.New(ctx, dsn)
	if err != nil {
		return nil, err
	}
	if err := pool.Ping(ctx); err != nil {
		pool.Close()
		return nil, err
	}
	return &DB{Pool: pool}, nil
}

// Migrate runs the embedded SQL migration files.
func (d *DB) Migrate(ctx context.Context) error {
	sql, err := migrations.ReadFile("migrations/001_init.sql")
	if err != nil {
		return err
	}
	_, err = d.Pool.Exec(ctx, string(sql))
	return err
}

// Close shuts down the connection pool.
func (d *DB) Close() {
	d.Pool.Close()
}

// ---- Rooms ----

func (d *DB) GetRooms(ctx context.Context) ([]*model.Room, error) {
	rows, err := d.Pool.Query(ctx, "SELECT id, name FROM rooms ORDER BY name")
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var rooms []*model.Room
	for rows.Next() {
		r := &model.Room{}
		if err := rows.Scan(&r.ID, &r.Name); err != nil {
			return nil, err
		}
		rooms = append(rooms, r)
	}
	return rooms, rows.Err()
}

func (d *DB) CreateRoom(ctx context.Context, room *model.Room) (*model.Room, error) {
	// ON CONFLICT makes this idempotent — creating an existing room returns it.
	err := d.Pool.QueryRow(ctx,
		"INSERT INTO rooms (id, name) VALUES ($1, $2) ON CONFLICT (name) DO UPDATE SET name = EXCLUDED.name RETURNING id, name",
		room.ID, room.Name,
	).Scan(&room.ID, &room.Name)
	if err != nil {
		return nil, err
	}
	return room, nil
}

// ---- Messages ----

func (d *DB) SaveMessage(ctx context.Context, msg *model.Message) error {
	_, err := d.Pool.Exec(ctx,
		"INSERT INTO messages (id, room, sender, text, timestamp, upvotes) VALUES ($1, $2, $3, $4, $5, $6)",
		msg.ID, msg.Room, msg.Sender, msg.Text, msg.Timestamp, msg.Upvotes,
	)
	return err
}

func (d *DB) GetMessage(ctx context.Context, id string) (*model.Message, error) {
	m := &model.Message{}
	var ts time.Time
	err := d.Pool.QueryRow(ctx,
		"SELECT id, room, sender, text, timestamp, upvotes FROM messages WHERE id = $1", id,
	).Scan(&m.ID, &m.Room, &m.Sender, &m.Text, &ts, &m.Upvotes)
	if err != nil {
		return nil, err
	}
	m.Timestamp = ts.UTC().Format(time.RFC3339)
	return m, nil
}

func (d *DB) GetMessages(ctx context.Context, room string) ([]*model.Message, error) {
	rows, err := d.Pool.Query(ctx,
		"SELECT id, room, sender, text, timestamp, upvotes FROM messages WHERE room = $1 ORDER BY timestamp ASC",
		room,
	)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var msgs []*model.Message
	for rows.Next() {
		m := &model.Message{}
		var ts time.Time
		if err := rows.Scan(&m.ID, &m.Room, &m.Sender, &m.Text, &ts, &m.Upvotes); err != nil {
			return nil, err
		}
		m.Timestamp = ts.UTC().Format(time.RFC3339)
		msgs = append(msgs, m)
	}
	return msgs, rows.Err()
}

// UpdateMessageUpvotes sets the upvotes count for a specific message.
func (d *DB) UpdateMessageUpvotes(ctx context.Context, id string, upvotes int) error {
	_, err := d.Pool.Exec(ctx,
		"UPDATE messages SET upvotes = $2 WHERE id = $1",
		id, upvotes,
	)
	return err
}

// ---- Calorie Queries ----

func (d *DB) SaveCalorieQuery(ctx context.Context, q *model.CalorieQuery) error {
	_, err := d.Pool.Exec(ctx,
		"INSERT INTO calorie_queries (id, meal_text, model, reasoning_effort, status, created_at) VALUES ($1, $2, $3, $4, $5, $6)",
		q.ID, q.MealText, q.Model, q.ReasoningEffort, q.Status, q.CreatedAt,
	)
	return err
}

func (d *DB) GetCalorieQuery(ctx context.Context, id string) (*model.CalorieQuery, error) {
	q := &model.CalorieQuery{}
	var createdAt time.Time
	err := d.Pool.QueryRow(ctx,
		"SELECT id, meal_text, model, reasoning_effort, calories, response_time_ms, total_tokens, status, error_message, created_at FROM calorie_queries WHERE id = $1", id,
	).Scan(&q.ID, &q.MealText, &q.Model, &q.ReasoningEffort, &q.Calories, &q.ResponseTimeMs, &q.TotalTokens, &q.Status, &q.ErrorMessage, &createdAt)
	if err != nil {
		return nil, err
	}
	q.CreatedAt = createdAt.UTC().Format(time.RFC3339)
	return q, nil
}

func (d *DB) GetCalorieQueries(ctx context.Context) ([]*model.CalorieQuery, error) {
	rows, err := d.Pool.Query(ctx,
		"SELECT id, meal_text, model, reasoning_effort, calories, response_time_ms, total_tokens, status, error_message, created_at FROM calorie_queries ORDER BY created_at DESC",
	)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var queries []*model.CalorieQuery
	for rows.Next() {
		q := &model.CalorieQuery{}
		var createdAt time.Time
		if err := rows.Scan(&q.ID, &q.MealText, &q.Model, &q.ReasoningEffort, &q.Calories, &q.ResponseTimeMs, &q.TotalTokens, &q.Status, &q.ErrorMessage, &createdAt); err != nil {
			return nil, err
		}
		q.CreatedAt = createdAt.UTC().Format(time.RFC3339)
		queries = append(queries, q)
	}
	return queries, rows.Err()
}

func (d *DB) DeleteAllCalorieQueries(ctx context.Context) error {
	_, err := d.Pool.Exec(ctx, "DELETE FROM calorie_queries")
	return err
}

func (d *DB) DeleteCalorieQuery(ctx context.Context, id string) error {
	_, err := d.Pool.Exec(ctx, "DELETE FROM calorie_queries WHERE id = $1", id)
	return err
}

func (d *DB) UpdateCalorieQuery(ctx context.Context, id string, calories *int, responseTimeMs *int, totalTokens *int, status string, errorMessage *string) error {
	_, err := d.Pool.Exec(ctx,
		"UPDATE calorie_queries SET calories = $2, response_time_ms = $3, total_tokens = $4, status = $5, error_message = $6 WHERE id = $1",
		id, calories, responseTimeMs, totalTokens, status, errorMessage,
	)
	return err
}
