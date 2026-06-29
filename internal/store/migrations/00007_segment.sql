-- +goose Up
CREATE TABLE segment (
    id         uuid PRIMARY KEY,
    owner_id   uuid NOT NULL,
    name       text NOT NULL,
    match      text NOT NULL DEFAULT 'all',
    conditions jsonb NOT NULL DEFAULT '[]',
    created_at timestamptz NOT NULL DEFAULT now()
);
CREATE INDEX ix_segment_owner ON segment (owner_id);

-- +goose Down
DROP TABLE IF EXISTS segment;
