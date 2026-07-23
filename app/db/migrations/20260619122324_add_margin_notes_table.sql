-- +goose Up
-- +goose StatementBegin
CREATE TABLE IF NOT EXISTS margin_notes
(
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    article_id INTEGER NOT NULL,
    related_clickable_block_id INTEGER NOT NULL,
    note TEXT NOT NULL DEFAULT '', 
    date_added TIMESTAMP NOT NULL
);

ALTER TABLE article_cache ADD COLUMN clickable_block_count INTEGER NOT NULL;
-- +goose StatementEnd
