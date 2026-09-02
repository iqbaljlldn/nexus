package dto

import (
	"time"

	"github.com/google/uuid"
	"github.com/iqbaljlldn/nexus/apps/api/internal/role/domain"
)

type CreateRoleRequest struct {
	Name              string `json:"name" binding:"required,max=100"`
	PermissionBitmask int64  `json:"permission_bitmask"`
	Position          *int32 `json:"position" binding:"omitempty,min=0"`
}

type CreateRoleResponse struct {
	ID                uuid.UUID `json:"id"`
	WorkspaceID       uuid.UUID `json:"workspace_id"`
	Name              string    `json:"name"`
	PermissionBitmask int64     `json:"permission_bitmask"`
	Position          int32     `json:"position"`
	IsEveryone        bool      `json:"is_everyone"`
	CreatedAt         time.Time `json:"created_at"`
	UpdatedAt         time.Time `json:"updated_at"`
}

// FromRole converts a domain Role to a CreateRoleResponse DTO.
func FromRole(role *domain.Role) *CreateRoleResponse {
	return &CreateRoleResponse{
		ID:                role.ID,
		WorkspaceID:       role.WorkspaceID,
		Name:              role.Name,
		PermissionBitmask: role.PermissionBitmask,
		Position:          role.Position,
		IsEveryone:        role.IsEveryone,
		CreatedAt:         role.CreatedAt,
		UpdatedAt:         role.UpdatedAt,
	}
}

type AssignRolesRequest struct {
	RoleIDs []uuid.UUID `json:"role_ids" binding:"required"`
}
