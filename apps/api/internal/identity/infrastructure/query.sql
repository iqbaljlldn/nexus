-- name: CreateUser :one
INSERT INTO users (
    id, email, username, display_name, password_hash, created_at, updated_at
) VALUES (
    $1, $2, $3, $4, $5, $6, $7
)
RETURNING *;

-- name: FindUserByEmailOrUsername :one
SELECT * FROM users
WHERE (email = $1 OR username = $2) AND deleted_at IS NULL
LIMIT 1;

-- name: CreateSession :one
INSERT INTO sessions (
    user_id, refresh_token_hash, user_agent, ip_address, expires_at
) VALUES (
    $1, $2, $3, $4, $5
)
RETURNING *;
