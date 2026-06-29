-- name: GetListByID :one
SELECT * FROM list WHERE id = $1;
