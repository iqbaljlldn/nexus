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
	"github.com/iqbaljlldn/nexus/apps/api/internal/channel/application"
	"github.com/iqbaljlldn/nexus/apps/api/internal/channel/domain"
	channelhttp "github.com/iqbaljlldn/nexus/apps/api/internal/channel/interface/http"
	roleDomain "github.com/iqbaljlldn/nexus/apps/api/internal/role/domain"
	"github.com/iqbaljlldn/nexus/pkg/contextutil"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"go.uber.org/zap/zaptest"
)

// --- Mocks ---

type MockPermissionResolver struct {
	mock.Mock
}

func (m *MockPermissionResolver) Resolve(ctx context.Context, userID, workspaceID, channelID uuid.UUID, required roleDomain.PermissionFlag) (bool, error) {
	args := m.Called(ctx, userID, workspaceID, channelID, required)
	return args.Bool(0), args.Error(1)
}

type MockChannelRepository struct {
	mock.Mock
}

func (m *MockChannelRepository) Create(ctx context.Context, channel *domain.Channel) error {
	args := m.Called(ctx, channel)
	if args.Error(0) == nil {
		channel.ID = uuid.New()
	}
	return args.Error(0)
}
func (m *MockChannelRepository) GetByID(ctx context.Context, id uuid.UUID) (*domain.Channel, error) {
	args := m.Called(ctx, id)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*domain.Channel), args.Error(1)
}
func (m *MockChannelRepository) ListByWorkspaceID(ctx context.Context, workspaceID uuid.UUID) ([]domain.Channel, error) {
	args := m.Called(ctx, workspaceID)
	return args.Get(0).([]domain.Channel), args.Error(1)
}
func (m *MockChannelRepository) GetMaxPosition(ctx context.Context, workspaceID uuid.UUID) (int32, error) {
	args := m.Called(ctx, workspaceID)
	return args.Get(0).(int32), args.Error(1)
}
func (m *MockChannelRepository) CreatePermissionOverride(ctx context.Context, override *domain.ChannelPermissionOverride) error {
	args := m.Called(ctx, override)
	if args.Error(0) == nil {
		override.ID = uuid.New()
	}
	return args.Error(0)
}
func (m *MockChannelRepository) GetPermissionOverrides(ctx context.Context, channelID uuid.UUID) ([]domain.ChannelPermissionOverride, error) {
	args := m.Called(ctx, channelID)
	return args.Get(0).([]domain.ChannelPermissionOverride), args.Error(1)
}
func (m *MockChannelRepository) GetChannelPermissionOverrideByRole(ctx context.Context, channelID, roleID uuid.UUID) (*domain.ChannelPermissionOverride, error) {
	args := m.Called(ctx, channelID, roleID)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*domain.ChannelPermissionOverride), args.Error(1)
}
func (m *MockChannelRepository) GetChannelPermissionOverrideByMember(ctx context.Context, channelID, memberID uuid.UUID) (*domain.ChannelPermissionOverride, error) {
	args := m.Called(ctx, channelID, memberID)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*domain.ChannelPermissionOverride), args.Error(1)
}
func (m *MockChannelRepository) UpdatePermissionOverride(ctx context.Context, id uuid.UUID, allowBitmask, denyBitmask int64) error {
	args := m.Called(ctx, id, allowBitmask, denyBitmask)
	return args.Error(0)
}
func (m *MockChannelRepository) DeletePermissionOverride(ctx context.Context, id uuid.UUID) error {
	args := m.Called(ctx, id)
	return args.Error(0)
}
func (m *MockChannelRepository) GetCategoryWorkspaceID(ctx context.Context, categoryID uuid.UUID) (uuid.UUID, error) {
	args := m.Called(ctx, categoryID)
	return args.Get(0).(uuid.UUID), args.Error(1)
}

func setupRouter(handler *channelhttp.ChannelHandler, userID uuid.UUID) *gin.Engine {
	gin.SetMode(gin.TestMode)
	r := gin.New()
	r.Use(func(c *gin.Context) {
		ctx := contextutil.WithUserID(c.Request.Context(), userID)
		c.Request = c.Request.WithContext(ctx)
		c.Next()
	})
	r.POST("/workspaces/:id/channels", handler.CreateTextChannel)
	r.PATCH("/channels/:id/permission-overrides", handler.PatchPermissionOverrides)
	return r
}

func TestChannelHandler_CreateTextChannel_Forbidden(t *testing.T) {
	log := zaptest.NewLogger(t)
	mockRepo := new(MockChannelRepository)
	mockPermResolver := new(MockPermissionResolver)
	svc := application.NewChannelService(mockRepo, log)
	handler := channelhttp.NewChannelHandler(svc, mockPermResolver)

	userID := uuid.New()
	workspaceID := uuid.New()
	router := setupRouter(handler, userID)

	mockPermResolver.On("Resolve", mock.Anything, userID, workspaceID, uuid.Nil, mock.Anything).Return(false, nil)

	body := map[string]string{"name": "general", "type": "text"}
	jsonBody, _ := json.Marshal(body)

	req, _ := http.NewRequest(http.MethodPost, "/workspaces/"+workspaceID.String()+"/channels", bytes.NewBuffer(jsonBody))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusForbidden, w.Code)
}

func TestChannelHandler_CreateTextChannel_Success(t *testing.T) {
	log := zaptest.NewLogger(t)
	mockRepo := new(MockChannelRepository)
	mockPermResolver := new(MockPermissionResolver)
	svc := application.NewChannelService(mockRepo, log)
	handler := channelhttp.NewChannelHandler(svc, mockPermResolver)

	userID := uuid.New()
	workspaceID := uuid.New()
	router := setupRouter(handler, userID)

	mockPermResolver.On("Resolve", mock.Anything, userID, workspaceID, uuid.Nil, mock.Anything).Return(true, nil)
	mockRepo.On("GetMaxPosition", mock.Anything, workspaceID).Return(int32(1), nil)
	mockRepo.On("Create", mock.Anything, mock.AnythingOfType("*domain.Channel")).Return(nil)

	body := map[string]string{"name": "general", "type": "text"}
	jsonBody, _ := json.Marshal(body)

	req, _ := http.NewRequest(http.MethodPost, "/workspaces/"+workspaceID.String()+"/channels", bytes.NewBuffer(jsonBody))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusCreated, w.Code)
}

func TestChannelHandler_PatchPermissionOverrides_Forbidden(t *testing.T) {
	log := zaptest.NewLogger(t)
	mockRepo := new(MockChannelRepository)
	mockPermResolver := new(MockPermissionResolver)
	svc := application.NewChannelService(mockRepo, log)
	handler := channelhttp.NewChannelHandler(svc, mockPermResolver)

	userID := uuid.New()
	workspaceID := uuid.New()
	channelID := uuid.New()
	roleID := uuid.New()
	router := setupRouter(handler, userID)

	ch := &domain.Channel{ID: channelID, WorkspaceID: &workspaceID}
	mockRepo.On("GetByID", mock.Anything, channelID).Return(ch, nil)
	mockPermResolver.On("Resolve", mock.Anything, userID, workspaceID, channelID, mock.Anything).Return(false, nil)

	body := map[string]interface{}{"role_id": roleID, "allow_bitmask": 1}
	jsonBody, _ := json.Marshal(body)

	req, _ := http.NewRequest(http.MethodPatch, "/channels/"+channelID.String()+"/permission-overrides", bytes.NewBuffer(jsonBody))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusForbidden, w.Code)
}

func TestChannelHandler_PatchPermissionOverrides_Success(t *testing.T) {
	log := zaptest.NewLogger(t)
	mockRepo := new(MockChannelRepository)
	mockPermResolver := new(MockPermissionResolver)
	svc := application.NewChannelService(mockRepo, log)
	handler := channelhttp.NewChannelHandler(svc, mockPermResolver)

	userID := uuid.New()
	workspaceID := uuid.New()
	channelID := uuid.New()
	roleID := uuid.New()
	router := setupRouter(handler, userID)

	ch := &domain.Channel{ID: channelID, WorkspaceID: &workspaceID}
	mockRepo.On("GetByID", mock.Anything, channelID).Return(ch, nil)
	mockPermResolver.On("Resolve", mock.Anything, userID, workspaceID, channelID, mock.Anything).Return(true, nil)
	mockRepo.On("GetChannelPermissionOverrideByRole", mock.Anything, channelID, roleID).Return(nil, nil)
	mockRepo.On("CreatePermissionOverride", mock.Anything, mock.AnythingOfType("*domain.ChannelPermissionOverride")).Return(nil)

	body := map[string]interface{}{"role_id": roleID, "allow_bitmask": 1}
	jsonBody, _ := json.Marshal(body)

	req, _ := http.NewRequest(http.MethodPatch, "/channels/"+channelID.String()+"/permission-overrides", bytes.NewBuffer(jsonBody))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)
}
