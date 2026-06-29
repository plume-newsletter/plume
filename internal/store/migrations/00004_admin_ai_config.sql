-- +goose Up
ALTER TABLE admin_user
    ADD COLUMN anthropic_api_key text NOT NULL DEFAULT '',
    ADD COLUMN ai_model          text NOT NULL DEFAULT '';

-- +goose Down
ALTER TABLE admin_user
    DROP COLUMN anthropic_api_key,
    DROP COLUMN ai_model;
