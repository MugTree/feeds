-- +goose Up
-- +goose StatementBegin
PRAGMA foreign_keys = ON;
CREATE TABLE IF NOT EXISTS feeds
(
    id           INTEGER PRIMARY KEY AUTOINCREMENT,
    url          TEXT NOT NULL UNIQUE,
    title        TEXT,
    last_fetched TIMESTAMP
);

CREATE TABLE IF NOT EXISTS articles
(
    id        INTEGER PRIMARY KEY AUTOINCREMENT,
    feed_id   INTEGER NOT NULL,
    title     TEXT    NOT NULL,
    link      TEXT    NOT NULL,
    published TIMESTAMP,
    published_parsed TEXT NOT NULL,
    updated TIMESTAMP,
    updated_parsed TEXT NOT NULL,
    summary   TEXT,
    read      INTEGER DEFAULT 0,
    starred   INTEGER DEFAULT 0,
    UNIQUE (feed_id, link),
    FOREIGN KEY (feed_id) REFERENCES feeds (id) ON DELETE CASCADE
);
-- +goose StatementEnd

-- +goose Down
-- +goose StatementBegin
DROP TABLE IF EXISTS articles;
DROP TABLE IF EXISTS feeds;
-- +goose StatementEnd
