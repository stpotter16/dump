CREATE TABLE IF NOT EXISTS session (
    session_key TEXT PRIMARY KEY,
    value BLOB,
    expires_at TEXT NOT NULL
) STRICT;

CREATE TABLE IF NOT EXISTS idea (
    id INTEGER PRIMARY KEY,
    text TEXT NOT NULL,
    created_time TEXT NOT NULL,
    embedding BLOB
) STRICT;
