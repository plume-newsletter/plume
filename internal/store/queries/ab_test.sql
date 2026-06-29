-- name: CreateABTest :one
INSERT INTO ab_test (id, owner_id, campaign_id, list_id, subject_a, subject_b, test_percent)
VALUES ($1, $2, $3, $4, $5, $6, $7) RETURNING *;

-- name: ListABTestsByOwner :many
SELECT * FROM ab_test WHERE owner_id = $1 ORDER BY created_at DESC;

-- name: GetABTestForOwner :one
SELECT * FROM ab_test WHERE id = $1 AND owner_id = $2;

-- name: DeleteABTest :exec
DELETE FROM ab_test WHERE id = $1 AND owner_id = $2;

-- name: UpdateABTestStatus :exec
UPDATE ab_test SET status = $2 WHERE id = $1;

-- name: SetABTestWinner :exec
UPDATE ab_test SET status = 'complete', winner = $2 WHERE id = $1;

-- name: CreateRecipientVariant :one
INSERT INTO campaign_recipient (id, campaign_id, subscriber_id, status, variant, subject)
VALUES ($1, $2, $3, 'queued', $4, $5) RETURNING *;

-- name: CountVariantRecipients :one
SELECT count(*) FROM campaign_recipient WHERE campaign_id = $1 AND variant = $2;

-- name: CountVariantEvents :one
SELECT count(DISTINCT e.subscriber_id) FILTER (WHERE e.type = 'open')  AS opens,
       count(*)                        FILTER (WHERE e.type = 'click') AS clicks
FROM email_event e
JOIN campaign_recipient r ON r.campaign_id = e.campaign_id AND r.subscriber_id = e.subscriber_id
WHERE e.campaign_id = $1 AND r.variant = $2;

-- name: ListRecipientSubscriberIDs :many
SELECT subscriber_id FROM campaign_recipient WHERE campaign_id = $1;

-- name: ListRecipientsForCampaign :many
SELECT * FROM campaign_recipient WHERE campaign_id = $1;
