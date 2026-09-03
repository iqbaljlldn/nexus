package domain

import (
	"context"

	"github.com/google/uuid"
)

type RoleRepository interface {
	// Create persists a new role. The DB-generated ID is set on the
	// role pointer upon success.
	Create(ctx context.Context, role *Role) error
	// GetByID retrieves a role by its ID.
	GetByID(ctx context.Context, id uuid.UUID) (*Role, error)
	// Assign creates a member-role assignment.
	Assign(ctx context.Context, memberID, roleID uuid.UUID) error
	// GetEveryoneRole retrieves the @everyone role for a workspace.
	GetEveryoneRole(ctx context.Context, workspaceID uuid.UUID) (*Role, error)
	// GetMaxPosition returns the highest position value among roles in
	// a workspace. Returns 0 if the workspace has no roles.
	GetMaxPosition(ctx context.Context, workspaceID uuid.UUID) (int32, error)
	// ListAssignmentsByMember returns all role assignments for a member.
	ListAssignmentsByMember(ctx context.Context, memberID uuid.UUID) ([]RoleAssignment, error)
	// DeleteAssignmentsByMember removes all role assignments for a member.
	// Used for replace-all semantics on PATCH role assignment.
	DeleteAssignmentsByMember(ctx context.Context, memberID uuid.UUID) error
	// FindMemberRolesSortedByPosition returns a member's roles, sorted by position descending.
	FindMemberRolesSortedByPosition(ctx context.Context, memberID uuid.UUID) ([]*Role, error)
}
