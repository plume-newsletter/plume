-- name: InsertEmailEvent :one
INSERT INTO email_event (id, campaign_id, subscriber_id, link_id, type, metadata)
VALUES ($1, $2, $3, $4, $5, $6) RETURNING *;

-- name: GetRecipientByID :one
SELECT * FROM campaign_recipient WHERE id = $1;

-- name: GetLinkByID :one
SELECT * FROM link WHERE id = $1;

-- name: InsertSuppression :exec
INSERT INTO suppression_entry (id, owner_id, email, reason)
VALUES ($1, $2, $3, $4)
ON CONFLICT (owner_id, email) DO UPDATE SET reason = EXCLUDED.reason;

-- name: SetSubscriberStatusByID :exec
UPDATE subscriber SET status = $2 WHERE id = $1;

-- name: ListSubscribersByEmail :many
SELECT * FROM subscriber WHERE email = $1;

-- name: ListEmailEventsForCampaign :many
SELECT * FROM email_event WHERE campaign_id = $1 ORDER BY created_at;

-- name: IsSuppressed :one
SELECT EXISTS(SELECT 1 FROM suppression_entry WHERE owner_id = $1 AND email = $2);

-- name: DeleteSuppressionByOwnerEmail :exec
DELETE FROM suppression_entry WHERE owner_id = $1 AND email = $2;

-- name: GetMostRecentCampaignIDForSubscriber :one
SELECT campaign_id FROM campaign_recipient
WHERE subscriber_id = $1
ORDER BY sent_at DESC NULLS LAST
LIMIT 1;
