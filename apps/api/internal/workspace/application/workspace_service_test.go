package application_test

import (
	"context"
	"errors"
	"testing"

	"github.com/google/uuid"
	memberDomain "github.com/iqbaljlldn/nexus/apps/api/internal/member/domain"
	roleDomain "github.com/iqbaljlldn/nexus/apps/api/internal/role/domain"
	"github.com/iqbaljlldn/nexus/apps/api/internal/workspace/application"
	wpDomain "github.com/iqbaljlldn/nexus/apps/api/internal/workspace/domain"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"go.uber.org/zap/zaptest"
)

// --- Mocks ---

type MockWorkspaceRepository struct {
	mock.Mock
}

func (m *MockWorkspaceRepository) Create(ctx context.Context, workspace *wpDomain.Workspace) error {
	args := m.Called(ctx, workspace)
	if args.Error(0) == nil {
		// Simulate DB assigning UUID
		workspace.ID = uuid.New()
	}
	return args.Error(0)
}

func (m *MockWorkspaceRepository) GetByID(ctx context.Context, id uuid.UUID) (*wpDomain.Workspace, error) {
	args := m.Called(ctx, id)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*wpDomain.Workspace), args.Error(1)
}

type MockMemberPort struct {
	mock.Mock
}

func (m *MockMemberPort) Create(ctx context.Context, member *memberDomain.Member) error {
	args := m.Called(ctx, member)
	if args.Error(0) == nil {
		// Simulate DB assigning UUID
		member.ID = uuid.New()
	}
	return args.Error(0)
}

func (m *MockMemberPort) GetByWorkspaceAndUser(ctx context.Context, workspaceID, userID uuid.UUID) (*memberDomain.Member, error) {
	args := m.Called(ctx, workspaceID, userID)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*memberDomain.Member), args.Error(1)
}

type MockRolePort struct {
	mock.Mock
}

func (m *MockRolePort) Create(ctx context.Context, role *roleDomain.Role) error {
	args := m.Called(ctx, role)
	if args.Error(0) == nil {
		// Simulate DB assigning UUID
		role.ID = uuid.New()
	}
	return args.Error(0)
}

func (m *MockRolePort) Assign(ctx context.Context, memberID, roleID uuid.UUID) error {
	args := m.Called(ctx, memberID, roleID)
	return args.Error(0)
}

func (m *MockRolePort) GetEveryoneRole(ctx context.Context, workspaceID uuid.UUID) (*roleDomain.Role, error) {
	args := m.Called(ctx, workspaceID)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*roleDomain.Role), args.Error(1)
}

func (m *MockRolePort) FindMemberRolesSortedByPosition(ctx context.Context, memberID uuid.UUID) ([]*roleDomain.Role, error) {
	args := m.Called(ctx, memberID)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).([]*roleDomain.Role), args.Error(1)
}

// FakeTxManager executes fn synchronously without a real DB transaction,
// suitable for unit testing the service orchestration logic.
type FakeTxManager struct {
	shouldFail bool
}

func (f *FakeTxManager) WithinTransaction(ctx context.Context, fn func(ctx context.Context) error) error {
	err := fn(ctx)
	if f.shouldFail && err == nil {
		return errors.New("simulated commit failure")
	}
	return err
}

// --- Tests ---

func TestWorkspaceService_Create_Success(t *testing.T) {
	log := zaptest.NewLogger(t)
	mockWsRepo := new(MockWorkspaceRepository)
	mockMemberPort := new(MockMemberPort)
	mockRolePort := new(MockRolePort)
	txManager := &FakeTxManager{}

	svc := application.NewWorkspaceService(mockWsRepo, mockMemberPort, mockRolePort, txManager, log)

	ownerID := uuid.New()

	mockWsRepo.On("Create", mock.Anything, mock.AnythingOfType("*domain.Workspace")).Return(nil)
	mockMemberPort.On("Create", mock.Anything, mock.AnythingOfType("*domain.Member")).Return(nil)
	mockRolePort.On("Create", mock.Anything, mock.AnythingOfType("*domain.Role")).Return(nil)
	mockRolePort.On("Assign", mock.Anything, mock.AnythingOfType("uuid.UUID"), mock.AnythingOfType("uuid.UUID")).Return(nil)

	ws, err := svc.Create(context.Background(), ownerID, "Test Workspace", "")

	assert.NoError(t, err)
	assert.NotNil(t, ws)
	assert.Equal(t, "Test Workspace", ws.Name)
	assert.Equal(t, ownerID, ws.OwnerID)
	assert.NotEqual(t, uuid.Nil, ws.ID)

	mockWsRepo.AssertExpectations(t)
	mockMemberPort.AssertExpectations(t)
	mockRolePort.AssertExpectations(t)

	// Verify @everyone role was created with correct attributes
	roleCall := mockRolePort.Calls[0]
	createdRole := roleCall.Arguments[1].(*roleDomain.Role)
	assert.Equal(t, "@everyone", createdRole.Name)
	assert.True(t, createdRole.IsEveryone)
	assert.Equal(t, roleDomain.DefaultEveryonePermissions, createdRole.PermissionBitmask)
	assert.Equal(t, int32(0), createdRole.Position)

	// Verify member was created as owner with correct workspace ID
	memberCall := mockMemberPort.Calls[0]
	createdMember := memberCall.Arguments[1].(*memberDomain.Member)
	assert.Equal(t, ownerID, createdMember.UserID)
	assert.Equal(t, ws.ID, createdMember.WorkspaceID)
}

func TestWorkspaceService_Create_InvalidName_Empty(t *testing.T) {
	log := zaptest.NewLogger(t)
	svc := application.NewWorkspaceService(nil, nil, nil, &FakeTxManager{}, log)

	ws, err := svc.Create(context.Background(), uuid.New(), "", "")

	assert.ErrorIs(t, err, wpDomain.ErrEmptyName)
	assert.Nil(t, ws)
}

func TestWorkspaceService_Create_InvalidName_TooShort(t *testing.T) {
	log := zaptest.NewLogger(t)
	svc := application.NewWorkspaceService(nil, nil, nil, &FakeTxManager{}, log)

	ws, err := svc.Create(context.Background(), uuid.New(), "ab", "")

	assert.ErrorIs(t, err, wpDomain.ErrInvalidName)
	assert.Nil(t, ws)
}

func TestWorkspaceService_Create_InvalidName_TooLong(t *testing.T) {
	log := zaptest.NewLogger(t)
	svc := application.NewWorkspaceService(nil, nil, nil, &FakeTxManager{}, log)

	longName := ""
	for i := 0; i < 101; i++ {
		longName += "a"
	}

	ws, err := svc.Create(context.Background(), uuid.New(), longName, "")

	assert.ErrorIs(t, err, wpDomain.ErrInvalidName)
	assert.Nil(t, ws)
}

func TestWorkspaceService_Create_WorkspaceRepoFails(t *testing.T) {
	log := zaptest.NewLogger(t)
	mockWsRepo := new(MockWorkspaceRepository)
	mockMemberPort := new(MockMemberPort)
	mockRolePort := new(MockRolePort)
	txManager := &FakeTxManager{}

	svc := application.NewWorkspaceService(mockWsRepo, mockMemberPort, mockRolePort, txManager, log)

	dbErr := errors.New("db connection error")
	mockWsRepo.On("Create", mock.Anything, mock.Anything).Return(dbErr)

	ws, err := svc.Create(context.Background(), uuid.New(), "Test Workspace", "")

	assert.Error(t, err)
	assert.Nil(t, ws)
	assert.Contains(t, err.Error(), "create workspace")

	// Ensure subsequent steps were NOT called
	mockMemberPort.AssertNotCalled(t, "Create")
	mockRolePort.AssertNotCalled(t, "Create")
	mockRolePort.AssertNotCalled(t, "Assign")
}

func TestWorkspaceService_Create_MemberRepoFails(t *testing.T) {
	log := zaptest.NewLogger(t)
	mockWsRepo := new(MockWorkspaceRepository)
	mockMemberPort := new(MockMemberPort)
	mockRolePort := new(MockRolePort)
	txManager := &FakeTxManager{}

	svc := application.NewWorkspaceService(mockWsRepo, mockMemberPort, mockRolePort, txManager, log)

	mockWsRepo.On("Create", mock.Anything, mock.Anything).Return(nil)
	mockMemberPort.On("Create", mock.Anything, mock.Anything).Return(errors.New("member insert failed"))

	ws, err := svc.Create(context.Background(), uuid.New(), "Test Workspace", "")

	assert.Error(t, err)
	assert.Nil(t, ws)
	assert.Contains(t, err.Error(), "create owner member")

	// Role steps should NOT be called
	mockRolePort.AssertNotCalled(t, "Create")
	mockRolePort.AssertNotCalled(t, "Assign")
}

func TestWorkspaceService_Create_RoleCreateFails(t *testing.T) {
	log := zaptest.NewLogger(t)
	mockWsRepo := new(MockWorkspaceRepository)
	mockMemberPort := new(MockMemberPort)
	mockRolePort := new(MockRolePort)
	txManager := &FakeTxManager{}

	svc := application.NewWorkspaceService(mockWsRepo, mockMemberPort, mockRolePort, txManager, log)

	mockWsRepo.On("Create", mock.Anything, mock.Anything).Return(nil)
	mockMemberPort.On("Create", mock.Anything, mock.Anything).Return(nil)
	mockRolePort.On("Create", mock.Anything, mock.Anything).Return(errors.New("role insert failed"))

	ws, err := svc.Create(context.Background(), uuid.New(), "Test Workspace", "")

	assert.Error(t, err)
	assert.Nil(t, ws)
	assert.Contains(t, err.Error(), "create @everyone role")

	// Assign should NOT be called
	mockRolePort.AssertNotCalled(t, "Assign")
}

func TestWorkspaceService_Create_RoleAssignFails(t *testing.T) {
	log := zaptest.NewLogger(t)
	mockWsRepo := new(MockWorkspaceRepository)
	mockMemberPort := new(MockMemberPort)
	mockRolePort := new(MockRolePort)
	txManager := &FakeTxManager{}

	svc := application.NewWorkspaceService(mockWsRepo, mockMemberPort, mockRolePort, txManager, log)

	mockWsRepo.On("Create", mock.Anything, mock.Anything).Return(nil)
	mockMemberPort.On("Create", mock.Anything, mock.Anything).Return(nil)
	mockRolePort.On("Create", mock.Anything, mock.Anything).Return(nil)
	mockRolePort.On("Assign", mock.Anything, mock.Anything, mock.Anything).Return(errors.New("assign failed"))

	ws, err := svc.Create(context.Background(), uuid.New(), "Test Workspace", "")

	assert.Error(t, err)
	assert.Nil(t, ws)
	assert.Contains(t, err.Error(), "assign @everyone role")
}
