-- name: CountActiveSubscribersForOwner :one
SELECT count(*) FROM subscriber WHERE owner_id = $1 AND status = 'active';

-- name: CountSubscribersCreatedSince :one
SELECT count(*) FROM subscriber WHERE owner_id = $1 AND created_at >= $2;

-- name: CountSentForOwnerSince :one
SELECT count(*) FROM campaign_recipient r
JOIN campaign c ON c.id = r.campaign_id
WHERE c.owner_id = $1 AND r.status = 'sent' AND r.sent_at >= sqlc.arg(sent_at)::timestamptz;

-- name: CountEventsForOwnerSince :one
SELECT count(*) FROM email_event e
JOIN campaign c ON c.id = e.campaign_id
WHERE c.owner_id = $1 AND e.type = $2 AND e.created_at >= $3;

-- name: CountDistinctOpenersForOwnerSince :one
SELECT count(DISTINCT e.subscriber_id) FROM email_event e
JOIN campaign c ON c.id = e.campaign_id
WHERE c.owner_id = $1 AND e.type = 'open' AND e.created_at >= $2;

-- name: SubscriberGainedByDay :many
SELECT date_trunc('day', created_at)::timestamptz AS day, count(*) AS n
FROM subscriber WHERE owner_id = $1 AND created_at >= $2
GROUP BY day ORDER BY day;

-- name: SubscriberLostByDay :many
SELECT date_trunc('day', e.created_at)::timestamptz AS day, count(*) AS n
FROM email_event e JOIN campaign c ON c.id = e.campaign_id
WHERE c.owner_id = $1 AND e.type = 'unsubscribe' AND e.created_at >= $2
GROUP BY day ORDER BY day;

-- name: SendVolumeByDay :many
SELECT date_trunc('day', r.sent_at)::timestamptz AS day, count(*) AS sent
FROM campaign_recipient r JOIN campaign c ON c.id = r.campaign_id
WHERE c.owner_id = $1 AND r.status = 'sent' AND r.sent_at >= sqlc.arg(sent_at)::timestamptz
GROUP BY day ORDER BY day;

-- name: OpensByDay :many
SELECT date_trunc('day', e.created_at)::timestamptz AS day, count(*) AS opens
FROM email_event e JOIN campaign c ON c.id = e.campaign_id
WHERE c.owner_id = $1 AND e.type = 'open' AND e.created_at >= $2
GROUP BY day ORDER BY day;

-- name: OpensByWeekdayHour :many
SELECT extract(dow from e.created_at)::int AS dow,
       extract(hour from e.created_at)::int AS hour,
       count(*) AS n
FROM email_event e JOIN campaign c ON c.id = e.campaign_id
WHERE c.owner_id = $1 AND e.type = 'open' AND e.created_at >= $2
GROUP BY dow, hour ORDER BY n DESC LIMIT 3;

-- name: CampaignsWithMetrics :many
SELECT c.id, c.subject, c.status, c.created_at,
  (SELECT count(*) FROM campaign_recipient r WHERE r.campaign_id = c.id AND r.status = 'sent') AS sent,
  (SELECT count(DISTINCT e.subscriber_id) FROM email_event e WHERE e.campaign_id = c.id AND e.type = 'open') AS unique_opens,
  (SELECT count(*) FROM email_event e WHERE e.campaign_id = c.id AND e.type = 'click') AS clicks
FROM campaign c
WHERE c.owner_id = $1
ORDER BY c.created_at DESC;

-- name: TopCampaignsByOpens :many
SELECT c.id, c.subject, count(DISTINCT e.subscriber_id) AS opens
FROM campaign c
JOIN email_event e ON e.campaign_id = c.id AND e.type = 'open' AND e.created_at >= $2
WHERE c.owner_id = $1
GROUP BY c.id, c.subject ORDER BY opens DESC LIMIT 4;

-- name: CountSuppressionsForOwner :one
SELECT count(*) FROM suppression_entry WHERE owner_id = $1;

-- name: RecentSuppressionsForOwner :many
SELECT email, reason, created_at FROM suppression_entry
WHERE owner_id = $1 ORDER BY created_at DESC LIMIT 100;
