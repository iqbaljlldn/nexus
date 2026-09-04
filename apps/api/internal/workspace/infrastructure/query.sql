-- name: CreateWorkspace :one
INSERT INTO workspaces (owner_id, name, icon_url)
VALUES ($1, $2, $3)
RETURNING *;

-- name: GetWorkspaceByID :one
SELECT * FROM workspaces
WHERE id = $1;

-- name: ListWorkspacesByNewest :many
SELECT w.* 
FROM workspaces w
JOIN members m ON w.id = m.workspace_id
WHERE m.user_id = $1 
  AND (sqlc.narg('search')::text IS NULL OR w.name ILIKE '%' || sqlc.narg('search')::text || '%')
  AND (
    sqlc.narg('cursor_created_at')::timestamptz IS NULL OR 
    (w.created_at, w.id) < (sqlc.narg('cursor_created_at')::timestamptz, sqlc.narg('cursor_id')::uuid)
  )
ORDER BY w.created_at DESC, w.id DESC
LIMIT $2;

-- name: ListWorkspacesByNameAsc :many
SELECT w.* 
FROM workspaces w
JOIN members m ON w.id = m.workspace_id
WHERE m.user_id = $1 
  AND (sqlc.narg('search')::text IS NULL OR w.name ILIKE '%' || sqlc.narg('search')::text || '%')
  AND (
    sqlc.narg('cursor_name')::text IS NULL OR 
    (w.name, w.id) > (sqlc.narg('cursor_name')::text, sqlc.narg('cursor_id')::uuid)
  )
ORDER BY w.name ASC, w.id ASC
LIMIT $2;

-- name: GetWorkspacesCountByUserID :one
SELECT count(*) 
FROM workspaces w
JOIN members m ON w.id = m.workspace_id
WHERE m.user_id = $1
  AND (sqlc.narg('search')::text IS NULL OR w.name ILIKE '%' || sqlc.narg('search')::text || '%');

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
