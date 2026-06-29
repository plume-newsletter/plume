-- name: CreateSignupForm :one
INSERT INTO signup_form (id, owner_id, list_id, name, heading, description, button_text)
VALUES ($1, $2, $3, $4, $5, $6, $7) RETURNING *;

-- name: ListSignupFormsByOwner :many
SELECT * FROM signup_form WHERE owner_id = $1 ORDER BY created_at DESC;

-- name: GetSignupFormForOwner :one
SELECT * FROM signup_form WHERE id = $1 AND owner_id = $2;

-- name: UpdateSignupForm :one
UPDATE signup_form SET list_id = $3, name = $4, heading = $5, description = $6, button_text = $7
WHERE id = $1 AND owner_id = $2 RETURNING *;

-- name: DeleteSignupForm :exec
DELETE FROM signup_form WHERE id = $1 AND owner_id = $2;

-- name: GetSignupForm :one
SELECT * FROM signup_form WHERE id = $1;
