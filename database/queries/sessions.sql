-- name: GetUserBySessionId :one
SELECT u.* FROM users AS u
JOIN sessions AS s ON u.id = s.user_id
WHERE s.id = @session_id;

-- name: CreateSession :one
INSERT INTO sessions(user_id,user_agent,ip_address,expires_at)
VALUES (@user_id,@user_agent,@ip_address,@expires_at)
RETURNING id;