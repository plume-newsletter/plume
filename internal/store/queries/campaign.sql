-- name: CreateCampaign :one
INSERT INTO campaign (id, owner_id, brand_id, subject, html_body, plain_body, status)
VALUES ($1, $2, $3, $4, $5, $6, 'draft') RETURNING *;

-- name: ListCampaignsByOwner :many
SELECT * FROM campaign WHERE owner_id = $1 ORDER BY created_at;

-- name: GetCampaignForOwner :one
SELECT * FROM campaign WHERE id = $1 AND owner_id = $2;

-- name: UpdateCampaign :one
UPDATE campaign SET subject = $3, html_body = $4, plain_body = $5, body_json = $6
WHERE id = $1 AND owner_id = $2 RETURNING *;

-- name: SetCampaignStatus :one
UPDATE campaign SET status = $3 WHERE id = $1 AND owner_id = $2 RETURNING *;

-- name: DeleteCampaign :exec
DELETE FROM campaign WHERE id = $1 AND owner_id = $2;
