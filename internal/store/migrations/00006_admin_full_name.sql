-- +goose Up
ALTER TABLE admin_user ADD COLUMN full_name text NOT NULL DEFAULT '';

-- +goose Down
ALTER TABLE admin_user DROP COLUMN full_name;
