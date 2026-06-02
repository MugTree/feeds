-- +goose Up
-- +goose StatementBegin
CREATE TABLE IF NOT EXISTS article_cache
(
    id           INTEGER PRIMARY KEY AUTOINCREMENT,
    link         TEXT NOT NULL UNIQUE,
    article_content      TEXT,
    created      TIMESTAMP
);
-- +goose StatementEnd

-- +goose Down
-- +goose StatementBegin
DROP TABLE IF EXISTS article_cache;
-- +goose StatementEnd
