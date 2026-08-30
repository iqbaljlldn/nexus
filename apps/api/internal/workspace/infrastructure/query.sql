-- name: CreateWorkspace :one
INSERT INTO workspaces (owner_id, name, icon_url)
VALUES ($1, $2, $3)
RETURNING *;

-- name: ListWorkspacesByUserID :many
SELECT w.* 
FROM workspaces w
JOIN members m ON w.id = m.workspace_id
WHERE m.user_id = $1 AND (sqlc.narg('cursor')::uuid IS NULL OR w.id < sqlc.narg('cursor')::uuid)
ORDER BY w.id DESC
LIMIT $2;

-- name: CreateCategory :one
INSERT INTO categories (workspace_id, name)
VALUES ($1, $2)
RETURNING *;

-- name: CreateInvite :one
INSERT INTO invites (workspace_id, code, created_by, max_uses, expires_at)
VALUES ($1, $2, $3, $4, $5)
RETURNING *;

-- name: FindInviteByCode :one
SELECT * FROM invites
WHERE code = $1;

-- name: IncrementInviteUseCount :one
UPDATE invites
SET use_count = use_count + 1
WHERE id = $1 AND (max_uses IS NULL OR use_count < max_uses)
RETURNING *;
