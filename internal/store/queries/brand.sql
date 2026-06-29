-- name: CreateBrand :one
INSERT INTO brand (id, owner_id, name, from_name, from_email, reply_to)
VALUES ($1, $2, $3, $4, $5, $6)
RETURNING *;

-- name: GetBrand :one
SELECT * FROM brand WHERE id = $1;

-- name: ListBrandsByOwner :many
SELECT * FROM brand WHERE owner_id = $1 ORDER BY created_at;

-- name: GetBrandForOwner :one
SELECT * FROM brand WHERE id = $1 AND owner_id = $2;

-- name: UpdateBrand :one
UPDATE brand SET name = $3, from_name = $4, from_email = $5, reply_to = $6
WHERE id = $1 AND owner_id = $2
RETURNING *;

-- name: DeleteBrand :exec
DELETE FROM brand WHERE id = $1 AND owner_id = $2;
