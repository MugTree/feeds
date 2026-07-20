-- +goose Up
-- +goose StatementBegin
CREATE TABLE IF NOT EXISTS margin_notes
(
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    article_id INTEGER NOT NULL,
    block_id INTEGER NOT NULL,
    note TEXT NOT NULL DEFAULT '', 
    date_added TIMESTAMP NOT NULL
);
-- +goose StatementEnd
