-- name: CreateApiKey :one
INSERT INTO api_key (id, workspace_id, name, prefix, hash)
VALUES ($1, $2, $3, $4, $5) RETURNING *;

-- name: ListApiKeysForOwner :many
SELECT id, name, prefix, created_at, last_used_at FROM api_key
WHERE workspace_id = $1 ORDER BY created_at DESC;

-- name: GetApiKeyByHash :one
SELECT id, workspace_id FROM api_key WHERE hash = $1;

-- name: TouchApiKey :exec
UPDATE api_key SET last_used_at = now() WHERE id = $1;

-- name: DeleteApiKey :exec
DELETE FROM api_key WHERE id = $1 AND workspace_id = $2;
