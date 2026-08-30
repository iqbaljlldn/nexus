package application

import (
	"context"
	"fmt"

	"github.com/google/uuid"
	memberDomain "github.com/iqbaljlldn/nexus/apps/api/internal/member/domain"
	roleDomain "github.com/iqbaljlldn/nexus/apps/api/internal/role/domain"
	wpDomain "github.com/iqbaljlldn/nexus/apps/api/internal/workspace/domain"
	"github.com/iqbaljlldn/nexus/pkg/logger"
	"go.uber.org/zap"
)

type WorkspaceService struct {
	workspaceRepo wpDomain.WorkspaceRepository
	memberPort    MemberPort
	rolePort      RolePort
	txManager     TransactionManager
	log           *zap.Logger
}

func NewWorkspaceService(
	workspaceRepo wpDomain.WorkspaceRepository,
	memberPort MemberPort,
	rolePort RolePort,
	txManager TransactionManager,
	log *zap.Logger,
) *WorkspaceService {
	return &WorkspaceService{
		workspaceRepo: workspaceRepo,
		memberPort:    memberPort,
		rolePort:      rolePort,
		txManager:     txManager,
		log:           log,
	}
}

// Create creates a new workspace in a single transaction. It atomically:
//  1. Creates the workspace itself
//  2. Creates the owner as the first member
//  3. Creates the default @everyone role with SEND_MESSAGES permission
//  4. Assigns the @everyone role to the owner member
//
// If any step fails, the entire transaction is rolled back.
func (s *WorkspaceService) Create(ctx context.Context, ownerID uuid.UUID, name string, iconURL string) (*wpDomain.Workspace, error) {
	log := logger.FromContext(ctx, s.log)

	// Validate via domain constructor
	ws, err := wpDomain.NewWorkspace(name)
	if err != nil {
		log.Warn("workspace name validation failed", zap.Error(err), zap.String("name", name))
		return nil, err
	}
	ws.OwnerID = ownerID
	ws.IconURL = iconURL

	err = s.txManager.WithinTransaction(ctx, func(txCtx context.Context) error {
		// Step 1: Create workspace
		if err := s.workspaceRepo.Create(txCtx, ws); err != nil {
			log.Error("failed to create workspace", zap.Error(err))
			return fmt.Errorf("create workspace: %w", err)
		}

		// Step 2: Create owner member
		member := &memberDomain.Member{
			WorkspaceID: ws.ID,
			UserID:      ownerID,
		}
		if err := s.memberPort.Create(txCtx, member); err != nil {
			log.Error("failed to create owner member", zap.Error(err), zap.String("workspace_id", ws.ID.String()))
			return fmt.Errorf("create owner member: %w", err)
		}

		// Step 3: Create default @everyone role
		everyoneRole := roleDomain.NewEveryoneRole(ws.ID)
		if err := s.rolePort.Create(txCtx, everyoneRole); err != nil {
			log.Error("failed to create @everyone role", zap.Error(err), zap.String("workspace_id", ws.ID.String()))
			return fmt.Errorf("create @everyone role: %w", err)
		}

		// Step 4: Assign @everyone role to the owner
		if err := s.rolePort.Assign(txCtx, member.ID, everyoneRole.ID); err != nil {
			log.Error("failed to assign @everyone role to owner",
				zap.Error(err),
				zap.String("member_id", member.ID.String()),
				zap.String("role_id", everyoneRole.ID.String()),
			)
			return fmt.Errorf("assign @everyone role: %w", err)
		}

		log.Info("workspace created successfully",
			zap.String("workspace_id", ws.ID.String()),
			zap.String("owner_id", ownerID.String()),
			zap.String("member_id", member.ID.String()),
			zap.String("everyone_role_id", everyoneRole.ID.String()),
		)

		return nil
	})

	if err != nil {
		return nil, err
	}

	return ws, nil
}
