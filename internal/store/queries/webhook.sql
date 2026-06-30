-- name: CreateWebhook :one
INSERT INTO webhook_endpoint (id, workspace_id, url, secret, events)
VALUES ($1, $2, $3, $4, $5) RETURNING *;

-- name: ListWebhooksForOwner :many
SELECT * FROM webhook_endpoint WHERE workspace_id = $1 ORDER BY created_at DESC;

-- name: ListActiveWebhooksForOwner :many
SELECT * FROM webhook_endpoint
WHERE workspace_id = $1 AND active = true AND sqlc.arg(event)::text = ANY(events);

-- name: DeleteWebhook :exec
DELETE FROM webhook_endpoint WHERE id = $1 AND workspace_id = $2;
