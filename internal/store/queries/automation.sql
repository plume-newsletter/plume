-- name: CreateAutomation :one
INSERT INTO automation (id, owner_id, name, list_id) VALUES ($1, $2, $3, $4) RETURNING *;

-- name: ListAutomationsByOwner :many
SELECT * FROM automation WHERE owner_id = $1 ORDER BY created_at DESC;

-- name: GetAutomationForOwner :one
SELECT * FROM automation WHERE id = $1 AND owner_id = $2;

-- name: GetAutomationByID :one
SELECT * FROM automation WHERE id = $1;

-- name: UpdateAutomation :one
UPDATE automation SET name = $3, list_id = $4 WHERE id = $1 AND owner_id = $2 RETURNING *;

-- name: DeleteAutomation :exec
DELETE FROM automation WHERE id = $1 AND owner_id = $2;

-- name: SetAutomationStatus :exec
UPDATE automation SET status = $3 WHERE id = $1 AND owner_id = $2;

-- name: ListLiveAutomationsForList :many
SELECT * FROM automation WHERE list_id = $1 AND owner_id = $2 AND status = 'live';

-- name: CreateStep :exec
INSERT INTO automation_step (id, automation_id, position, kind, subject, html, wait_days)
VALUES ($1, $2, $3, $4, $5, $6, $7);

-- name: ListStepsForAutomation :many
SELECT * FROM automation_step WHERE automation_id = $1 ORDER BY position;

-- name: DeleteStepsForAutomation :exec
DELETE FROM automation_step WHERE automation_id = $1;

-- name: CreateEnrollment :exec
INSERT INTO automation_enrollment (id, automation_id, subscriber_id) VALUES ($1, $2, $3)
ON CONFLICT (automation_id, subscriber_id) DO NOTHING;

-- name: ClaimDueEnrollments :many
SELECT e.* FROM automation_enrollment e
JOIN automation a ON a.id = e.automation_id
WHERE e.status = 'active' AND e.next_run_at <= now() AND a.status = 'live'
ORDER BY e.next_run_at LIMIT $1;

-- name: AdvanceEnrollment :exec
UPDATE automation_enrollment SET step_index = $2, next_run_at = $3 WHERE id = $1;

-- name: MarkEnrollmentComplete :exec
UPDATE automation_enrollment SET status = 'complete' WHERE id = $1;

-- name: CountEnrollmentsByStatus :one
SELECT count(*) FROM automation_enrollment WHERE automation_id = $1 AND status = $2;
