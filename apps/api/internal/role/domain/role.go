package domain

import (
	"time"

	"github.com/google/uuid"
)

// PermissionFlag represents a single permission bit.
// TODO(task-3.4.2): Extend with remaining Sprint 3 flags: MANAGE_WORKSPACE,
// MANAGE_ROLES, MANAGE_CHANNELS, MANAGE_INVITES, MANAGE_MESSAGES,
// KICK_MEMBERS, BAN_MEMBERS. Add flags for future sprints as needed (YAGNI).
type PermissionFlag int64

const (
	PermSendMessages PermissionFlag = 1 << iota
	// Additional flags will be defined in Task 3.4.2
)

// DefaultEveryonePermissions is the bitmask applied to the auto-created
// @everyone role when a new workspace is created.
const DefaultEveryonePermissions int64 = int64(PermSendMessages)

// Has reports whether the bitmask includes the given flag.
func (p PermissionFlag) Has(flag PermissionFlag) bool {
	return p&flag == flag
}

type Role struct {
	ID                uuid.UUID
	WorkspaceID       uuid.UUID
	Name              string
	PermissionBitmask int64 // stored as BIGINT in DB (RULES.md §7)
	Position          int32
	IsEveryone        bool
	CreatedAt         time.Time
	UpdatedAt         time.Time
}

type RoleAssignment struct {
	MemberID uuid.UUID
	RoleID   uuid.UUID
}

func NewRole(workspaceID uuid.UUID, name string, permissionBitmask int64, position int32, isEveryone bool) *Role {
	return &Role{
		WorkspaceID:       workspaceID,
		Name:              name,
		PermissionBitmask: permissionBitmask,
		Position:          position,
		IsEveryone:        isEveryone,
		CreatedAt:         time.Now(),
		UpdatedAt:         time.Now(),
	}
}

// NewEveryoneRole creates the default @everyone role for a workspace.
func NewEveryoneRole(workspaceID uuid.UUID) *Role {
	return NewRole(workspaceID, "@everyone", DefaultEveryonePermissions, 0, true)
}
