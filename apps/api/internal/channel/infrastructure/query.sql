-- name: CreateChannel :one
INSERT INTO channels (
    workspace_id, category_id, type, name, position
) VALUES (
    $1, $2, $3, $4, $5
) RETURNING *;

-- name: GetChannelByID :one
SELECT * FROM channels
WHERE id = $1 AND deleted_at IS NULL;

-- name: ListChannelsByWorkspaceID :many
SELECT * FROM channels
WHERE workspace_id = $1 AND deleted_at IS NULL
ORDER BY position ASC, created_at ASC;

-- name: UpdateChannelName :one
UPDATE channels
SET name = $2
WHERE id = $1 AND deleted_at IS NULL
RETURNING *;

-- name: UpdateChannelPosition :one
UPDATE channels
SET position = $2
WHERE id = $1 AND deleted_at IS NULL
RETURNING *;

-- name: SoftDeleteChannel :exec
UPDATE channels
SET deleted_at = now()
WHERE id = $1 AND deleted_at IS NULL;

-- name: CreateChannelPermissionOverride :one
INSERT INTO channel_permission_overrides (
    channel_id, role_id, member_id, allow_bitmask, deny_bitmask
) VALUES (
    $1, $2, $3, $4, $5
) RETURNING *;

-- name: GetChannelPermissionOverrides :many
SELECT * FROM channel_permission_overrides
WHERE channel_id = $1;

-- name: UpdateChannelPermissionOverride :one
UPDATE channel_permission_overrides
SET allow_bitmask = $2, deny_bitmask = $3
WHERE id = $1
RETURNING *;

-- name: DeleteChannelPermissionOverride :exec
DELETE FROM channel_permission_overrides
WHERE id = $1;

-- name: GetMaxChannelPosition :one
SELECT COALESCE(MAX(position), 0)::int
FROM channels
WHERE workspace_id = $1 AND deleted_at IS NULL;

-- name: GetCategoryWorkspaceID :one
SELECT workspace_id FROM categories WHERE id = $1;

-- name: GetChannelPermissionOverrideByRole :one
SELECT * FROM channel_permission_overrides
WHERE channel_id = $1 AND role_id = $2;

-- name: GetChannelPermissionOverrideByMember :one
SELECT * FROM channel_permission_overrides
WHERE channel_id = $1 AND member_id = $2;
