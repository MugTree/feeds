-- +goose Up
-- +goose StatementBegin
PRAGMA foreign_keys = ON;
CREATE TABLE IF NOT EXISTS feeds
(
    id           INTEGER PRIMARY KEY AUTOINCREMENT,
    url          TEXT NOT NULL UNIQUE,
    title        TEXT DEFAULT '',
    last_fetched TIMESTAMP NOT NULL
);

CREATE TABLE IF NOT EXISTS articles
(
    id        INTEGER PRIMARY KEY AUTOINCREMENT,
    feed_id   INTEGER NOT NULL,
    title     TEXT    DEFAULT '',
    link      TEXT    DEFAULT '',
    published TIMESTAMP NOT NULL,
    published_parsed TEXT NOT NULL,
    summary   TEXT,
    read      INTEGER DEFAULT 0,
    starred   INTEGER DEFAULT 0,
    UNIQUE (feed_id, link),
    FOREIGN KEY (feed_id) REFERENCES feeds (id) ON DELETE CASCADE
);
-- +goose StatementEnd

