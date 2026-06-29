-- name: CreateSubscriber :one
INSERT INTO subscriber (id, owner_id, list_id, email, name, status)
VALUES ($1, $2, $3, $4, $5, $6) RETURNING *;

-- name: GetSubscriberInListByEmail :one
SELECT * FROM subscriber WHERE list_id = $1 AND email = $2;

-- name: ListSubscribersInList :many
SELECT * FROM subscriber WHERE list_id = $1 AND owner_id = $2 ORDER BY created_at;

-- name: UpdateSubscriberStatus :one
UPDATE subscriber SET status = $3 WHERE id = $1 AND owner_id = $2 RETURNING *;

-- name: DeleteSubscriber :exec
DELETE FROM subscriber WHERE id = $1 AND owner_id = $2;

-- name: CreateCustomField :one
INSERT INTO custom_field (id, owner_id, list_id, name) VALUES ($1, $2, $3, $4) RETURNING *;

-- name: ListCustomFieldsForList :many
SELECT * FROM custom_field WHERE list_id = $1 AND owner_id = $2 ORDER BY created_at;

-- name: UpsertFieldValue :exec
INSERT INTO subscriber_field_value (id, subscriber_id, custom_field_id, value)
VALUES ($1, $2, $3, $4)
ON CONFLICT (subscriber_id, custom_field_id) DO UPDATE SET value = EXCLUDED.value;
