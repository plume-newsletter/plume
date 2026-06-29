-- name: CreateUser :one
INSERT INTO admin_user (id, email, password_hash, full_name, role, workspace_id, timezone)
VALUES ($1, $2, $3, $4, $5, $6, 'UTC') RETURNING *;

-- name: ListUsersByWorkspace :many
SELECT * FROM admin_user WHERE workspace_id = $1 ORDER BY created_at;

-- name: UpdateUserRole :one
UPDATE admin_user SET role = $3 WHERE id = $1 AND workspace_id = $2 RETURNING *;

-- name: DeleteUser :exec
DELETE FROM admin_user WHERE id = $1 AND workspace_id = $2;

-- name: CountOwners :one
SELECT count(*) FROM admin_user WHERE workspace_id = $1 AND role = 'owner';

-- name: CreateWorkspace :one
INSERT INTO workspace (id, name) VALUES ($1, $2) RETURNING *;

-- name: GetWorkspace :one
SELECT * FROM workspace WHERE id = $1;

-- name: UpdateWorkspaceName :exec
UPDATE workspace SET name = $2 WHERE id = $1;

-- name: CreateInvite :one
INSERT INTO invite (id, workspace_id, email, role, token, expires_at)
VALUES ($1, $2, $3, $4, $5, $6) RETURNING *;

-- name: GetInviteByToken :one
SELECT * FROM invite WHERE token = $1;

-- name: ListInvitesByWorkspace :many
SELECT * FROM invite WHERE workspace_id = $1 AND accepted_at IS NULL ORDER BY created_at DESC;

-- name: DeleteInvite :exec
DELETE FROM invite WHERE id = $1 AND workspace_id = $2;

-- name: MarkInviteAccepted :exec
UPDATE invite SET accepted_at = now() WHERE id = $1;
