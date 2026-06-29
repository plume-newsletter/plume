-- name: CreateSegment :one
INSERT INTO segment (id, owner_id, name, match, conditions)
VALUES ($1, $2, $3, $4, $5) RETURNING *;

-- name: ListSegmentsByOwner :many
SELECT * FROM segment WHERE owner_id = $1 ORDER BY created_at DESC;

-- name: GetSegmentForOwner :one
SELECT * FROM segment WHERE id = $1 AND owner_id = $2;

-- name: UpdateSegment :one
UPDATE segment SET name = $3, match = $4, conditions = $5
WHERE id = $1 AND owner_id = $2 RETURNING *;

-- name: DeleteSegment :exec
DELETE FROM segment WHERE id = $1 AND owner_id = $2;

-- name: ListCustomFieldNamesForOwner :many
SELECT DISTINCT f.name FROM custom_field f
JOIN list l ON l.id = f.list_id
WHERE l.owner_id = $1 ORDER BY f.name;
