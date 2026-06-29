-- +goose Up
CREATE TABLE signup_form (
    id          uuid PRIMARY KEY,
    owner_id    uuid NOT NULL,
    list_id     uuid NOT NULL REFERENCES list(id) ON DELETE CASCADE,
    name        text NOT NULL,
    heading     text NOT NULL DEFAULT '',
    description text NOT NULL DEFAULT '',
    button_text text NOT NULL DEFAULT 'Subscribe',
    created_at  timestamptz NOT NULL DEFAULT now()
);
CREATE INDEX ix_signup_form_owner ON signup_form (owner_id);

-- +goose Down
DROP TABLE IF EXISTS signup_form;
