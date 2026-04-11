CREATE TABLE IF NOT EXISTS rooms (
    id   TEXT PRIMARY KEY,
    name TEXT UNIQUE NOT NULL
);

CREATE TABLE IF NOT EXISTS messages (
    id        TEXT PRIMARY KEY,
    room      TEXT NOT NULL,
    sender    TEXT NOT NULL,
    text      TEXT NOT NULL,
    timestamp TIMESTAMPTZ NOT NULL
);

CREATE INDEX IF NOT EXISTS idx_messages_room_ts ON messages(room, timestamp);
