-- name: CreateUser :one
INSERT INTO users (
    id, email, username, display_name, password_hash, created_at, updated_at
) VALUES (
    $1, $2, $3, $4, $5, $6, $7
)
RETURNING *;

-- name: FindUserByEmail :one
SELECT * FROM users
WHERE email = $1 AND deleted_at IS NULL
LIMIT 1;

-- name: FindUserByUsername :one
SELECT * FROM users
WHERE username = $1 AND deleted_at IS NULL
LIMIT 1;

-- name: CreateSession :one
INSERT INTO sessions (
    user_id, refresh_token_hash, user_agent, ip_address, expires_at
) VALUES (
    $1, $2, $3, $4, $5
)
RETURNING *;

-- name: RevokeSession :one
UPDATE sessions
SET 
    deleted_at = now(),
    updated_at = now(),
    status = 'revoked'
WHERE
    refresh_token_hash = $1
    AND deleted_at IS NULL
RETURNING *;

-- name: FindSessionByTokenHash :one
SELECT * FROM sessions
WHERE
    refresh_token_hash = $1
    AND deleted_at IS NULL
LIMIT 1;

-- name: FindActiveSessionsByUserId :many
SELECT *
FROM sessions
WHERE
    user_id = $1
    AND deleted_at IS NULL
    AND status = 'active';

-- name: RevokeAllSessionsByUserId :exec
UPDATE sessions
SET 
    deleted_at = now(),
    updated_at = now(),
    status = 'revoked'
WHERE
    user_id = $1
    AND deleted_at IS NULL
    AND status = 'active';

-- name: RevokeSessionById :one
UPDATE sessions
SET
    deleted_at = now(),
    updated_at = now(),
    status = 'revoked'
WHERE
    id = $1
    AND deleted_at IS NULL
    AND status = 'active'
RETURNING *;

-- name: FindSessionById :one
SELECT *
FROM sessions
WHERE
    id = $1
    AND deleted_at IS NULL
    AND status = 'active'
LIMIT 1;
