-- +goose Up
CREATE TABLE workspace (
    id         uuid PRIMARY KEY,
    name       text NOT NULL DEFAULT 'My workspace',
    created_at timestamptz NOT NULL DEFAULT now()
);

ALTER TABLE admin_user ADD COLUMN workspace_id uuid;
ALTER TABLE admin_user ADD COLUMN role text NOT NULL DEFAULT 'owner';

-- Backfill existing admins: one workspace per admin, workspace.id = admin.id,
-- so workspace.id == admin.id == every existing owner_id (zero row migration).
INSERT INTO workspace (id, name) SELECT id, 'My workspace' FROM admin_user;
UPDATE admin_user SET workspace_id = id WHERE workspace_id IS NULL;

-- Now that every existing row is backfilled, require it + add the FK so sqlc
-- emits a clean uuid.UUID (not a nullable type). On a fresh DB the table is
-- empty here, so SET NOT NULL is a no-op and EnsureAdmin sets workspace_id on insert.
ALTER TABLE admin_user ALTER COLUMN workspace_id SET NOT NULL;
ALTER TABLE admin_user ADD CONSTRAINT fk_admin_user_workspace
    FOREIGN KEY (workspace_id) REFERENCES workspace(id);

CREATE TABLE invite (
    id           uuid PRIMARY KEY,
    workspace_id uuid NOT NULL REFERENCES workspace(id) ON DELETE CASCADE,
    email        text NOT NULL,
    role         text NOT NULL DEFAULT 'editor',
    token        text NOT NULL,
    expires_at   timestamptz NOT NULL,
    accepted_at  timestamptz,
    created_at   timestamptz NOT NULL DEFAULT now()
);
CREATE UNIQUE INDEX ux_invite_token ON invite (token);
CREATE INDEX ix_invite_workspace ON invite (workspace_id);
CREATE INDEX ix_admin_user_workspace ON admin_user (workspace_id);

-- +goose Down
DROP TABLE IF EXISTS invite;
ALTER TABLE admin_user DROP COLUMN role;
ALTER TABLE admin_user DROP COLUMN workspace_id;
DROP TABLE IF EXISTS workspace;
