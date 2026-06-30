-- +goose Up
CREATE TABLE webhook_endpoint (
    id           uuid PRIMARY KEY,
    workspace_id uuid NOT NULL,
    url          text NOT NULL,
    secret       text NOT NULL,            -- HMAC signing secret; shown to the user to verify deliveries
    events       text[] NOT NULL DEFAULT '{}',
    active       boolean NOT NULL DEFAULT true,
    created_at   timestamptz NOT NULL DEFAULT now()
);
CREATE INDEX ix_webhook_workspace ON webhook_endpoint (workspace_id);

-- +goose Down
DROP TABLE IF EXISTS webhook_endpoint;
