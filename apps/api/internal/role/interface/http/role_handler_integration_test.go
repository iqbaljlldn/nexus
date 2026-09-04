package http_test

import (
	"bytes"
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"github.com/iqbaljlldn/nexus/apps/api/internal/role/application"
	roleDomain "github.com/iqbaljlldn/nexus/apps/api/internal/role/domain"
	rolehttp "github.com/iqbaljlldn/nexus/apps/api/internal/role/interface/http"
	"github.com/iqbaljlldn/nexus/pkg/contextutil"
	"github.com/iqbaljlldn/nexus/pkg/httpresponse"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"go.uber.org/zap"
	"go.uber.org/zap/zaptest"
)

// --- Mocks ---

type MockRoleRepository struct {
	mock.Mock
}

func (m *MockRoleRepository) Create(ctx context.Context, role *roleDomain.Role) error {
	args := m.Called(ctx, role)
	if args.Error(0) == nil {
		role.ID = uuid.New()
	}
	return args.Error(0)
}

func (m *MockRoleRepository) GetByID(ctx context.Context, id uuid.UUID) (*roleDomain.Role, error) {
	args := m.Called(ctx, id)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*roleDomain.Role), args.Error(1)
}

func (m *MockRoleRepository) Assign(ctx context.Context, memberID, roleID uuid.UUID) error {
	args := m.Called(ctx, memberID, roleID)
	return args.Error(0)
}

func (m *MockRoleRepository) GetEveryoneRole(ctx context.Context, workspaceID uuid.UUID) (*roleDomain.Role, error) {
	args := m.Called(ctx, workspaceID)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*roleDomain.Role), args.Error(1)
}

func (m *MockRoleRepository) GetMaxPosition(ctx context.Context, workspaceID uuid.UUID) (int32, error) {
	args := m.Called(ctx, workspaceID)
	return args.Get(0).(int32), args.Error(1)
}

func (m *MockRoleRepository) ListAssignmentsByMember(ctx context.Context, memberID uuid.UUID) ([]roleDomain.RoleAssignment, error) {
	args := m.Called(ctx, memberID)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).([]roleDomain.RoleAssignment), args.Error(1)
}

func (m *MockRoleRepository) DeleteAssignmentsByMember(ctx context.Context, memberID uuid.UUID) error {
	args := m.Called(ctx, memberID)
	return args.Error(0)
}

func (m *MockRoleRepository) FindMemberRolesSortedByPosition(ctx context.Context, memberID uuid.UUID) ([]*roleDomain.Role, error) {
	args := m.Called(ctx, memberID)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).([]*roleDomain.Role), args.Error(1)
}

type FakeTxManager struct {
}

func (f *FakeTxManager) WithinTransaction(ctx context.Context, fn func(ctx context.Context) error) error {
	return fn(ctx)
}

type MockPermissionResolver struct {
	mock.Mock
}

func (m *MockPermissionResolver) Resolve(ctx context.Context, userID, workspaceID, channelID uuid.UUID, required roleDomain.PermissionFlag) (bool, error) {
	args := m.Called(ctx, userID, workspaceID, channelID, required)
	return args.Bool(0), args.Error(1)
}

type MockPermissionInvalidator struct {
	mock.Mock
}

func (m *MockPermissionInvalidator) InvalidateUserPermissions(ctx context.Context, workspaceID, userID uuid.UUID) error {
	args := m.Called(ctx, workspaceID, userID)
	return args.Error(0)
}

func setupRouter(roleRepo *MockRoleRepository, permResolver *MockPermissionResolver, invalidator *MockPermissionInvalidator, log *zap.Logger) *gin.Engine {
	gin.SetMode(gin.TestMode)

	r := gin.New()
	r.Use(func(c *gin.Context) {
		c.Next()
		if len(c.Errors) > 0 {
			err := c.Errors.Last().Err
			httpresponse.Error(c, err)
		}
	})

	roleService := application.NewRoleService(roleRepo, &FakeTxManager{}, invalidator, log)
	handler := rolehttp.NewRoleHandler(roleService, permResolver)

	// Register manually to bypass Auth middleware
	protected := r.Group("")
	protected.POST("/workspaces/:id/roles", handler.Create)
	protected.PATCH("/workspaces/:id/members/:memberId/roles", handler.AssignRoles)

	return r
}

func TestRoleHandler_Create(t *testing.T) {
	logger := zaptest.NewLogger(t)

	t.Run("success with MANAGE_ROLES permission", func(t *testing.T) {
		mockRepo := new(MockRoleRepository)
		mockPerm := new(MockPermissionResolver)
		mockInval := new(MockPermissionInvalidator)
		router := setupRouter(mockRepo, mockPerm, mockInval, logger)

		userID := uuid.New()
		workspaceID := uuid.New()

		// Allowed to manage roles
		mockPerm.On("Resolve", mock.Anything, userID, workspaceID, uuid.Nil, roleDomain.PermManageRoles).Return(true, nil)
		mockRepo.On("GetMaxPosition", mock.Anything, workspaceID).Return(int32(3), nil)
		mockRepo.On("Create", mock.Anything, mock.AnythingOfType("*domain.Role")).Return(nil)

		reqBody := map[string]interface{}{
			"name":               "Test Role",
			"permission_bitmask": 8,
		}
		jsonValue, _ := json.Marshal(reqBody)
		req, _ := http.NewRequest(http.MethodPost, "/workspaces/"+workspaceID.String()+"/roles", bytes.NewBuffer(jsonValue))

		// Setup context manually to bypass Auth middleware
		w := httptest.NewRecorder()
		c, _ := gin.CreateTestContext(w)
		c.Request = req.WithContext(contextutil.WithUserID(req.Context(), userID))
		router.ServeHTTP(w, c.Request)

		assert.Equal(t, http.StatusCreated, w.Code)
		mockPerm.AssertExpectations(t)
		mockRepo.AssertExpectations(t)
	})

	t.Run("forbidden without MANAGE_ROLES permission", func(t *testing.T) {
		mockRepo := new(MockRoleRepository)
		mockPerm := new(MockPermissionResolver)
		mockInval := new(MockPermissionInvalidator)
		router := setupRouter(mockRepo, mockPerm, mockInval, logger)

		userID := uuid.New()
		workspaceID := uuid.New()

		// Denied
		mockPerm.On("Resolve", mock.Anything, userID, workspaceID, uuid.Nil, roleDomain.PermManageRoles).Return(false, nil)

		reqBody := map[string]interface{}{
			"name":               "Test Role",
			"permission_bitmask": 8,
		}
		jsonValue, _ := json.Marshal(reqBody)
		req, _ := http.NewRequest(http.MethodPost, "/workspaces/"+workspaceID.String()+"/roles", bytes.NewBuffer(jsonValue))

		w := httptest.NewRecorder()
		c, _ := gin.CreateTestContext(w)
		c.Request = req.WithContext(contextutil.WithUserID(req.Context(), userID))
		router.ServeHTTP(w, c.Request)

		assert.Equal(t, http.StatusForbidden, w.Code)
		mockRepo.AssertNotCalled(t, "Create")
	})
}

func TestRoleHandler_AssignRoles(t *testing.T) {
	logger := zaptest.NewLogger(t)

	t.Run("success with MANAGE_ROLES permission", func(t *testing.T) {
		mockRepo := new(MockRoleRepository)
		mockPerm := new(MockPermissionResolver)
		mockInval := new(MockPermissionInvalidator)
		router := setupRouter(mockRepo, mockPerm, mockInval, logger)

		userID := uuid.New()
		workspaceID := uuid.New()
		memberID := uuid.New()
		roleID := uuid.New()

		mockPerm.On("Resolve", mock.Anything, userID, workspaceID, uuid.Nil, roleDomain.PermManageRoles).Return(true, nil)

		everyoneRole := &roleDomain.Role{
			ID:          uuid.New(),
			WorkspaceID: workspaceID,
			IsEveryone:  true,
		}
		customRole := &roleDomain.Role{
			ID:          roleID,
			WorkspaceID: workspaceID,
		}

		mockRepo.On("GetEveryoneRole", mock.Anything, workspaceID).Return(everyoneRole, nil)
		mockRepo.On("GetByID", mock.Anything, roleID).Return(customRole, nil)
		mockRepo.On("DeleteAssignmentsByMember", mock.Anything, memberID).Return(nil)
		mockRepo.On("Assign", mock.Anything, memberID, mock.AnythingOfType("uuid.UUID")).Return(nil)
		mockInval.On("InvalidateUserPermissions", mock.Anything, workspaceID, userID).Return(nil)

		reqBody := map[string]interface{}{
			"role_ids": []string{roleID.String()},
		}
		jsonValue, _ := json.Marshal(reqBody)
		req, _ := http.NewRequest(http.MethodPatch, "/workspaces/"+workspaceID.String()+"/members/"+memberID.String()+"/roles", bytes.NewBuffer(jsonValue))

		w := httptest.NewRecorder()
		c, _ := gin.CreateTestContext(w)
		c.Request = req.WithContext(contextutil.WithUserID(req.Context(), userID))
		router.ServeHTTP(w, c.Request)

		assert.Equal(t, http.StatusOK, w.Code)
		mockPerm.AssertExpectations(t)
		mockRepo.AssertExpectations(t)
	})
}
