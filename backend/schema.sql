CREATE TABLE IF NOT EXISTS reservations (
    id         SERIAL PRIMARY KEY,
    name       TEXT NOT NULL,
    people     INTEGER NOT NULL,
    created_at TIMESTAMPTZ DEFAULT NOW()
);