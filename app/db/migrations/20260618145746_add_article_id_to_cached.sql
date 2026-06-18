-- +goose Up
-- +goose StatementBegin
ALTER TABLE article_cache ADD COLUMN article_id  INTEGER NOT NULL;
-- +goose StatementEnd

