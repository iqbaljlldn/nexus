package domain

import (
	"time"

	"github.com/google/uuid"
)

// PermissionFlag represents a single permission bit.
// Flags are defined per-sprint as needed (YAGNI). Sprint 3 flags below;
// additional flags (e.g. MENTION_EVERYONE, SHARE_SCREEN) added in future sprints.
type PermissionFlag int64

const (
	PermSendMessages    PermissionFlag = 1 << iota // 1
	PermManageWorkspace                            // 2
	PermManageRoles                                // 4
	PermManageChannels                             // 8
	PermManageInvites                              // 16
	PermManageMessages                             // 32
	PermKickMembers                                // 64
	PermBanMembers                                 // 128
)

// DefaultEveryonePermissions is the bitmask applied to the auto-created
// @everyone role when a new workspace is created.
const DefaultEveryonePermissions int64 = int64(PermSendMessages)

// Has reports whether the bitmask includes the given flag.
func (p PermissionFlag) Has(flag PermissionFlag) bool {
	return p&flag == flag
}

// Add returns a new bitmask with the given flag set.
func (p PermissionFlag) Add(flag PermissionFlag) PermissionFlag {
	return p | flag
}

// Remove returns a new bitmask with the given flag cleared.
func (p PermissionFlag) Remove(flag PermissionFlag) PermissionFlag {
	return p &^ flag
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
