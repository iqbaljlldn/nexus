-- name: CreateMember :one
INSERT INTO members (workspace_id, user_id, nickname)
VALUES ($1, $2, $3)
RETURNING *;

-- name: FindMemberByWorkspaceAndUser :one
SELECT * FROM members
WHERE workspace_id = $1 AND user_id = $2;

-- name: ListMembersByWorkspace :many
SELECT * FROM members
WHERE workspace_id = $1 AND (sqlc.narg('cursor')::uuid IS NULL OR id > sqlc.narg('cursor')::uuid)
ORDER BY id ASC
LIMIT $2;
