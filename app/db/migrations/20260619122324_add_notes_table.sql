-- +goose Up
-- +goose StatementBegin
CREATE TABLE IF NOT EXISTS notes
(
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    article_id INTEGER NOT NULL,
    page_id INTEGER NOT NULL,
    note_text TEXT NOT NULL DEFAULT '', 
    date_added TIMESTAMP NOT NULL
);
-- +goose StatementEnd
