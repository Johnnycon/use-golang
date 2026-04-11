package db

import (
	"context"
	"embed"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/gotesting/api/graph/model"
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
		"INSERT INTO messages (id, room, sender, text, timestamp) VALUES ($1, $2, $3, $4, $5)",
		msg.ID, msg.Room, msg.Sender, msg.Text, msg.Timestamp,
	)
	return err
}

func (d *DB) GetMessage(ctx context.Context, id string) (*model.Message, error) {
	m := &model.Message{}
	var ts time.Time
	err := d.Pool.QueryRow(ctx,
		"SELECT id, room, sender, text, timestamp FROM messages WHERE id = $1", id,
	).Scan(&m.ID, &m.Room, &m.Sender, &m.Text, &ts)
	if err != nil {
		return nil, err
	}
	m.Timestamp = ts.UTC().Format(time.RFC3339)
	return m, nil
}

func (d *DB) GetMessages(ctx context.Context, room string) ([]*model.Message, error) {
	rows, err := d.Pool.Query(ctx,
		"SELECT id, room, sender, text, timestamp FROM messages WHERE room = $1 ORDER BY timestamp ASC",
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
		if err := rows.Scan(&m.ID, &m.Room, &m.Sender, &m.Text, &ts); err != nil {
			return nil, err
		}
		m.Timestamp = ts.UTC().Format(time.RFC3339)
		msgs = append(msgs, m)
	}
	return msgs, rows.Err()
}
