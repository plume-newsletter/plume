-- name: CreateList :one
INSERT INTO list (id, owner_id, brand_id, name) VALUES ($1, $2, $3, $4) RETURNING *;

-- name: ListListsByOwner :many
SELECT * FROM list WHERE owner_id = $1 ORDER BY created_at;

-- name: GetListForOwner :one
SELECT * FROM list WHERE id = $1 AND owner_id = $2;

-- name: UpdateList :one
UPDATE list SET name = $3 WHERE id = $1 AND owner_id = $2 RETURNING *;

-- name: DeleteList :exec
DELETE FROM list WHERE id = $1 AND owner_id = $2;
