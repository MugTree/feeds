-- +goose Up
-- +goose StatementBegin
ALTER TABLE feeds ADD COLUMN css_sel_container TEXT;
ALTER TABLE feeds ADD COLUMN css_sel_start TEXT;
ALTER TABLE feeds ADD COLUMN css_sel_stop TEXT;
ALTER TABLE feeds ADD COLUMN html_extraction_strategy TEXT;

-- +goose StatementEnd

-- +goose Down
-- +goose StatementBegin
ALTER TABLE feeds DROP COLUMN css_sel_container;
ALTER TABLE feeds DROP COLUMN css_sel_start;
ALTER TABLE feeds DROP COLUMN css_sel_stop;
ALTER TABLE feeds DROP COLUMN html_extraction_strategy;
-- +goose StatementEnd
