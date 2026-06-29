-- name: ListTemplatesForOwner :many
SELECT * FROM template
WHERE owner_id = $1 OR prebuilt = true
ORDER BY prebuilt DESC, created_at DESC;

-- name: ListTemplatesForOwnerByCategory :many
SELECT * FROM template
WHERE (owner_id = $1 OR prebuilt = true) AND category = $2
ORDER BY prebuilt DESC, created_at DESC;

-- name: CreateTemplate :one
INSERT INTO template (id, owner_id, name, category, body_json, prebuilt)
VALUES ($1, $2, $3, $4, $5, false)
RETURNING *;

-- name: GetTemplateForUse :one
SELECT * FROM template
WHERE id = $1 AND (owner_id = $2 OR prebuilt = true);

-- name: DeleteTemplateForOwner :exec
DELETE FROM template
WHERE id = $1 AND owner_id = $2 AND prebuilt = false;
