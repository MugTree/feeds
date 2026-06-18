-- +goose Up
-- +goose StatementBegin
ALTER TABLE article_cache ADD COLUMN modified_article_content TEXT DEFAULT '';
ALTER TABLE article_cache ADD COLUMN modified TIMESTAMP NULL;
-- +goose StatementEnd

