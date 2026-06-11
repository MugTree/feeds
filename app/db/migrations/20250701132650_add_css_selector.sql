-- +goose Up
-- +goose StatementBegin
ALTER TABLE feeds ADD COLUMN css_sel_container TEXT DEFAULT '';
ALTER TABLE feeds ADD COLUMN css_sel_start  TEXT DEFAULT '';
ALTER TABLE feeds ADD COLUMN css_sel_stop  TEXT DEFAULT '';
ALTER TABLE feeds ADD COLUMN html_extraction_strategy  TEXT DEFAULT '';
-- +goose StatementEnd


