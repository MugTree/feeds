-- +goose Up
-- +goose StatementBegin
ALTER TABLE feeds ADD COLUMN css_sel_container TEXT NOT NULL DEFAULT '';
ALTER TABLE feeds ADD COLUMN css_sel_start  TEXT NOT NULL DEFAULT '';
ALTER TABLE feeds ADD COLUMN css_sel_stop  TEXT NOT NULL DEFAULT '';
ALTER TABLE feeds ADD COLUMN html_extraction_strategy  TEXT NOT NULL DEFAULT '';
-- +goose StatementEnd


