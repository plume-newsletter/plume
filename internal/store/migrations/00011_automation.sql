-- +goose Up
CREATE TABLE automation (
    id           uuid PRIMARY KEY,
    owner_id     uuid NOT NULL,
    name         text NOT NULL,
    list_id      uuid NOT NULL REFERENCES list(id) ON DELETE CASCADE,
    trigger_type text NOT NULL DEFAULT 'list_subscribe',
    status       text NOT NULL DEFAULT 'draft',   -- draft | live | paused
    created_at   timestamptz NOT NULL DEFAULT now()
);
CREATE INDEX ix_automation_owner ON automation (owner_id);
CREATE INDEX ix_automation_list_status ON automation (list_id, status);

CREATE TABLE automation_step (
    id            uuid PRIMARY KEY,
    automation_id uuid NOT NULL REFERENCES automation(id) ON DELETE CASCADE,
    position      int  NOT NULL,
    kind          text NOT NULL,                  -- send | wait
    subject       text NOT NULL DEFAULT '',
    html          text NOT NULL DEFAULT '',
    wait_days     int  NOT NULL DEFAULT 0
);
CREATE INDEX ix_automation_step_auto ON automation_step (automation_id, position);

CREATE TABLE automation_enrollment (
    id            uuid PRIMARY KEY,
    automation_id uuid NOT NULL REFERENCES automation(id) ON DELETE CASCADE,
    subscriber_id uuid NOT NULL REFERENCES subscriber(id) ON DELETE CASCADE,
    step_index    int  NOT NULL DEFAULT 0,
    next_run_at   timestamptz NOT NULL DEFAULT now(),
    status        text NOT NULL DEFAULT 'active',  -- active | complete
    created_at    timestamptz NOT NULL DEFAULT now()
);
CREATE INDEX ix_enrollment_due ON automation_enrollment (status, next_run_at);
CREATE UNIQUE INDEX ux_enrollment_auto_sub ON automation_enrollment (automation_id, subscriber_id);

-- +goose Down
DROP TABLE IF EXISTS automation_enrollment;
DROP TABLE IF EXISTS automation_step;
DROP TABLE IF EXISTS automation;
