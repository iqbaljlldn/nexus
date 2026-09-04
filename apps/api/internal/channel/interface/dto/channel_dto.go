package dto

import (
	"time"

	"github.com/google/uuid"
	"github.com/iqbaljlldn/nexus/apps/api/internal/channel/domain"
)

type CreateChannelRequest struct {
	Name       string     `json:"name" binding:"required,max=100"`
	Type       string     `json:"type" binding:"required,oneof=text"` // Only text is supported for now
	CategoryID *uuid.UUID `json:"category_id,omitempty"`
}

type PatchPermissionOverrideRequest struct {
	RoleID       *uuid.UUID `json:"role_id"`
	MemberID     *uuid.UUID `json:"member_id"`
	AllowBitmask int64      `json:"allow_bitmask"`
	DenyBitmask  int64      `json:"deny_bitmask"`
}

type ChannelResponse struct {
	ID          uuid.UUID  `json:"id"`
	WorkspaceID *uuid.UUID `json:"workspace_id,omitempty"`
	CategoryID  *uuid.UUID `json:"category_id,omitempty"`
	Type        string     `json:"type"`
	Name        *string    `json:"name,omitempty"`
	Position    int32      `json:"position"`
	CreatedAt   time.Time  `json:"created_at"`
}

func FromChannel(ch *domain.Channel) *ChannelResponse {
	return &ChannelResponse{
		ID:          ch.ID,
		WorkspaceID: ch.WorkspaceID,
		CategoryID:  ch.CategoryID,
		Type:        string(ch.Type),
		Name:        ch.Name,
		Position:    ch.Position,
		CreatedAt:   ch.CreatedAt,
	}
}
