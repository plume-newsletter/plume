-- +goose Up
CREATE TABLE ab_test (
    id           uuid PRIMARY KEY,
    owner_id     uuid NOT NULL,
    campaign_id  uuid NOT NULL REFERENCES campaign(id) ON DELETE CASCADE,
    list_id      uuid NOT NULL REFERENCES list(id) ON DELETE CASCADE,
    subject_a    text NOT NULL DEFAULT '',
    subject_b    text NOT NULL DEFAULT '',
    test_percent int  NOT NULL DEFAULT 20,
    status       text NOT NULL DEFAULT 'draft',   -- draft | running | complete
    winner       text NOT NULL DEFAULT '',        -- '' | a | b
    created_at   timestamptz NOT NULL DEFAULT now()
);
CREATE INDEX ix_ab_test_owner ON ab_test (owner_id);

ALTER TABLE campaign_recipient ADD COLUMN variant text NOT NULL DEFAULT '';  -- '' | a | b
ALTER TABLE campaign_recipient ADD COLUMN subject text NOT NULL DEFAULT '';  -- per-recipient subject override

-- +goose Down
ALTER TABLE campaign_recipient DROP COLUMN subject;
ALTER TABLE campaign_recipient DROP COLUMN variant;
DROP TABLE IF EXISTS ab_test;
