package application

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/google/uuid"
	memberDomain "github.com/iqbaljlldn/nexus/apps/api/internal/member/domain"
	roleDomain "github.com/iqbaljlldn/nexus/apps/api/internal/role/domain"
	wpDomain "github.com/iqbaljlldn/nexus/apps/api/internal/workspace/domain"
	"github.com/iqbaljlldn/nexus/apps/api/internal/workspace/interface/dto"
	"github.com/iqbaljlldn/nexus/pkg/logger"
	"github.com/iqbaljlldn/nexus/pkg/pagination"
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

// List retrieves all workspaces owned by the given owner with pagination and search support.
//
// It performs the following steps:
//  1. Sanitizes the search query
//  2. Queries for workspaces owned by the user
//  3. Applies cursor-base pagination
//  4. Maps domain workspaces to HTTP response format
//
// Returns a ListWorkspacesResponse containing the paginated workspaces and total count, or an error if the query fails.
func (s *WorkspaceService) ListByUserID(ctx context.Context, userID uuid.UUID, req *dto.ListWorkspacesRequest) (*dto.ListWorkspacesResponse, *dto.PaginationMeta, error) {
	log := logger.FromContext(ctx, s.log)

	limit := req.Limit
	if limit == 0 {
		limit = 20
	}
	if limit > 100 {
		limit = 100
	}

	sortMode := req.SortMode
	if sortMode == "" {
		sortMode = "newest"
	}

	var parsedCursor *pagination.Cursor
	if req.Cursor != "" {
		c, err := pagination.DecodeCursor(req.Cursor)
		if err != nil {
			log.Warn("invalid cursor string", zap.Error(err), zap.String("cursor", req.Cursor))
			return nil, nil, fmt.Errorf("invalid cursor")
		}
		if c.SortMode != sortMode {
			log.Warn("cursor sort mode mismatch", zap.String("cursor_mode", c.SortMode), zap.String("req_mode", sortMode))
			return nil, nil, fmt.Errorf("cursor sort mode mismatch")
		}
		parsedCursor = &c
	}

	var workspaces []wpDomain.Workspace
	var err error

	// Fetch limit+1 to determine if there's a next page
	queryLimit := limit + 1

	switch sortMode {
	case "newest":
		workspaces, err = s.workspaceRepo.ListByNewest(ctx, userID, req.Search, parsedCursor, queryLimit)
	case "name_asc":
		workspaces, err = s.workspaceRepo.ListByNameAsc(ctx, userID, req.Search, parsedCursor, queryLimit)
	default:
		return nil, nil, fmt.Errorf("invalid sort_mode")
	}

	if err != nil {
		log.Error("failed to list workspaces", zap.Error(err))
		return nil, nil, fmt.Errorf("list workspaces: %w", err)
	}

	total, err := s.workspaceRepo.CountByUserID(ctx, userID, req.Search)
	if err != nil {
		log.Error("failed to count workspaces", zap.Error(err))
		// don't fail the whole request just because count failed
	}

	hasMore := false
	if len(workspaces) > int(limit) {
		hasMore = true
		workspaces = workspaces[:limit]
	}

	var nextCursorStr *string
	if hasMore && len(workspaces) > 0 {
		lastItem := workspaces[len(workspaces)-1]
		nextCur := pagination.Cursor{
			SortMode: sortMode,
			LastID:   lastItem.ID,
		}

		switch sortMode {
		case "newest":
			raw, _ := json.Marshal(lastItem.CreatedAt)
			nextCur.SortValue = raw
		case "name_asc":
			raw, _ := json.Marshal(lastItem.Name)
			nextCur.SortValue = raw
		}

		encoded, err := pagination.EncodeCursor(nextCur)
		if err == nil {
			nextCursorStr = &encoded
		} else {
			log.Error("failed to encode next cursor", zap.Error(err))
		}
	}

	return &dto.ListWorkspacesResponse{
			Workspaces: workspaces,
		}, &dto.PaginationMeta{
			Total:   total,
			Limit:   limit,
			Cursor:  nextCursorStr,
			HasMore: hasMore,
		}, nil
}

func (s *WorkspaceService) ListMembers(ctx context.Context, workspaceID uuid.UUID) ([]memberDomain.Member, error) {
	return s.memberPort.ListByWorkspaceID(ctx, workspaceID, 100)
}
