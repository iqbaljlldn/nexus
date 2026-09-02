package infrastructure

import (
	"context"
	"database/sql"

	"github.com/google/uuid"
	"github.com/iqbaljlldn/nexus/apps/api/internal/platform/txcontext"
	"github.com/iqbaljlldn/nexus/apps/api/internal/role/domain"
)

type PostgresRoleRepository struct {
	db *sql.DB
}

func NewPostgresRoleRepository(db *sql.DB) domain.RoleRepository {
	return &PostgresRoleRepository{db: db}
}

func (r *PostgresRoleRepository) Create(ctx context.Context, role *domain.Role) error {
	dbtx := txcontext.ExtractDBTX(ctx, r.db)

	params := CreateRoleParams{
		WorkspaceID:       role.WorkspaceID,
		Name:              role.Name,
		PermissionBitmask: role.PermissionBitmask,
		Position:          role.Position,
		IsEveryone:        role.IsEveryone,
	}

	dbRole, err := New(dbtx).CreateRole(ctx, params)
	if err != nil {
		return err
	}

	role.ID = dbRole.ID
	role.CreatedAt = dbRole.CreatedAt
	role.UpdatedAt = dbRole.UpdatedAt

	return nil
}

func (r *PostgresRoleRepository) Assign(ctx context.Context, memberID, roleID uuid.UUID) error {
	dbtx := txcontext.ExtractDBTX(ctx, r.db)

	_, err := New(dbtx).AssignRoleToMember(ctx, AssignRoleToMemberParams{
		MemberID: memberID,
		RoleID:   roleID,
	})

	return err
}

func (r *PostgresRoleRepository) GetEveryoneRole(ctx context.Context, workspaceID uuid.UUID) (*domain.Role, error) {
	dbtx := txcontext.ExtractDBTX(ctx, r.db)

	dbRole, err := New(dbtx).GetEveryoneRoleByWorkspaceID(ctx, workspaceID)
	if err != nil {
		return nil, err
	}

	return &domain.Role{
		ID:                dbRole.ID,
		WorkspaceID:       dbRole.WorkspaceID,
		Name:              dbRole.Name,
		PermissionBitmask: dbRole.PermissionBitmask,
		Position:          dbRole.Position,
		IsEveryone:        dbRole.IsEveryone,
		CreatedAt:         dbRole.CreatedAt,
		UpdatedAt:         dbRole.UpdatedAt,
	}, nil
}

func (r *PostgresRoleRepository) GetByID(ctx context.Context, id uuid.UUID) (*domain.Role, error) {
	dbtx := txcontext.ExtractDBTX(ctx, r.db)

	dbRole, err := New(dbtx).GetRoleByID(ctx, id)
	if err != nil {
		return nil, err
	}

	return &domain.Role{
		ID:                dbRole.ID,
		WorkspaceID:       dbRole.WorkspaceID,
		Name:              dbRole.Name,
		PermissionBitmask: dbRole.PermissionBitmask,
		Position:          dbRole.Position,
		IsEveryone:        dbRole.IsEveryone,
		CreatedAt:         dbRole.CreatedAt,
		UpdatedAt:         dbRole.UpdatedAt,
	}, nil
}

func (r *PostgresRoleRepository) GetMaxPosition(ctx context.Context, workspaceID uuid.UUID) (int32, error) {
	dbtx := txcontext.ExtractDBTX(ctx, r.db)
	return New(dbtx).GetMaxRolePositionByWorkspace(ctx, workspaceID)
}

func (r *PostgresRoleRepository) ListAssignmentsByMember(ctx context.Context, memberID uuid.UUID) ([]domain.RoleAssignment, error) {
	dbtx := txcontext.ExtractDBTX(ctx, r.db)

	rows, err := New(dbtx).ListRoleAssignmentsByMember(ctx, memberID)
	if err != nil {
		return nil, err
	}

	assignments := make([]domain.RoleAssignment, len(rows))
	for i, row := range rows {
		assignments[i] = domain.RoleAssignment{
			MemberID: row.MemberID,
			RoleID:   row.RoleID,
		}
	}
	return assignments, nil
}

func (r *PostgresRoleRepository) DeleteAssignmentsByMember(ctx context.Context, memberID uuid.UUID) error {
	dbtx := txcontext.ExtractDBTX(ctx, r.db)
	return New(dbtx).DeleteAllRoleAssignmentsByMember(ctx, memberID)
}
