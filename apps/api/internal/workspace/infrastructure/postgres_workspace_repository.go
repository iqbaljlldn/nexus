package infrastructure

import (
	"context"
	"database/sql"

	"github.com/google/uuid"
	"github.com/iqbaljlldn/nexus/apps/api/internal/platform/txcontext"
	"github.com/iqbaljlldn/nexus/apps/api/internal/workspace/domain"
	"github.com/iqbaljlldn/nexus/pkg/pagination"
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

func (r *PostgresWorkspaceRepository) GetByID(ctx context.Context, id uuid.UUID) (*domain.Workspace, error) {
	dbtx := txcontext.ExtractDBTX(ctx, r.db)

	dbWorkspace, err := New(dbtx).GetWorkspaceByID(ctx, id)
	if err != nil {
		if err == sql.ErrNoRows {
			return nil, domain.ErrWorkspaceNotFound
		}
		return nil, err
	}

	return &domain.Workspace{
		ID:        dbWorkspace.ID,
		OwnerID:   dbWorkspace.OwnerID,
		Name:      dbWorkspace.Name,
		IconURL:   dbWorkspace.IconUrl.String,
		CreatedAt: dbWorkspace.CreatedAt,
		UpdatedAt: dbWorkspace.UpdatedAt,
	}, nil
}

func (r *PostgresWorkspaceRepository) ListByNewest(ctx context.Context, userID uuid.UUID, search string, cursor *pagination.Cursor, limit uint) ([]domain.Workspace, error) {
	dbtx := txcontext.ExtractDBTX(ctx, r.db)

	params := ListWorkspacesByNewestParams{
		UserID: userID,
		Limit:  int32(limit), //nolint:gosec // limit is capped at 100 in service layer
	}

	if search != "" {
		params.Search = sql.NullString{String: search, Valid: true}
	}

	if cursor != nil {
		createdAt, err := cursor.GetTimeValue()
		if err == nil {
			params.CursorCreatedAt = sql.NullTime{Time: createdAt, Valid: true}
			params.CursorID = uuid.NullUUID{UUID: cursor.LastID, Valid: true}
		}
	}

	rows, err := New(dbtx).ListWorkspacesByNewest(ctx, params)
	if err != nil {
		return nil, err
	}

	var workspaces []domain.Workspace
	for _, row := range rows {
		workspaces = append(workspaces, domain.Workspace{
			ID:        row.ID,
			OwnerID:   row.OwnerID,
			Name:      row.Name,
			IconURL:   row.IconUrl.String,
			CreatedAt: row.CreatedAt,
			UpdatedAt: row.UpdatedAt,
		})
	}
	return workspaces, nil
}

func (r *PostgresWorkspaceRepository) ListByNameAsc(ctx context.Context, userID uuid.UUID, search string, cursor *pagination.Cursor, limit uint) ([]domain.Workspace, error) {
	dbtx := txcontext.ExtractDBTX(ctx, r.db)

	params := ListWorkspacesByNameAscParams{
		UserID: userID,
		Limit:  int32(limit), //nolint:gosec // limit is capped at 100 in service layer
	}

	if search != "" {
		params.Search = sql.NullString{String: search, Valid: true}
	}

	if cursor != nil {
		nameVal, err := cursor.GetStringValue()
		if err == nil {
			params.CursorName = sql.NullString{String: nameVal, Valid: true}
			params.CursorID = uuid.NullUUID{UUID: cursor.LastID, Valid: true}
		}
	}

	rows, err := New(dbtx).ListWorkspacesByNameAsc(ctx, params)
	if err != nil {
		return nil, err
	}

	var workspaces []domain.Workspace
	for _, row := range rows {
		workspaces = append(workspaces, domain.Workspace{
			ID:        row.ID,
			OwnerID:   row.OwnerID,
			Name:      row.Name,
			IconURL:   row.IconUrl.String,
			CreatedAt: row.CreatedAt,
			UpdatedAt: row.UpdatedAt,
		})
	}
	return workspaces, nil
}

func (r *PostgresWorkspaceRepository) CountByUserID(ctx context.Context, userID uuid.UUID, search string) (uint64, error) {
	dbtx := txcontext.ExtractDBTX(ctx, r.db)

	params := GetWorkspacesCountByUserIDParams{
		UserID: userID,
	}
	if search != "" {
		params.Search = sql.NullString{String: search, Valid: true}
	}

	count, err := New(dbtx).GetWorkspacesCountByUserID(ctx, params)
	return uint64(count), err //nolint:gosec // count from DB won't be negative
}
