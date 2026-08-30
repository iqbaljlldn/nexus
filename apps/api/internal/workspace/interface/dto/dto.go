package dto

import (
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
