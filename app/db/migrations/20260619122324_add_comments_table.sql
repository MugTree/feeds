-- +goose Up
-- +goose StatementBegin
CREATE TABLE IF NOT EXISTS comments
(
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    article_id INTEGER NOT NULL,
    related_paragraph_id INTEGER NOT NULL,
    comment_text TEXT NOT NULL DEFAULT '', 
    date_added TIMESTAMP NOT NULL
);
-- +goose StatementEnd
