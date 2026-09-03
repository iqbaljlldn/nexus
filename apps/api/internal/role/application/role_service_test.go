package application_test

import (
	"context"
	"errors"
	"testing"

	"github.com/google/uuid"
	roleDomain "github.com/iqbaljlldn/nexus/apps/api/internal/role/domain"

	"github.com/iqbaljlldn/nexus/apps/api/internal/role/application"
	pkgerrors "github.com/iqbaljlldn/nexus/pkg/errors"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
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

// FakeTxManager executes fn synchronously without a real DB transaction.
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

// --- Create Tests ---

func TestRoleService_Create_Success_AutoPosition(t *testing.T) {
	log := zaptest.NewLogger(t)
	mockRepo := new(MockRoleRepository)
	txManager := &FakeTxManager{}

	svc := application.NewRoleService(mockRepo, txManager, nil, log)

	workspaceID := uuid.New()

	mockRepo.On("GetMaxPosition", mock.Anything, workspaceID).Return(int32(3), nil)
	mockRepo.On("Create", mock.Anything, mock.AnythingOfType("*domain.Role")).Return(nil)

	role, err := svc.Create(context.Background(), workspaceID, "Moderator", int64(roleDomain.PermManageMessages|roleDomain.PermKickMembers), nil)

	assert.NoError(t, err)
	assert.NotNil(t, role)
	assert.Equal(t, "Moderator", role.Name)
	assert.Equal(t, int32(4), role.Position) // max(3) + 1
	assert.False(t, role.IsEveryone)
	assert.Equal(t, int64(roleDomain.PermManageMessages|roleDomain.PermKickMembers), role.PermissionBitmask)

	mockRepo.AssertExpectations(t)
}

func TestRoleService_Create_Success_ExplicitPosition(t *testing.T) {
	log := zaptest.NewLogger(t)
	mockRepo := new(MockRoleRepository)
	txManager := &FakeTxManager{}

	svc := application.NewRoleService(mockRepo, txManager, nil, log)

	workspaceID := uuid.New()
	pos := int32(5)

	// Should NOT call GetMaxPosition when position is explicitly provided
	mockRepo.On("Create", mock.Anything, mock.AnythingOfType("*domain.Role")).Return(nil)

	role, err := svc.Create(context.Background(), workspaceID, "Admin", int64(roleDomain.PermManageRoles), &pos)

	assert.NoError(t, err)
	assert.NotNil(t, role)
	assert.Equal(t, int32(5), role.Position)

	mockRepo.AssertNotCalled(t, "GetMaxPosition")
	mockRepo.AssertExpectations(t)
}

func TestRoleService_Create_EmptyName(t *testing.T) {
	log := zaptest.NewLogger(t)
	svc := application.NewRoleService(nil, &FakeTxManager{}, nil, log)

	role, err := svc.Create(context.Background(), uuid.New(), "", 0, nil)

	assert.Error(t, err)
	assert.Nil(t, role)
	var domainErr *pkgerrors.DomainError
	assert.True(t, errors.As(err, &domainErr))
	assert.Equal(t, pkgerrors.CodeMissingRequiredField, domainErr.Code)
}

func TestRoleService_Create_NameTooLong(t *testing.T) {
	log := zaptest.NewLogger(t)
	svc := application.NewRoleService(nil, &FakeTxManager{}, nil, log)

	longName := ""
	for i := 0; i < 101; i++ {
		longName += "a"
	}

	role, err := svc.Create(context.Background(), uuid.New(), longName, 0, nil)

	assert.Error(t, err)
	assert.Nil(t, role)
	var domainErr *pkgerrors.DomainError
	assert.True(t, errors.As(err, &domainErr))
	assert.Equal(t, pkgerrors.CodeInvalidFieldFormat, domainErr.Code)
}

func TestRoleService_Create_RepoFails(t *testing.T) {
	log := zaptest.NewLogger(t)
	mockRepo := new(MockRoleRepository)
	txManager := &FakeTxManager{}

	svc := application.NewRoleService(mockRepo, txManager, nil, log)

	mockRepo.On("GetMaxPosition", mock.Anything, mock.Anything).Return(int32(0), nil)
	mockRepo.On("Create", mock.Anything, mock.Anything).Return(errors.New("db error"))

	role, err := svc.Create(context.Background(), uuid.New(), "Test", 0, nil)

	assert.Error(t, err)
	assert.Nil(t, role)
	assert.Contains(t, err.Error(), "create role")
}

// --- AssignRoles Tests ---

func TestRoleService_AssignRoles_Success(t *testing.T) {
	log := zaptest.NewLogger(t)
	mockRepo := new(MockRoleRepository)
	txManager := &FakeTxManager{}

	svc := application.NewRoleService(mockRepo, txManager, nil, log)

	workspaceID := uuid.New()
	memberID := uuid.New()
	userID := uuid.New()
	everyoneRoleID := uuid.New()
	customRoleID := uuid.New()

	everyoneRole := &roleDomain.Role{
		ID:          everyoneRoleID,
		WorkspaceID: workspaceID,
		IsEveryone:  true,
	}
	customRole := &roleDomain.Role{
		ID:          customRoleID,
		WorkspaceID: workspaceID,
		Name:        "Moderator",
	}

	mockRepo.On("GetEveryoneRole", mock.Anything, workspaceID).Return(everyoneRole, nil)
	mockRepo.On("GetByID", mock.Anything, customRoleID).Return(customRole, nil)
	mockRepo.On("DeleteAssignmentsByMember", mock.Anything, memberID).Return(nil)
	mockRepo.On("Assign", mock.Anything, memberID, mock.AnythingOfType("uuid.UUID")).Return(nil)

	err := svc.AssignRoles(context.Background(), workspaceID, memberID, userID, []uuid.UUID{customRoleID})

	assert.NoError(t, err)

	// Verify both @everyone and custom role were assigned
	assignCalls := 0
	for _, call := range mockRepo.Calls {
		if call.Method == "Assign" {
			assignCalls++
		}
	}
	assert.Equal(t, 2, assignCalls) // @everyone + custom

	mockRepo.AssertExpectations(t)
}

func TestRoleService_AssignRoles_EveryoneAlwaysIncluded(t *testing.T) {
	log := zaptest.NewLogger(t)
	mockRepo := new(MockRoleRepository)
	txManager := &FakeTxManager{}

	svc := application.NewRoleService(mockRepo, txManager, nil, log)

	workspaceID := uuid.New()
	memberID := uuid.New()
	userID := uuid.New()
	everyoneRoleID := uuid.New()

	everyoneRole := &roleDomain.Role{
		ID:          everyoneRoleID,
		WorkspaceID: workspaceID,
		IsEveryone:  true,
	}

	mockRepo.On("GetEveryoneRole", mock.Anything, workspaceID).Return(everyoneRole, nil)
	mockRepo.On("DeleteAssignmentsByMember", mock.Anything, memberID).Return(nil)
	mockRepo.On("Assign", mock.Anything, memberID, everyoneRoleID).Return(nil)

	// Pass empty slice — @everyone should still be assigned
	err := svc.AssignRoles(context.Background(), workspaceID, memberID, userID, []uuid.UUID{})

	assert.NoError(t, err)

	// Verify @everyone was assigned even with empty input
	mockRepo.AssertCalled(t, "Assign", mock.Anything, memberID, everyoneRoleID)
	mockRepo.AssertExpectations(t)
}

func TestRoleService_AssignRoles_RoleNotFound(t *testing.T) {
	log := zaptest.NewLogger(t)
	mockRepo := new(MockRoleRepository)
	txManager := &FakeTxManager{}

	svc := application.NewRoleService(mockRepo, txManager, nil, log)

	workspaceID := uuid.New()
	memberID := uuid.New()
	userID := uuid.New()
	nonExistentRoleID := uuid.New()

	everyoneRole := &roleDomain.Role{
		ID:          uuid.New(),
		WorkspaceID: workspaceID,
		IsEveryone:  true,
	}

	mockRepo.On("GetEveryoneRole", mock.Anything, workspaceID).Return(everyoneRole, nil)
	mockRepo.On("GetByID", mock.Anything, nonExistentRoleID).Return(nil, errors.New("not found"))

	err := svc.AssignRoles(context.Background(), workspaceID, memberID, userID, []uuid.UUID{nonExistentRoleID})

	assert.Error(t, err)
	var domainErr *pkgerrors.DomainError
	assert.True(t, errors.As(err, &domainErr))
	assert.Equal(t, pkgerrors.CodeRecordNotFound, domainErr.Code)

	// Should NOT reach transaction
	mockRepo.AssertNotCalled(t, "DeleteAssignmentsByMember")
	mockRepo.AssertNotCalled(t, "Assign")
}

func TestRoleService_AssignRoles_RoleWrongWorkspace(t *testing.T) {
	log := zaptest.NewLogger(t)
	mockRepo := new(MockRoleRepository)
	txManager := &FakeTxManager{}

	svc := application.NewRoleService(mockRepo, txManager, nil, log)

	workspaceID := uuid.New()
	otherWorkspaceID := uuid.New()
	memberID := uuid.New()
	userID := uuid.New()
	foreignRoleID := uuid.New()

	everyoneRole := &roleDomain.Role{
		ID:          uuid.New(),
		WorkspaceID: workspaceID,
		IsEveryone:  true,
	}
	foreignRole := &roleDomain.Role{
		ID:          foreignRoleID,
		WorkspaceID: otherWorkspaceID, // different workspace!
		Name:        "Foreign Role",
	}

	mockRepo.On("GetEveryoneRole", mock.Anything, workspaceID).Return(everyoneRole, nil)
	mockRepo.On("GetByID", mock.Anything, foreignRoleID).Return(foreignRole, nil)

	err := svc.AssignRoles(context.Background(), workspaceID, memberID, userID, []uuid.UUID{foreignRoleID})

	assert.Error(t, err)
	var domainErr *pkgerrors.DomainError
	assert.True(t, errors.As(err, &domainErr))
	assert.Equal(t, pkgerrors.CodeBusinessRuleViolation, domainErr.Code)

	// Should NOT reach transaction
	mockRepo.AssertNotCalled(t, "DeleteAssignmentsByMember")
}
