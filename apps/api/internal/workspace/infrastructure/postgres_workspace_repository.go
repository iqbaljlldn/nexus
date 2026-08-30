package infrastructure

import (
	"context"
	"database/sql"

	"github.com/iqbaljlldn/nexus/apps/api/internal/platform/txcontext"
	"github.com/iqbaljlldn/nexus/apps/api/internal/workspace/domain"
)

type PostgresWorkspaceRepository struct {
	db *sql.DB
}

func NewPostgresWorkspaceRepository(db *sql.DB) domain.WorkspaceRepository {
	return &PostgresWorkspaceRepository{db: db}
}

func (r *PostgresWorkspaceRepository) Create(ctx context.Context, workspace *domain.Workspace) error {
	dbtx := txcontext.ExtractDBTX(ctx, r.db)

	params := CreateWorkspaceParams{
		OwnerID: workspace.OwnerID,
		Name:    workspace.Name,
		IconUrl: sql.NullString{String: workspace.IconURL, Valid: workspace.IconURL != ""},
	}

	dbWorkspace, err := New(dbtx).CreateWorkspace(ctx, params)
	if err != nil {
		return err
	}

	workspace.ID = dbWorkspace.ID
	workspace.CreatedAt = dbWorkspace.CreatedAt
	workspace.UpdatedAt = dbWorkspace.UpdatedAt

	return nil
}
