package application_test

import (
	"context"
	"errors"
	"testing"

	"github.com/google/uuid"
	"github.com/iqbaljlldn/nexus/apps/api/internal/channel/application"
	"github.com/iqbaljlldn/nexus/apps/api/internal/channel/domain"
	pkgerrors "github.com/iqbaljlldn/nexus/pkg/errors"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"go.uber.org/zap/zaptest"
)

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

func TestChannelService_CreateTextChannel_Success_NoCategory(t *testing.T) {
	log := zaptest.NewLogger(t)
	mockRepo := new(MockChannelRepository)
	svc := application.NewChannelService(mockRepo, log)

	workspaceID := uuid.New()
	mockRepo.On("GetMaxPosition", mock.Anything, workspaceID).Return(int32(1), nil)
	mockRepo.On("Create", mock.Anything, mock.AnythingOfType("*domain.Channel")).Return(nil)

	ch, err := svc.CreateTextChannel(context.Background(), workspaceID, "general", nil)

	assert.NoError(t, err)
	assert.NotNil(t, ch)
	assert.Equal(t, "general", *ch.Name)
	assert.Equal(t, int32(2), ch.Position)
	assert.Nil(t, ch.CategoryID)
	mockRepo.AssertExpectations(t)
}

func TestChannelService_CreateTextChannel_Success_WithCategory(t *testing.T) {
	log := zaptest.NewLogger(t)
	mockRepo := new(MockChannelRepository)
	svc := application.NewChannelService(mockRepo, log)

	workspaceID := uuid.New()
	categoryID := uuid.New()

	mockRepo.On("GetCategoryWorkspaceID", mock.Anything, categoryID).Return(workspaceID, nil)
	mockRepo.On("GetMaxPosition", mock.Anything, workspaceID).Return(int32(0), nil)
	mockRepo.On("Create", mock.Anything, mock.AnythingOfType("*domain.Channel")).Return(nil)

	ch, err := svc.CreateTextChannel(context.Background(), workspaceID, "announcements", &categoryID)

	assert.NoError(t, err)
	assert.NotNil(t, ch)
	assert.Equal(t, &categoryID, ch.CategoryID)
	mockRepo.AssertExpectations(t)
}

func TestChannelService_CreateTextChannel_CategoryWrongWorkspace(t *testing.T) {
	log := zaptest.NewLogger(t)
	mockRepo := new(MockChannelRepository)
	svc := application.NewChannelService(mockRepo, log)

	workspaceID := uuid.New()
	otherWorkspaceID := uuid.New()
	categoryID := uuid.New()

	mockRepo.On("GetCategoryWorkspaceID", mock.Anything, categoryID).Return(otherWorkspaceID, nil)

	ch, err := svc.CreateTextChannel(context.Background(), workspaceID, "announcements", &categoryID)

	assert.Error(t, err)
	assert.Nil(t, ch)
	var domainErr *pkgerrors.DomainError
	assert.True(t, errors.As(err, &domainErr))
	assert.Equal(t, pkgerrors.CodeBusinessRuleViolation, domainErr.Code)
}

func TestChannelService_PatchPermissionOverride_XORFailure(t *testing.T) {
	log := zaptest.NewLogger(t)
	mockRepo := new(MockChannelRepository)
	svc := application.NewChannelService(mockRepo, log)

	err := svc.PatchPermissionOverride(context.Background(), uuid.New(), nil, nil, 0, 0)
	assert.Error(t, err)

	roleID := uuid.New()
	memberID := uuid.New()
	err = svc.PatchPermissionOverride(context.Background(), uuid.New(), &roleID, &memberID, 0, 0)
	assert.Error(t, err)
}

func TestChannelService_PatchPermissionOverride_CreateNew(t *testing.T) {
	log := zaptest.NewLogger(t)
	mockRepo := new(MockChannelRepository)
	svc := application.NewChannelService(mockRepo, log)

	channelID := uuid.New()
	roleID := uuid.New()
	ch := &domain.Channel{ID: channelID, WorkspaceID: &uuid.UUID{}}

	mockRepo.On("GetByID", mock.Anything, channelID).Return(ch, nil)
	mockRepo.On("GetChannelPermissionOverrideByRole", mock.Anything, channelID, roleID).Return(nil, nil)
	mockRepo.On("CreatePermissionOverride", mock.Anything, mock.AnythingOfType("*domain.ChannelPermissionOverride")).Return(nil)

	err := svc.PatchPermissionOverride(context.Background(), channelID, &roleID, nil, 1, 0)
	assert.NoError(t, err)
	mockRepo.AssertExpectations(t)
}

func TestChannelService_PatchPermissionOverride_UpdateExisting(t *testing.T) {
	log := zaptest.NewLogger(t)
	mockRepo := new(MockChannelRepository)
	svc := application.NewChannelService(mockRepo, log)

	channelID := uuid.New()
	roleID := uuid.New()
	ch := &domain.Channel{ID: channelID, WorkspaceID: &uuid.UUID{}}
	existingOverride := &domain.ChannelPermissionOverride{
		ID:        uuid.New(),
		ChannelID: channelID,
		RoleID:    &roleID,
	}

	mockRepo.On("GetByID", mock.Anything, channelID).Return(ch, nil)
	mockRepo.On("GetChannelPermissionOverrideByRole", mock.Anything, channelID, roleID).Return(existingOverride, nil)
	mockRepo.On("UpdatePermissionOverride", mock.Anything, existingOverride.ID, int64(1), int64(2)).Return(nil)

	err := svc.PatchPermissionOverride(context.Background(), channelID, &roleID, nil, 1, 2)
	assert.NoError(t, err)
	mockRepo.AssertExpectations(t)
}
