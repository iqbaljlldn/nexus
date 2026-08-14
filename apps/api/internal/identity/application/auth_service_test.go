package application_test

import (
	"context"
	"testing"

	"github.com/iqbaljlldn/nexus/apps/api/internal/identity/application"
	"github.com/iqbaljlldn/nexus/apps/api/internal/identity/domain"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"go.uber.org/zap/zaptest"
)

type MockUserRepository struct {
	mock.Mock
}

func (m *MockUserRepository) Create(ctx context.Context, user *domain.User) error {
	args := m.Called(ctx, user)
	return args.Error(0)
}

func (m *MockUserRepository) FindByEmail(ctx context.Context, identifier string) (*domain.User, error) {
	args := m.Called(ctx, identifier)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*domain.User), args.Error(1)
}

func (m *MockUserRepository) FindByUsername(ctx context.Context, identifier string) (*domain.User, error) {
	args := m.Called(ctx, identifier)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*domain.User), args.Error(1)
}

func TestAuthService_Register(t *testing.T) {
	logger := zaptest.NewLogger(t)

	t.Run("success", func(t *testing.T) {
		mockRepo := new(MockUserRepository)
		service := application.NewAuthService(mockRepo, nil, nil, logger)

		mockRepo.On("FindByEmail", mock.Anything, mock.Anything).Return((*domain.User)(nil), domain.ErrUserNotFound).Maybe()
		mockRepo.On("FindByUsername", mock.Anything, mock.Anything).Return((*domain.User)(nil), domain.ErrUserNotFound).Maybe()
		mockRepo.On("Create", mock.Anything, mock.AnythingOfType("*domain.User")).Return(nil)

		user, err := service.Register(context.Background(), "test@example.com", "testuser", "Test User", "password123")

		assert.NoError(t, err)
		assert.NotNil(t, user)
		assert.Equal(t, "test@example.com", user.Email.String())
		assert.Equal(t, "testuser", user.Username.String())
		assert.Equal(t, "Test User", user.DisplayName)
		assert.NotEmpty(t, user.PasswordHash)
		mockRepo.AssertExpectations(t)
	})

	t.Run("invalid email", func(t *testing.T) {
		mockRepo := new(MockUserRepository)
		service := application.NewAuthService(mockRepo, nil, nil, logger)

		mockRepo.On("FindByEmail", mock.Anything, mock.Anything).Return((*domain.User)(nil), domain.ErrUserNotFound).Maybe()
		mockRepo.On("FindByUsername", mock.Anything, mock.Anything).Return((*domain.User)(nil), domain.ErrUserNotFound).Maybe()

		user, err := service.Register(context.Background(), "invalid-email", "testuser", "Test User", "password123")

		assert.ErrorIs(t, err, domain.ErrInvalidEmail)
		assert.Nil(t, user)
		mockRepo.AssertNotCalled(t, "Create")
	})

	t.Run("invalid username", func(t *testing.T) {
		mockRepo := new(MockUserRepository)
		service := application.NewAuthService(mockRepo, nil, nil, logger)

		mockRepo.On("FindByEmail", mock.Anything, mock.Anything).Return((*domain.User)(nil), domain.ErrUserNotFound).Maybe()
		mockRepo.On("FindByUsername", mock.Anything, mock.Anything).Return((*domain.User)(nil), domain.ErrUserNotFound).Maybe()

		user, err := service.Register(context.Background(), "test@example.com", "in", "Test User", "password123") // username too short

		assert.ErrorIs(t, err, domain.ErrInvalidUsername)
		assert.Nil(t, user)
		mockRepo.AssertNotCalled(t, "Create")
	})

	t.Run("duplicate email", func(t *testing.T) {
		mockRepo := new(MockUserRepository)
		service := application.NewAuthService(mockRepo, nil, nil, logger)

		mockRepo.On("FindByEmail", mock.Anything, "test@example.com").Return(&domain.User{}, nil)
		mockRepo.On("FindByUsername", mock.Anything, mock.Anything).Return((*domain.User)(nil), domain.ErrUserNotFound).Maybe()

		user, err := service.Register(context.Background(), "test@example.com", "testuser", "Test User", "password123")

		assert.ErrorIs(t, err, domain.ErrDuplicateEmail)
		assert.Nil(t, user)
		mockRepo.AssertExpectations(t)
	})

	t.Run("duplicate username", func(t *testing.T) {
		mockRepo := new(MockUserRepository)
		service := application.NewAuthService(mockRepo, nil, nil, logger)

		mockRepo.On("FindByEmail", mock.Anything, mock.Anything).Return((*domain.User)(nil), domain.ErrUserNotFound).Maybe()
		mockRepo.On("FindByUsername", mock.Anything, "testuser").Return(&domain.User{}, nil)

		user, err := service.Register(context.Background(), "test@example.com", "testuser", "Test User", "password123")

		assert.ErrorIs(t, err, domain.ErrDuplicateUsername)
		assert.Nil(t, user)
		mockRepo.AssertExpectations(t)
	})
}
