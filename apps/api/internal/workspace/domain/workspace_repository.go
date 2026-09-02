package domain

import (
	"context"

	"github.com/google/uuid"
	"github.com/iqbaljlldn/nexus/pkg/pagination"
)

type WorkspaceRepository interface {
	// Create persists a new workspace. The DB-generated ID is set on the
	// workspace pointer upon success.
	Create(ctx context.Context, workspace *Workspace) error
	ListByNewest(ctx context.Context, userID uuid.UUID, search string, cursor *pagination.Cursor, limit uint) ([]Workspace, error)
	ListByNameAsc(ctx context.Context, userID uuid.UUID, search string, cursor *pagination.Cursor, limit uint) ([]Workspace, error)
	CountByUserID(ctx context.Context, userID uuid.UUID, search string) (uint64, error)
}
