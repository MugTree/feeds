-- +goose Up
-- +goose StatementBegin
CREATE TABLE IF NOT EXISTS annotations
(
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    article_id INTEGER NOT NULL,
    start_pos INTEGER NOT NULL,
    end_pos INTEGER NOT NULL,
    snippet TEXT NOT NULL DEFAULT '',
    note TEXT NOT NULL DEFAULT '', 
    date_added TIMESTAMP NOT NULL
);
-- +goose StatementEnd
