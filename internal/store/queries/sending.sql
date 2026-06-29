-- name: CreateLink :one
INSERT INTO link (id, campaign_id, url) VALUES ($1, $2, $3) RETURNING *;

-- name: ListLinksForCampaign :many
SELECT * FROM link WHERE campaign_id = $1;

-- name: ListActiveSubscriberIDsInList :many
SELECT id FROM subscriber WHERE list_id = $1 AND status = 'active';

-- name: CreateRecipient :one
INSERT INTO campaign_recipient (id, campaign_id, subscriber_id, status)
VALUES ($1, $2, $3, 'queued') RETURNING *;

-- NOTE: not an atomic claim — safe only with a single worker goroutine. For multiple workers, switch to UPDATE ... RETURNING with FOR UPDATE SKIP LOCKED.
-- name: ClaimQueuedRecipients :many
SELECT * FROM campaign_recipient WHERE status = 'queued' ORDER BY id LIMIT $1;

-- name: MarkRecipientSent :exec
UPDATE campaign_recipient SET status = 'sent', sent_at = now() WHERE id = $1;

-- name: MarkRecipientFailed :exec
UPDATE campaign_recipient SET status = 'failed', error = $2 WHERE id = $1;

-- name: CountQueuedForCampaign :one
SELECT count(*) FROM campaign_recipient WHERE campaign_id = $1 AND status = 'queued';

-- name: GetCampaignByID :one
SELECT * FROM campaign WHERE id = $1;

-- name: GetSubscriberByID :one
SELECT * FROM subscriber WHERE id = $1;

-- name: GetBrandByID :one
SELECT * FROM brand WHERE id = $1;

-- name: SetCampaignStatusByID :one
UPDATE campaign SET status = $2 WHERE id = $1 RETURNING *;
