-- +goose Up
CREATE TABLE admin_user (
    id            uuid PRIMARY KEY,
    email         text NOT NULL,
    password_hash text NOT NULL,
    timezone      text NOT NULL DEFAULT 'UTC',
    created_at    timestamptz NOT NULL DEFAULT now()
);
CREATE UNIQUE INDEX ux_admin_user_email ON admin_user (email);

CREATE TABLE brand (
    id         uuid PRIMARY KEY,
    owner_id   uuid NOT NULL,
    name       text NOT NULL,
    from_name  text NOT NULL DEFAULT '',
    from_email text NOT NULL DEFAULT '',
    reply_to   text NOT NULL DEFAULT '',
    created_at timestamptz NOT NULL DEFAULT now()
);

CREATE TABLE list (
    id         uuid PRIMARY KEY,
    owner_id   uuid NOT NULL,
    brand_id   uuid NOT NULL REFERENCES brand(id) ON DELETE CASCADE,
    name       text NOT NULL,
    created_at timestamptz NOT NULL DEFAULT now()
);

CREATE TABLE subscriber (
    id         uuid PRIMARY KEY,
    owner_id   uuid NOT NULL,
    list_id    uuid NOT NULL REFERENCES list(id) ON DELETE CASCADE,
    email      text NOT NULL,
    name       text NOT NULL DEFAULT '',
    status     text NOT NULL DEFAULT 'pending',
    created_at timestamptz NOT NULL DEFAULT now()
);
CREATE UNIQUE INDEX ux_subscriber_list_email ON subscriber (list_id, email);

CREATE TABLE custom_field (
    id         uuid PRIMARY KEY,
    owner_id   uuid NOT NULL,
    list_id    uuid NOT NULL REFERENCES list(id) ON DELETE CASCADE,
    name       text NOT NULL,
    created_at timestamptz NOT NULL DEFAULT now()
);

CREATE TABLE subscriber_field_value (
    id             uuid PRIMARY KEY,
    subscriber_id  uuid NOT NULL REFERENCES subscriber(id) ON DELETE CASCADE,
    custom_field_id uuid NOT NULL REFERENCES custom_field(id) ON DELETE CASCADE,
    value          text NOT NULL DEFAULT ''
);

CREATE TABLE campaign (
    id           uuid PRIMARY KEY,
    owner_id     uuid NOT NULL,
    brand_id     uuid NOT NULL REFERENCES brand(id) ON DELETE CASCADE,
    subject      text NOT NULL DEFAULT '',
    html_body    text NOT NULL DEFAULT '',
    plain_body   text NOT NULL DEFAULT '',
    status       text NOT NULL DEFAULT 'draft',
    scheduled_at timestamptz,
    created_at   timestamptz NOT NULL DEFAULT now()
);

CREATE TABLE link (
    id          uuid PRIMARY KEY,
    campaign_id uuid NOT NULL REFERENCES campaign(id) ON DELETE CASCADE,
    url         text NOT NULL
);

CREATE TABLE campaign_recipient (
    id            uuid PRIMARY KEY,
    campaign_id   uuid NOT NULL REFERENCES campaign(id) ON DELETE CASCADE,
    subscriber_id uuid NOT NULL REFERENCES subscriber(id) ON DELETE CASCADE,
    status        text NOT NULL DEFAULT 'queued',
    sent_at       timestamptz,
    error         text
);

CREATE TABLE email_event (
    id            uuid PRIMARY KEY,
    campaign_id   uuid NOT NULL REFERENCES campaign(id) ON DELETE CASCADE,
    subscriber_id uuid NOT NULL REFERENCES subscriber(id) ON DELETE CASCADE,
    link_id       uuid REFERENCES link(id) ON DELETE SET NULL,
    type          text NOT NULL,
    metadata      text,
    created_at    timestamptz NOT NULL DEFAULT now()
);

CREATE TABLE suppression_entry (
    id         uuid PRIMARY KEY,
    owner_id   uuid NOT NULL,
    email      text NOT NULL,
    reason     text NOT NULL DEFAULT '',
    created_at timestamptz NOT NULL DEFAULT now()
);
CREATE UNIQUE INDEX ux_suppression_owner_email ON suppression_entry (owner_id, email);

-- +goose Down
DROP TABLE IF EXISTS suppression_entry, email_event, campaign_recipient, link,
    campaign, subscriber_field_value, custom_field, subscriber, list, brand, admin_user;
