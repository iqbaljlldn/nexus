package application_test

import (
	"context"
	"testing"

	"github.com/google/uuid"
	memberDomain "github.com/iqbaljlldn/nexus/apps/api/internal/member/domain"
	roleDomain "github.com/iqbaljlldn/nexus/apps/api/internal/role/domain"
	"github.com/iqbaljlldn/nexus/apps/api/internal/workspace/application"
	wpDomain "github.com/iqbaljlldn/nexus/apps/api/internal/workspace/domain"
	"github.com/iqbaljlldn/nexus/pkg/pagination"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
)

// --- Mocks ---

type MockPermResolverWorkspaceRepo struct {
	mock.Mock
}

func (m *MockPermResolverWorkspaceRepo) GetByID(ctx context.Context, id uuid.UUID) (*wpDomain.Workspace, error) {
	args := m.Called(ctx, id)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*wpDomain.Workspace), args.Error(1)
}

func (m *MockPermResolverWorkspaceRepo) Create(ctx context.Context, workspace *wpDomain.Workspace) error {
	return nil
}

func (m *MockPermResolverWorkspaceRepo) ListByNewest(ctx context.Context, userID uuid.UUID, search string, cursor *pagination.Cursor, limit uint) ([]wpDomain.Workspace, error) {
	return nil, nil
}

func (m *MockPermResolverWorkspaceRepo) ListByNameAsc(ctx context.Context, userID uuid.UUID, search string, cursor *pagination.Cursor, limit uint) ([]wpDomain.Workspace, error) {
	return nil, nil
}

func (m *MockPermResolverWorkspaceRepo) CountByUserID(ctx context.Context, userID uuid.UUID, search string) (uint64, error) {
	return 0, nil
}

type MockPermResolverMemberPort struct {
	mock.Mock
}

func (m *MockPermResolverMemberPort) Create(ctx context.Context, member *memberDomain.Member) error {
	return nil
}

func (m *MockPermResolverMemberPort) GetByWorkspaceAndUser(ctx context.Context, workspaceID, userID uuid.UUID) (*memberDomain.Member, error) {
	args := m.Called(ctx, workspaceID, userID)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*memberDomain.Member), args.Error(1)
}

func (m *MockPermResolverMemberPort) ListByWorkspaceID(ctx context.Context, workspaceID uuid.UUID, limit int32) ([]memberDomain.Member, error) {
	args := m.Called(ctx, workspaceID, limit)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).([]memberDomain.Member), args.Error(1)
}

type MockChannelOverridePort struct {
	mock.Mock
}

func (m *MockChannelOverridePort) FindMemberOverride(ctx context.Context, channelID, userID uuid.UUID) (*application.ChannelOverride, bool, error) {
	args := m.Called(ctx, channelID, userID)
	if args.Get(0) == nil {
		return nil, args.Bool(1), args.Error(2)
	}
	return args.Get(0).(*application.ChannelOverride), args.Bool(1), args.Error(2)
}

func (m *MockChannelOverridePort) FindRoleOverride(ctx context.Context, channelID, roleID uuid.UUID) (*application.ChannelOverride, bool, error) {
	args := m.Called(ctx, channelID, roleID)
	if args.Get(0) == nil {
		return nil, args.Bool(1), args.Error(2)
	}
	return args.Get(0).(*application.ChannelOverride), args.Bool(1), args.Error(2)
}

type MockRolePort2 struct {
	mock.Mock
}

func (m *MockRolePort2) Create(ctx context.Context, role *roleDomain.Role) error {
	args := m.Called(ctx, role)
	return args.Error(0)
}

func (m *MockRolePort2) Assign(ctx context.Context, memberID, roleID uuid.UUID) error {
	args := m.Called(ctx, memberID, roleID)
	return args.Error(0)
}

func (m *MockRolePort2) FindMemberRolesSortedByPosition(ctx context.Context, memberID uuid.UUID) ([]*roleDomain.Role, error) {
	args := m.Called(ctx, memberID)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).([]*roleDomain.Role), args.Error(1)
}

func (m *MockRolePort2) GetEveryoneRole(ctx context.Context, workspaceID uuid.UUID) (*roleDomain.Role, error) {
	args := m.Called(ctx, workspaceID)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*roleDomain.Role), args.Error(1)
}

// --- Tests ---

func TestPermissionResolver_Resolve(t *testing.T) {
	t.Run("Owner bypasses everything", func(t *testing.T) {
		mockOverride := new(MockChannelOverridePort)
		mockRole := new(MockRolePort2)
		mockWorkspace := new(MockPermResolverWorkspaceRepo)
		mockMember := new(MockPermResolverMemberPort)
		resolver := application.NewPermissionResolver(mockWorkspace, mockMember, mockOverride, mockRole)

		userID, workspaceID, channelID := uuid.New(), uuid.New(), uuid.New()

		mockWorkspace.On("GetByID", mock.Anything, workspaceID).Return(&wpDomain.Workspace{OwnerID: userID}, nil)

		allowed, err := resolver.Resolve(context.Background(), userID, workspaceID, channelID, roleDomain.PermManageChannels)
		assert.NoError(t, err)
		assert.True(t, allowed)
	})

	t.Run("Fallback to @everyone - Allow", func(t *testing.T) {
		mockOverride := new(MockChannelOverridePort)
		mockRole := new(MockRolePort2)
		mockWorkspace := new(MockPermResolverWorkspaceRepo)
		mockMember := new(MockPermResolverMemberPort)
		resolver := application.NewPermissionResolver(mockWorkspace, mockMember, mockOverride, mockRole)

		userID, workspaceID, channelID := uuid.New(), uuid.New(), uuid.New()
		memberID := uuid.New()

		mockWorkspace.On("GetByID", mock.Anything, workspaceID).Return(&wpDomain.Workspace{OwnerID: uuid.New()}, nil)
		mockMember.On("GetByWorkspaceAndUser", mock.Anything, workspaceID, userID).Return(&memberDomain.Member{ID: memberID}, nil)

		mockOverride.On("FindMemberOverride", mock.Anything, channelID, memberID).Return(nil, false, nil)
		mockRole.On("FindMemberRolesSortedByPosition", mock.Anything, memberID).Return([]*roleDomain.Role{}, nil)
		mockRole.On("GetEveryoneRole", mock.Anything, workspaceID).Return(&roleDomain.Role{
			PermissionBitmask: int64(roleDomain.PermSendMessages),
		}, nil)

		allowed, err := resolver.Resolve(context.Background(), userID, workspaceID, channelID, roleDomain.PermSendMessages)
		assert.NoError(t, err)
		assert.True(t, allowed)
	})

	t.Run("Fallback to @everyone - Deny", func(t *testing.T) {
		mockOverride := new(MockChannelOverridePort)
		mockRole := new(MockRolePort2)
		mockWorkspace := new(MockPermResolverWorkspaceRepo)
		mockMember := new(MockPermResolverMemberPort)
		resolver := application.NewPermissionResolver(mockWorkspace, mockMember, mockOverride, mockRole)

		userID, workspaceID, channelID := uuid.New(), uuid.New(), uuid.New()
		memberID := uuid.New()

		mockWorkspace.On("GetByID", mock.Anything, workspaceID).Return(&wpDomain.Workspace{OwnerID: uuid.New()}, nil)
		mockMember.On("GetByWorkspaceAndUser", mock.Anything, workspaceID, userID).Return(&memberDomain.Member{ID: memberID}, nil)

		mockOverride.On("FindMemberOverride", mock.Anything, channelID, memberID).Return(nil, false, nil)
		mockRole.On("FindMemberRolesSortedByPosition", mock.Anything, memberID).Return([]*roleDomain.Role{}, nil)
		mockRole.On("GetEveryoneRole", mock.Anything, workspaceID).Return(&roleDomain.Role{
			PermissionBitmask: int64(roleDomain.PermSendMessages), // does not have MANAGE_CHANNELS
		}, nil)

		allowed, err := resolver.Resolve(context.Background(), userID, workspaceID, channelID, roleDomain.PermManageChannels)
		assert.NoError(t, err)
		assert.False(t, allowed)
	})

	t.Run("Role Override Deny wins over Role Default", func(t *testing.T) {
		mockOverride := new(MockChannelOverridePort)
		mockRole := new(MockRolePort2)
		mockWorkspace := new(MockPermResolverWorkspaceRepo)
		mockMember := new(MockPermResolverMemberPort)
		resolver := application.NewPermissionResolver(mockWorkspace, mockMember, mockOverride, mockRole)

		userID, workspaceID, channelID := uuid.New(), uuid.New(), uuid.New()
		memberID := uuid.New()
		roleID := uuid.New()

		mockWorkspace.On("GetByID", mock.Anything, workspaceID).Return(&wpDomain.Workspace{OwnerID: uuid.New()}, nil)
		mockMember.On("GetByWorkspaceAndUser", mock.Anything, workspaceID, userID).Return(&memberDomain.Member{ID: memberID}, nil)

		mockOverride.On("FindMemberOverride", mock.Anything, channelID, memberID).Return(nil, false, nil)
		mockRole.On("FindMemberRolesSortedByPosition", mock.Anything, memberID).Return([]*roleDomain.Role{
			{ID: roleID, PermissionBitmask: int64(roleDomain.PermSendMessages)}, // Default allows
		}, nil)
		mockOverride.On("FindRoleOverride", mock.Anything, channelID, roleID).Return(&application.ChannelOverride{
			Deny: roleDomain.PermSendMessages, // Override denies
		}, true, nil)

		allowed, err := resolver.Resolve(context.Background(), userID, workspaceID, channelID, roleDomain.PermSendMessages)
		assert.NoError(t, err)
		assert.False(t, allowed)
	})

	t.Run("Member Override Allow wins over Role Default Deny", func(t *testing.T) {
		mockOverride := new(MockChannelOverridePort)
		mockRole := new(MockRolePort2)
		mockWorkspace := new(MockPermResolverWorkspaceRepo)
		mockMember := new(MockPermResolverMemberPort)
		resolver := application.NewPermissionResolver(mockWorkspace, mockMember, mockOverride, mockRole)

		userID, workspaceID, channelID := uuid.New(), uuid.New(), uuid.New()
		memberID := uuid.New()

		mockWorkspace.On("GetByID", mock.Anything, workspaceID).Return(&wpDomain.Workspace{OwnerID: uuid.New()}, nil)
		mockMember.On("GetByWorkspaceAndUser", mock.Anything, workspaceID, userID).Return(&memberDomain.Member{ID: memberID}, nil)

		mockOverride.On("FindMemberOverride", mock.Anything, channelID, memberID).Return(&application.ChannelOverride{
			Allow: roleDomain.PermManageChannels, // Member explicitly allowed
		}, true, nil)

		allowed, err := resolver.Resolve(context.Background(), userID, workspaceID, channelID, roleDomain.PermManageChannels)
		assert.NoError(t, err)
		assert.True(t, allowed)
		mockRole.AssertNotCalled(t, "FindMemberRolesSortedByPosition")
	})

	t.Run("Multiple Roles - Highest Position Default Wins", func(t *testing.T) {
		mockOverride := new(MockChannelOverridePort)
		mockRole := new(MockRolePort2)
		mockWorkspace := new(MockPermResolverWorkspaceRepo)
		mockMember := new(MockPermResolverMemberPort)
		resolver := application.NewPermissionResolver(mockWorkspace, mockMember, mockOverride, mockRole)

		userID, workspaceID, channelID := uuid.New(), uuid.New(), uuid.New()
		memberID := uuid.New()
		role1ID, role2ID := uuid.New(), uuid.New()

		mockWorkspace.On("GetByID", mock.Anything, workspaceID).Return(&wpDomain.Workspace{OwnerID: uuid.New()}, nil)
		mockMember.On("GetByWorkspaceAndUser", mock.Anything, workspaceID, userID).Return(&memberDomain.Member{ID: memberID}, nil)

		mockOverride.On("FindMemberOverride", mock.Anything, channelID, memberID).Return(nil, false, nil)
		mockRole.On("FindMemberRolesSortedByPosition", mock.Anything, memberID).Return([]*roleDomain.Role{
			{ID: role1ID, PermissionBitmask: int64(roleDomain.PermManageMessages), Position: 10}, // High position (Allows)
			{ID: role2ID, PermissionBitmask: 0, Position: 5},                                     // Low position (Does not allow)
		}, nil)
		mockOverride.On("FindRoleOverride", mock.Anything, channelID, role1ID).Return(nil, false, nil)
		mockOverride.On("FindRoleOverride", mock.Anything, channelID, role2ID).Return(nil, false, nil)

		allowed, err := resolver.Resolve(context.Background(), userID, workspaceID, channelID, roleDomain.PermManageMessages)
		assert.NoError(t, err)
		assert.True(t, allowed)
	})

	t.Run("Multiple Roles - Highest Position Override Wins", func(t *testing.T) {
		mockOverride := new(MockChannelOverridePort)
		mockRole := new(MockRolePort2)
		mockWorkspace := new(MockPermResolverWorkspaceRepo)
		mockMember := new(MockPermResolverMemberPort)
		resolver := application.NewPermissionResolver(mockWorkspace, mockMember, mockOverride, mockRole)

		userID, workspaceID, channelID := uuid.New(), uuid.New(), uuid.New()
		memberID := uuid.New()
		role1ID, role2ID := uuid.New(), uuid.New()

		mockWorkspace.On("GetByID", mock.Anything, workspaceID).Return(&wpDomain.Workspace{OwnerID: uuid.New()}, nil)
		mockMember.On("GetByWorkspaceAndUser", mock.Anything, workspaceID, userID).Return(&memberDomain.Member{ID: memberID}, nil)

		mockOverride.On("FindMemberOverride", mock.Anything, channelID, memberID).Return(nil, false, nil)
		mockRole.On("FindMemberRolesSortedByPosition", mock.Anything, memberID).Return([]*roleDomain.Role{
			{ID: role1ID, PermissionBitmask: int64(roleDomain.PermSendMessages), Position: 10},
			{ID: role2ID, PermissionBitmask: int64(roleDomain.PermSendMessages), Position: 5},
		}, nil)

		// High position role denies sending messages
		mockOverride.On("FindRoleOverride", mock.Anything, channelID, role1ID).Return(&application.ChannelOverride{
			Deny: roleDomain.PermSendMessages,
		}, true, nil)

		allowed, err := resolver.Resolve(context.Background(), userID, workspaceID, channelID, roleDomain.PermSendMessages)
		assert.NoError(t, err)
		assert.False(t, allowed)

		// Should exit early and not check role2 override
		mockOverride.AssertNotCalled(t, "FindRoleOverride", mock.Anything, channelID, role2ID)
	})
}
