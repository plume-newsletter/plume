-- +goose Up
CREATE TABLE api_key (
    id           uuid PRIMARY KEY,
    workspace_id uuid NOT NULL,
    name         text NOT NULL,
    prefix       text NOT NULL,            -- first chars of the key, shown for identification
    hash         text NOT NULL,            -- sha256 of the full key; the key itself is never stored
    created_at   timestamptz NOT NULL DEFAULT now(),
    last_used_at timestamptz
);
CREATE INDEX ix_api_key_workspace ON api_key (workspace_id);
CREATE UNIQUE INDEX ux_api_key_hash ON api_key (hash);

-- +goose Down
DROP TABLE IF EXISTS api_key;
