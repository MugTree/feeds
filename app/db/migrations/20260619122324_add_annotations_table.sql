-- +goose Up
-- +goose StatementBegin
CREATE TABLE IF NOT EXISTS annotations
(
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    article_id INTEGER NOT NULL,
    start_data TEXT NOT NULL,
    end_data TEXT NOT NULL,
    snippet TEXT NOT NULL DEFAULT '',
    note TEXT NOT NULL DEFAULT '', 
    date_added TIMESTAMP NOT NULL
);
-- +goose StatementEnd
