package application_test

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/google/uuid"
	mbDomain "github.com/iqbaljlldn/nexus/apps/api/internal/member/domain"
	roleDomain "github.com/iqbaljlldn/nexus/apps/api/internal/role/domain"
	"github.com/iqbaljlldn/nexus/apps/api/internal/workspace/application"
	wpDomain "github.com/iqbaljlldn/nexus/apps/api/internal/workspace/domain"
	"github.com/iqbaljlldn/nexus/apps/api/internal/workspace/interface/dto"
	"github.com/iqbaljlldn/nexus/pkg/contextutil"
	pkgerrors "github.com/iqbaljlldn/nexus/pkg/errors"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"go.uber.org/zap/zaptest"
)

// --- Invite Mocks ---

type MockInviteRepository struct {
	mock.Mock
}

func (m *MockInviteRepository) Create(ctx context.Context, invite *wpDomain.Invite) error {
	args := m.Called(ctx, invite)
	if args.Error(0) == nil {
		invite.ID = uuid.New()
	}
	return args.Error(0)
}

func (m *MockInviteRepository) GetByCode(ctx context.Context, code string) (*wpDomain.Invite, error) {
	args := m.Called(ctx, code)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*wpDomain.Invite), args.Error(1)
}

func (m *MockInviteRepository) IncrementUseCount(ctx context.Context, inviteID uuid.UUID) error {
	args := m.Called(ctx, inviteID)
	return args.Error(0)
}

// --- Invite Service Unit Tests ---

func TestInviteService_Create_Success(t *testing.T) {
	log := zaptest.NewLogger(t)
	mockInviteRepo := new(MockInviteRepository)
	mockMemberPort := new(MockMemberPort)
	mockRolePort := new(MockRolePort)
	txManager := &FakeTxManager{}

	svc := application.NewInviteService(mockInviteRepo, mockMemberPort, mockRolePort, txManager, log)

	userID := uuid.New()
	workspaceID := uuid.New()

	ctx := contextutil.WithUserID(context.Background(), userID)

	mockInviteRepo.On("Create", mock.Anything, mock.AnythingOfType("*domain.Invite")).Return(nil)

	maxUses := 5
	expiresAt := time.Now().Add(24 * time.Hour)
	req := dto.CreateInviteReq{
		WorkspaceID: workspaceID,
		MaxUses:     &maxUses,
		ExpiresAt:   &expiresAt,
	}

	invite, err := svc.Create(ctx, req)

	assert.NoError(t, err)
	assert.NotNil(t, invite)
	assert.Equal(t, workspaceID, invite.WorkspaceID)
	assert.Equal(t, userID, invite.CreatedBy)
	assert.Equal(t, &maxUses, invite.MaxUses)
	assert.Equal(t, &expiresAt, invite.ExpiresAt)
	assert.NotEmpty(t, invite.Code)

	mockInviteRepo.AssertExpectations(t)
}

func TestInviteService_Create_Unauthorized(t *testing.T) {
	log := zaptest.NewLogger(t)
	mockInviteRepo := new(MockInviteRepository)
	svc := application.NewInviteService(mockInviteRepo, nil, nil, &FakeTxManager{}, log)

	req := dto.CreateInviteReq{
		WorkspaceID: uuid.New(),
	}

	invite, err := svc.Create(context.Background(), req)

	assert.Error(t, err)
	assert.Nil(t, invite)
	var domainErr *pkgerrors.DomainError
	assert.True(t, errors.As(err, &domainErr))
	assert.Equal(t, pkgerrors.CodeUserUnauthorized, domainErr.Code)
}

func TestInviteService_Redeem_Success(t *testing.T) {
	log := zaptest.NewLogger(t)
	mockInviteRepo := new(MockInviteRepository)
	mockMemberPort := new(MockMemberPort)
	mockRolePort := new(MockRolePort)
	txManager := &FakeTxManager{}

	svc := application.NewInviteService(mockInviteRepo, mockMemberPort, mockRolePort, txManager, log)

	userID := uuid.New()
	workspaceID := uuid.New()
	inviteID := uuid.New()
	everyoneRoleID := uuid.New()

	invite := &wpDomain.Invite{
		ID:          inviteID,
		WorkspaceID: workspaceID,
		Code:        "valid-code",
		CreatedBy:   uuid.New(),
		UseCount:    0,
	}

	mockInviteRepo.On("GetByCode", mock.Anything, "valid-code").Return(invite, nil)
	mockMemberPort.On("GetByWorkspaceAndUser", mock.Anything, workspaceID, userID).Return(nil, mbDomain.ErrMemberNotFound)
	mockInviteRepo.On("IncrementUseCount", mock.Anything, inviteID).Return(nil)
	mockMemberPort.On("Create", mock.Anything, mock.AnythingOfType("*domain.Member")).Return(nil)

	everyoneRole := &roleDomain.Role{
		ID:          everyoneRoleID,
		WorkspaceID: workspaceID,
		Name:        "@everyone",
		IsEveryone:  true,
	}
	mockRolePort.On("GetEveryoneRole", mock.Anything, workspaceID).Return(everyoneRole, nil)
	mockRolePort.On("Assign", mock.Anything, mock.AnythingOfType("uuid.UUID"), everyoneRoleID).Return(nil)

	member, err := svc.Redeem(context.Background(), "valid-code", userID)

	assert.NoError(t, err)
	assert.NotNil(t, member)
	assert.Equal(t, workspaceID, member.WorkspaceID)
	assert.Equal(t, userID, member.UserID)

	mockInviteRepo.AssertExpectations(t)
	mockMemberPort.AssertExpectations(t)
	mockRolePort.AssertExpectations(t)
}

func TestInviteService_Redeem_Duplicate_Idempotent(t *testing.T) {
	log := zaptest.NewLogger(t)
	mockInviteRepo := new(MockInviteRepository)
	mockMemberPort := new(MockMemberPort)
	mockRolePort := new(MockRolePort)
	txManager := &FakeTxManager{}

	svc := application.NewInviteService(mockInviteRepo, mockMemberPort, mockRolePort, txManager, log)

	userID := uuid.New()
	workspaceID := uuid.New()
	existingMemberID := uuid.New()

	invite := &wpDomain.Invite{
		ID:          uuid.New(),
		WorkspaceID: workspaceID,
		Code:        "already-joined-code",
	}

	existingMember := &mbDomain.Member{
		ID:          existingMemberID,
		WorkspaceID: workspaceID,
		UserID:      userID,
	}

	mockInviteRepo.On("GetByCode", mock.Anything, "already-joined-code").Return(invite, nil)
	mockMemberPort.On("GetByWorkspaceAndUser", mock.Anything, workspaceID, userID).Return(existingMember, nil)

	member, err := svc.Redeem(context.Background(), "already-joined-code", userID)

	assert.NoError(t, err)
	assert.NotNil(t, member)
	assert.Equal(t, existingMemberID, member.ID)

	// Ensure no new member or role assignment calls were made
	mockInviteRepo.AssertNotCalled(t, "IncrementUseCount")
	mockMemberPort.AssertNotCalled(t, "Create")
	mockRolePort.AssertNotCalled(t, "GetEveryoneRole")
	mockRolePort.AssertNotCalled(t, "Assign")
}

func TestInviteService_Redeem_Expired_422(t *testing.T) {
	log := zaptest.NewLogger(t)
	mockInviteRepo := new(MockInviteRepository)
	svc := application.NewInviteService(mockInviteRepo, nil, nil, &FakeTxManager{}, log)

	past := time.Now().Add(-1 * time.Hour)
	invite := &wpDomain.Invite{
		ID:          uuid.New(),
		WorkspaceID: uuid.New(),
		Code:        "expired-code",
		ExpiresAt:   &past,
	}

	mockInviteRepo.On("GetByCode", mock.Anything, "expired-code").Return(invite, nil)

	member, err := svc.Redeem(context.Background(), "expired-code", uuid.New())

	assert.Error(t, err)
	assert.Nil(t, member)

	var domainErr *pkgerrors.DomainError
	assert.True(t, errors.As(err, &domainErr))
	assert.Equal(t, pkgerrors.CodeBusinessRuleViolation, domainErr.Code)
}

func TestInviteService_Redeem_MaxUsesReached_422(t *testing.T) {
	log := zaptest.NewLogger(t)
	mockInviteRepo := new(MockInviteRepository)
	svc := application.NewInviteService(mockInviteRepo, nil, nil, &FakeTxManager{}, log)

	maxUses := 2
	invite := &wpDomain.Invite{
		ID:          uuid.New(),
		WorkspaceID: uuid.New(),
		Code:        "max-uses-code",
		MaxUses:     &maxUses,
		UseCount:    2,
	}

	mockInviteRepo.On("GetByCode", mock.Anything, "max-uses-code").Return(invite, nil)

	member, err := svc.Redeem(context.Background(), "max-uses-code", uuid.New())

	assert.Error(t, err)
	assert.Nil(t, member)

	var domainErr *pkgerrors.DomainError
	assert.True(t, errors.As(err, &domainErr))
	assert.Equal(t, pkgerrors.CodeBusinessRuleViolation, domainErr.Code)
}

func TestInviteService_Redeem_NotFound(t *testing.T) {
	log := zaptest.NewLogger(t)
	mockInviteRepo := new(MockInviteRepository)
	svc := application.NewInviteService(mockInviteRepo, nil, nil, &FakeTxManager{}, log)

	mockInviteRepo.On("GetByCode", mock.Anything, "invalid-code").Return(nil, wpDomain.ErrInviteNotFound)

	member, err := svc.Redeem(context.Background(), "invalid-code", uuid.New())

	assert.Error(t, err)
	assert.Nil(t, member)

	var domainErr *pkgerrors.DomainError
	assert.True(t, errors.As(err, &domainErr))
	assert.Equal(t, pkgerrors.CodeRecordNotFound, domainErr.Code)
}
