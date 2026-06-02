-- +goose Up
-- +goose StatementBegin
CREATE TABLE IF NOT EXISTS article_cache
(
    id           INTEGER PRIMARY KEY AUTOINCREMENT,
    link         TEXT NOT NULL UNIQUE,
    article_content      TEXT DEFAULT '',
    created      TIMESTAMP NOT NULL
);
-- +goose StatementEnd

