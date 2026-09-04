-- +goose Up
-- +goose StatementBegin
CREATE TABLE log (
    id           INTEGER PRIMARY KEY AUTOINCREMENT,
    time_ran DATETIME NOT NULL,
    run_type TEXT NOT NULL,
    articles_created INTEGER NOT NULL
);
-- +goose StatementEnd