-- +goose Up
ALTER TABLE app_settings RENAME COLUMN pool TO bus;

-- +goose Down
ALTER TABLE app_settings RENAME COLUMN bus TO pool;
