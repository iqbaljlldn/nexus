package application_test

import (
	"context"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/iqbaljlldn/nexus/apps/api/internal/workspace/application"
	wpDomain "github.com/iqbaljlldn/nexus/apps/api/internal/workspace/domain"
	"github.com/iqbaljlldn/nexus/apps/api/internal/workspace/interface/dto"
	"github.com/iqbaljlldn/nexus/pkg/pagination"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"go.uber.org/zap/zaptest"
)

// --- Add Mock methods for List ---

func (m *MockWorkspaceRepository) ListByNewest(ctx context.Context, userID uuid.UUID, search string, cursor *pagination.Cursor, limit uint) ([]wpDomain.Workspace, error) {
	args := m.Called(ctx, userID, search, cursor, limit)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).([]wpDomain.Workspace), args.Error(1)
}

func (m *MockWorkspaceRepository) ListByNameAsc(ctx context.Context, userID uuid.UUID, search string, cursor *pagination.Cursor, limit uint) ([]wpDomain.Workspace, error) {
	args := m.Called(ctx, userID, search, cursor, limit)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).([]wpDomain.Workspace), args.Error(1)
}

func (m *MockWorkspaceRepository) CountByUserID(ctx context.Context, userID uuid.UUID, search string) (uint64, error) {
	args := m.Called(ctx, userID, search)
	return args.Get(0).(uint64), args.Error(1)
}

func TestWorkspaceService_ListByUserID_Success(t *testing.T) {
	log := zaptest.NewLogger(t)
	mockWsRepo := new(MockWorkspaceRepository)
	svc := application.NewWorkspaceService(mockWsRepo, nil, nil, nil, log)

	userID := uuid.New()
	req := &dto.ListWorkspacesRequest{
		Limit:    10,
		SortMode: "newest",
	}

	workspaces := []wpDomain.Workspace{
		{ID: uuid.New(), Name: "WS 1", CreatedAt: time.Now()},
		{ID: uuid.New(), Name: "WS 2", CreatedAt: time.Now().Add(-1 * time.Hour)},
	}

	mockWsRepo.On("ListByNewest", mock.Anything, userID, "", (*pagination.Cursor)(nil), uint(11)).Return(workspaces, nil)
	mockWsRepo.On("CountByUserID", mock.Anything, userID, "").Return(uint64(2), nil)

	res, meta, err := svc.ListByUserID(context.Background(), userID, req)

	assert.NoError(t, err)
	assert.NotNil(t, res)
	assert.Len(t, res.Workspaces, 2)
	assert.NotNil(t, meta)
	assert.Equal(t, uint64(2), meta.Total)
	assert.False(t, meta.HasMore)
	assert.Nil(t, meta.Cursor)
}

func TestWorkspaceService_ListByUserID_HasMore(t *testing.T) {
	log := zaptest.NewLogger(t)
	mockWsRepo := new(MockWorkspaceRepository)
	svc := application.NewWorkspaceService(mockWsRepo, nil, nil, nil, log)

	userID := uuid.New()
	req := &dto.ListWorkspacesRequest{
		Limit:    2,
		SortMode: "name_asc",
	}

	workspaces := []wpDomain.Workspace{
		{ID: uuid.New(), Name: "A Workspace"},
		{ID: uuid.New(), Name: "B Workspace"},
		{ID: uuid.New(), Name: "C Workspace"}, // 3 items returned for limit 2
	}

	mockWsRepo.On("ListByNameAsc", mock.Anything, userID, "", (*pagination.Cursor)(nil), uint(3)).Return(workspaces, nil)
	mockWsRepo.On("CountByUserID", mock.Anything, userID, "").Return(uint64(5), nil)

	res, meta, err := svc.ListByUserID(context.Background(), userID, req)

	assert.NoError(t, err)
	assert.NotNil(t, res)
	assert.Len(t, res.Workspaces, 2) // truncated to limit
	assert.NotNil(t, meta)
	assert.True(t, meta.HasMore)
	assert.NotNil(t, meta.Cursor)

	// Verify cursor
	decoded, err := pagination.DecodeCursor(*meta.Cursor)
	assert.NoError(t, err)
	assert.Equal(t, "name_asc", decoded.SortMode)
	assert.Equal(t, workspaces[1].ID, decoded.LastID) // Cursor uses the last valid item
}

func TestWorkspaceService_ListByUserID_InvalidSortMode(t *testing.T) {
	log := zaptest.NewLogger(t)
	svc := application.NewWorkspaceService(nil, nil, nil, nil, log)

	req := &dto.ListWorkspacesRequest{
		SortMode: "invalid_mode",
	}

	res, meta, err := svc.ListByUserID(context.Background(), uuid.New(), req)

	assert.Error(t, err)
	assert.Nil(t, res)
	assert.Nil(t, meta)
	assert.Contains(t, err.Error(), "invalid sort_mode")
}

func TestWorkspaceService_ListByUserID_CursorMismatch(t *testing.T) {
	log := zaptest.NewLogger(t)
	svc := application.NewWorkspaceService(nil, nil, nil, nil, log)

	c := pagination.Cursor{
		SortMode: "name_asc",
		LastID:   uuid.New(),
	}
	encoded, _ := pagination.EncodeCursor(c)

	req := &dto.ListWorkspacesRequest{
		SortMode: "newest", // Mismatch with cursor!
		Cursor:   encoded,
	}

	res, meta, err := svc.ListByUserID(context.Background(), uuid.New(), req)

	assert.Error(t, err)
	assert.Nil(t, res)
	assert.Nil(t, meta)
	assert.Contains(t, err.Error(), "cursor sort mode mismatch")
}
