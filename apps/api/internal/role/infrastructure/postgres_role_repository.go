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
