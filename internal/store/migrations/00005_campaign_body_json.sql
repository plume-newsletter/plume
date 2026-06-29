-- +goose Up
ALTER TABLE campaign ADD COLUMN body_json jsonb NOT NULL DEFAULT '[]';

-- +goose Down
ALTER TABLE campaign DROP COLUMN body_json;
