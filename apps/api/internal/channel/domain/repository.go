package domain

import (
	"context"

	"github.com/google/uuid"
)

type ChannelRepository interface {
	Create(ctx context.Context, channel *Channel) error
	GetByID(ctx context.Context, id uuid.UUID) (*Channel, error)
	ListByWorkspaceID(ctx context.Context, workspaceID uuid.UUID) ([]Channel, error)
	GetMaxPosition(ctx context.Context, workspaceID uuid.UUID) (int32, error)

	// Permission Overrides
	CreatePermissionOverride(ctx context.Context, override *ChannelPermissionOverride) error
	GetPermissionOverrides(ctx context.Context, channelID uuid.UUID) ([]ChannelPermissionOverride, error)
	GetChannelPermissionOverrideByRole(ctx context.Context, channelID, roleID uuid.UUID) (*ChannelPermissionOverride, error)
	GetChannelPermissionOverrideByMember(ctx context.Context, channelID, memberID uuid.UUID) (*ChannelPermissionOverride, error)
	UpdatePermissionOverride(ctx context.Context, id uuid.UUID, allowBitmask, denyBitmask int64) error
	DeletePermissionOverride(ctx context.Context, id uuid.UUID) error

	// External validation
	GetCategoryWorkspaceID(ctx context.Context, categoryID uuid.UUID) (uuid.UUID, error)
}
