-- name: GetOtpByUserId :one
SELECT * FROM otps
WHERE user_id = @user_id
AND consumed_at IS NULL
AND expires_at > CURRENT_TIMESTAMP
ORDER BY created_at DESC
LIMIT 1;

-- name: CreateOtp :exec
INSERT INTO otps (user_id, code_hash, expires_at)
VALUES (@user_id, @code_hash, @expires_at);

-- name: IncrementOtpAttempts :exec
UPDATE otps
SET attempts = attempts + 1
WHERE user_id = @user_id;

-- name: DeleteOtp :exec
DELETE FROM otps
WHERE id = @id;