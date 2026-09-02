package dto

import (
	"time"

	"github.com/google/uuid"
	"github.com/iqbaljlldn/nexus/apps/api/internal/workspace/domain"
)

type CreateWorkspaceRequest struct {
	Name    string  `json:"name" binding:"required"`
	IconURL *string `json:"icon_url"`
}

type ListWorkspacesRequest struct {
	Limit    uint   `form:"limit" binding:"omitempty" default:"20"`
	Cursor   string `form:"cursor" binding:"omitempty"`
	Search   string `form:"search" binding:"omitempty"`
	SortMode string `form:"sort_mode" binding:"omitempty,oneof=newest name_asc"`
}

type ListWorkspacesResponse struct {
	Workspaces []domain.Workspace `json:"workspaces"`
}

type PaginationMeta struct {
	Total   uint64  `json:"total"`
	Limit   uint    `json:"limit"`
	Cursor  *string `json:"next_cursor"`
	HasMore bool    `json:"has_more"`
}

type CreateInviteReq struct {
	WorkspaceID uuid.UUID
	MaxUses     *int
	ExpiresAt   *time.Time
}

type CreateInviteRequest struct {
	MaxUses        *int `json:"max_uses" binding:"omitempty,min=1"`
	ExpiresInHours *int `json:"expires_in_hours" binding:"omitempty,min=1"`
}

type CreateInviteResponse struct {
	Code string `json:"code"`
	URL  string `json:"url"`
}

type RedeemInviteResponse struct {
	WorkspaceID string `json:"workspace_id"`
	MemberID    string `json:"member_id"`
}
