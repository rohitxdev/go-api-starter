-- name: CreateUser :one
INSERT INTO users (email, username)
VALUES ($1 , $2)
RETURNING *;

-- name: GetUserByID :one
SELECT id, username
FROM users
WHERE id = $1;


-- name: GetUserByEmail :one
SELECT id, username, email, created_at, updated_at
FROM users
WHERE email = $1;

-- name: ListUsers :many
SELECT id, username, email, created_at, updated_at
FROM users
ORDER BY id
LIMIT $1 OFFSET $2;


-- name: UpdateUser :one
UPDATE users
SET username = COALESCE(NULLIF(@username, ''), username),
    email = COALESCE(NULLIF(@email, ''), email),
    verified_at = COALESCE(@verified_at, verified_at)
WHERE id = @id
RETURNING *;

-- name: DeleteUser :execresult
DELETE FROM users
WHERE id = $1;

-- name: UpsertUser :one
INSERT INTO users (email)
VALUES (@email)
ON CONFLICT (email) DO UPDATE
SET email = @email
RETURNING *;