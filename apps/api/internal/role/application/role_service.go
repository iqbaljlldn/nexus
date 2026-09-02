package application

import (
	"context"
	"fmt"

	"github.com/google/uuid"
	roleDomain "github.com/iqbaljlldn/nexus/apps/api/internal/role/domain"
	pkgerrors "github.com/iqbaljlldn/nexus/pkg/errors"
	"github.com/iqbaljlldn/nexus/pkg/logger"
	"go.uber.org/zap"
)

type RoleService struct {
	roleRepo  roleDomain.RoleRepository
	txManager TransactionManager
	log       *zap.Logger
}

func NewRoleService(
	roleRepo roleDomain.RoleRepository,
	txManager TransactionManager,
	log *zap.Logger,
) *RoleService {
	return &RoleService{
		roleRepo:  roleRepo,
		txManager: txManager,
		log:       log,
	}
}

// Create creates a new custom role in the workspace.
// If position is nil, it auto-assigns position = max(existing positions) + 1.
// The @everyone role cannot be created through this method (it is auto-created
// during workspace creation).
func (s *RoleService) Create(ctx context.Context, workspaceID uuid.UUID, name string, permissionBitmask int64, position *int32) (*roleDomain.Role, error) {
	log := logger.FromContext(ctx, s.log)

	if name == "" {
		return nil, &pkgerrors.DomainError{
			Code:    pkgerrors.CodeMissingRequiredField,
			Message: "Nama role wajib diisi.",
			Err:     fmt.Errorf("role name is required"),
		}
	}

	if len(name) > 100 {
		return nil, &pkgerrors.DomainError{
			Code:    pkgerrors.CodeInvalidFieldFormat,
			Message: "Nama role maksimal 100 karakter.",
			Err:     fmt.Errorf("role name exceeds 100 characters"),
		}
	}

	var finalPosition int32
	if position != nil {
		finalPosition = *position
	} else {
		maxPos, err := s.roleRepo.GetMaxPosition(ctx, workspaceID)
		if err != nil {
			log.Error("failed to get max role position", zap.Error(err), zap.String("workspace_id", workspaceID.String()))
			return nil, fmt.Errorf("get max role position: %w", err)
		}
		finalPosition = maxPos + 1
	}

	role := roleDomain.NewRole(workspaceID, name, permissionBitmask, finalPosition, false)

	if err := s.roleRepo.Create(ctx, role); err != nil {
		log.Error("failed to create role", zap.Error(err), zap.String("workspace_id", workspaceID.String()))
		return nil, fmt.Errorf("create role: %w", err)
	}

	log.Info("role created successfully",
		zap.String("role_id", role.ID.String()),
		zap.String("workspace_id", workspaceID.String()),
		zap.String("name", name),
		zap.Int32("position", finalPosition),
	)

	return role, nil
}

// AssignRoles replaces all role assignments for a member with the given roleIDs.
// The @everyone role is always included — it cannot be removed from a member.
// All provided roleIDs must belong to the same workspace as the member.
func (s *RoleService) AssignRoles(ctx context.Context, workspaceID uuid.UUID, memberID uuid.UUID, roleIDs []uuid.UUID) error {
	log := logger.FromContext(ctx, s.log)

	// Fetch @everyone role to ensure it's always included
	everyoneRole, err := s.roleRepo.GetEveryoneRole(ctx, workspaceID)
	if err != nil {
		log.Error("failed to get @everyone role", zap.Error(err), zap.String("workspace_id", workspaceID.String()))
		return fmt.Errorf("get @everyone role: %w", err)
	}

	// Build final role set: always include @everyone
	finalRoleIDs := make(map[uuid.UUID]struct{})
	finalRoleIDs[everyoneRole.ID] = struct{}{}
	for _, rid := range roleIDs {
		finalRoleIDs[rid] = struct{}{}
	}

	// Validate all roles belong to this workspace
	for rid := range finalRoleIDs {
		if rid == everyoneRole.ID {
			continue // already validated
		}
		role, err := s.roleRepo.GetByID(ctx, rid)
		if err != nil {
			log.Warn("role not found for assignment", zap.Error(err), zap.String("role_id", rid.String()))
			return &pkgerrors.DomainError{
				Code:    pkgerrors.CodeRecordNotFound,
				Message: fmt.Sprintf("Role %s tidak ditemukan.", rid.String()),
				Err:     err,
			}
		}
		if role.WorkspaceID != workspaceID {
			return &pkgerrors.DomainError{
				Code:    pkgerrors.CodeBusinessRuleViolation,
				Message: fmt.Sprintf("Role %s bukan milik workspace ini.", rid.String()),
				Err:     fmt.Errorf("role %s belongs to workspace %s, not %s", rid, role.WorkspaceID, workspaceID),
			}
		}
	}

	// Replace all assignments in a transaction
	err = s.txManager.WithinTransaction(ctx, func(txCtx context.Context) error {
		// Delete all existing assignments
		if err := s.roleRepo.DeleteAssignmentsByMember(txCtx, memberID); err != nil {
			log.Error("failed to delete existing role assignments", zap.Error(err), zap.String("member_id", memberID.String()))
			return fmt.Errorf("delete existing assignments: %w", err)
		}

		// Assign new roles
		for rid := range finalRoleIDs {
			if err := s.roleRepo.Assign(txCtx, memberID, rid); err != nil {
				log.Error("failed to assign role",
					zap.Error(err),
					zap.String("member_id", memberID.String()),
					zap.String("role_id", rid.String()),
				)
				return fmt.Errorf("assign role %s: %w", rid, err)
			}
		}

		return nil
	})

	if err != nil {
		return err
	}

	log.Info("roles assigned to member",
		zap.String("member_id", memberID.String()),
		zap.String("workspace_id", workspaceID.String()),
		zap.Int("role_count", len(finalRoleIDs)),
	)

	return nil
}
