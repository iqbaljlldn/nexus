package domain

import (
	"context"

	"github.com/google/uuid"
)

type RoleRepository interface {
	// Create persists a new role. The DB-generated ID is set on the
	// role pointer upon success.
	Create(ctx context.Context, role *Role) error
	// Assign creates a member-role assignment.
	Assign(ctx context.Context, memberID, roleID uuid.UUID) error
}
