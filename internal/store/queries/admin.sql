-- name: CountAdmins :one
SELECT count(*) FROM admin_user;

-- name: GetAdminByEmail :one
SELECT * FROM admin_user WHERE email = $1;

-- name: GetAdminByID :one
SELECT * FROM admin_user WHERE id = $1;

-- name: GetSingleAdmin :one
SELECT * FROM admin_user ORDER BY created_at LIMIT 1;
