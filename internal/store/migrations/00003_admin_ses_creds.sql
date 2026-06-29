-- +goose Up
ALTER TABLE admin_user
    ADD COLUMN ses_access_key_id     text NOT NULL DEFAULT '',
    ADD COLUMN ses_secret_access_key text NOT NULL DEFAULT '',
    ADD COLUMN ses_region            text NOT NULL DEFAULT '';

-- +goose Down
ALTER TABLE admin_user
    DROP COLUMN ses_access_key_id,
    DROP COLUMN ses_secret_access_key,
    DROP COLUMN ses_region;
