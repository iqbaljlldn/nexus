-- name: CreateRole :one
INSERT INTO roles (
    workspace_id,
    name,
    permission_bitmask,
    position,
    is_everyone
) VALUES (
    $1, $2, $3, $4, $5
)
RETURNING *;

-- name: DeleteRole :one
DELETE FROM roles WHERE id = $1 RETURNING id;

-- name: UpdateRole :one
UPDATE roles SET name = $1, permission_bitmask = $2, position = $3, is_everyone = $4 WHERE id = $5 RETURNING *;

-- name: GetRoleByID :one
SELECT *
FROM roles
WHERE id = $1;

-- name: ListRolesByWorkspace :many
SELECT *
FROM roles
WHERE workspace_id = $1
ORDER BY position DESC
LIMIT $2 OFFSET $3;

-- name: AssignRoleToMember :one
INSERT INTO member_role_assignments (member_id, role_id) VALUES ($1, $2) RETURNING *;

-- name: RemoveRoleFromMember :one
DELETE FROM member_role_assignments WHERE member_id = $1 AND role_id = $2 RETURNING *;

-- name: GetEveryoneRoleByWorkspaceID :one
SELECT *
FROM roles
WHERE workspace_id = $1 AND is_everyone = true;

