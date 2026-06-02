-- +goose Up
-- +goose StatementBegin
PRAGMA foreign_keys = ON;
CREATE TABLE IF NOT EXISTS feeds
(
    id           INTEGER PRIMARY KEY AUTOINCREMENT,
    url          TEXT NOT NULL UNIQUE,
    title        TEXT NOT NULL DEFAULT '',
    last_fetched TIMESTAMP NOT NULL
);

CREATE TABLE IF NOT EXISTS articles
(
    id        INTEGER PRIMARY KEY AUTOINCREMENT,
    feed_id   INTEGER NOT NULL,
    title     TEXT NOT NULL DEFAULT '',
    link      TEXT NOT NULL DEFAULT '',
    published TIMESTAMP NOT NULL,
    published_parsed TEXT NOT NULL DEFAULT '',
    summary   TEXT NOT NULL DEFAULT '',
    read      INTEGER NOT NULL DEFAULT 0,
    starred   INTEGER NOT NULL DEFAULT 0,
    UNIQUE (feed_id, link),
    FOREIGN KEY (feed_id) REFERENCES feeds (id) ON DELETE CASCADE
);
-- +goose StatementEnd

