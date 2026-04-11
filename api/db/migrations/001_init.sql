CREATE TABLE IF NOT EXISTS rooms (
    id   TEXT PRIMARY KEY,
    name TEXT UNIQUE NOT NULL
);

CREATE TABLE IF NOT EXISTS messages (
    id        TEXT PRIMARY KEY,
    room      TEXT NOT NULL,
    sender    TEXT NOT NULL,
    text      TEXT NOT NULL,
    timestamp TIMESTAMPTZ NOT NULL,
    upvotes   INTEGER NOT NULL DEFAULT 0
);

CREATE INDEX IF NOT EXISTS idx_messages_room_ts ON messages(room, timestamp);

-- For existing databases where CREATE TABLE already ran without the upvotes column.
ALTER TABLE messages ADD COLUMN IF NOT EXISTS upvotes INTEGER NOT NULL DEFAULT 0;

CREATE TABLE IF NOT EXISTS calorie_queries (
    id               TEXT PRIMARY KEY,
    meal_text        TEXT NOT NULL,
    model            TEXT NOT NULL,
    calories         INTEGER,
    response_time_ms INTEGER,
    total_tokens     INTEGER,
    status           TEXT NOT NULL DEFAULT 'pending',
    error_message    TEXT,
    created_at       TIMESTAMPTZ NOT NULL DEFAULT NOW()
);
