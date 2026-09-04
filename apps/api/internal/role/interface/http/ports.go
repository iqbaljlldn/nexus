package http

import (
	"context"

	"github.com/google/uuid"
	roleDomain "github.com/iqbaljlldn/nexus/apps/api/internal/role/domain"
)

type PermissionResolver interface {
	Resolve(ctx context.Context, userID, workspaceID, channelID uuid.UUID, required roleDomain.PermissionFlag) (bool, error)
}
