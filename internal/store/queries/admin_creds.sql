-- name: SetAdminSESCreds :exec
UPDATE admin_user SET ses_access_key_id = $2, ses_secret_access_key = $3, ses_region = $4 WHERE id = $1;

-- name: SetAdminAIConfig :exec
UPDATE admin_user SET anthropic_api_key = $2, ai_model = $3 WHERE id = $1;
