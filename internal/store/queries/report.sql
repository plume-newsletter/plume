-- name: CountRecipients :one
SELECT count(*) FROM campaign_recipient WHERE campaign_id = $1;

-- name: CountRecipientsByStatus :one
SELECT count(*) FROM campaign_recipient WHERE campaign_id = $1 AND status = $2;

-- name: CountEventsByType :one
SELECT count(*) FROM email_event WHERE campaign_id = $1 AND type = $2;

-- name: CountDistinctSubscribersByEventType :one
SELECT count(DISTINCT subscriber_id) FROM email_event WHERE campaign_id = $1 AND type = $2;
