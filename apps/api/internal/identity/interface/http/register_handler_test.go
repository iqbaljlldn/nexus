package http_test

import (
	"bytes"
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gin-gonic/gin"
	"github.com/iqbaljlldn/nexus/apps/api/internal/identity/application"
	"github.com/iqbaljlldn/nexus/apps/api/internal/identity/domain"
	identityhttp "github.com/iqbaljlldn/nexus/apps/api/internal/identity/interface/http"
	"github.com/iqbaljlldn/nexus/pkg/httpresponse"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"go.uber.org/zap"
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

func setupRouter(repo domain.UserRepository, logger *zap.Logger) *gin.Engine {
	gin.SetMode(gin.TestMode)

	r := gin.New()
	// Middleware for error handling might be needed depending on pkg/validator setup,
	// assuming c.Error() triggers default or custom error middleware.
	// For this test, we might just test the response.
	// Let's add a simple error middleware if needed, but httptest should capture it.
	r.Use(func(c *gin.Context) {
		c.Next()
		if len(c.Errors) > 0 {
			err := c.Errors.Last().Err
			httpresponse.Error(c, err)
		}
	})

	authService := application.NewAuthService(repo, nil, nil, logger)
	handler := identityhttp.NewAuthHandler(authService)
	handler.RegisterRoutes(r.Group("/"))

	return r
}

func TestRegisterHandler_Register(t *testing.T) {
	logger := zaptest.NewLogger(t)

	t.Run("success", func(t *testing.T) {
		mockRepo := new(MockUserRepository)
		router := setupRouter(mockRepo, logger)

		mockRepo.On("FindByEmail", mock.Anything, "test@example.com").Return((*domain.User)(nil), domain.ErrUserNotFound)
		mockRepo.On("FindByUsername", mock.Anything, "testuser").Return((*domain.User)(nil), domain.ErrUserNotFound)
		mockRepo.On("Create", mock.Anything, mock.AnythingOfType("*domain.User")).Return(nil)

		reqBody := identityhttp.RegisterRequest{
			Email:       "test@example.com",
			Username:    "testuser",
			DisplayName: "Test User",
			Password:    "password123",
		}
		jsonValue, _ := json.Marshal(reqBody)
		req, _ := http.NewRequest(http.MethodPost, "/auth/register", bytes.NewBuffer(jsonValue))
		w := httptest.NewRecorder()

		router.ServeHTTP(w, req)

		assert.Equal(t, http.StatusCreated, w.Code)

		var response httpresponse.SuccessResponse
		err := json.Unmarshal(w.Body.Bytes(), &response)
		assert.NoError(t, err)
		assert.True(t, response.Success)

		data := response.Data.(map[string]interface{})
		assert.Equal(t, "test@example.com", data["email"])
		assert.Equal(t, "testuser", data["username"])
		mockRepo.AssertExpectations(t)
	})

	t.Run("bad request validation", func(t *testing.T) {
		mockRepo := new(MockUserRepository)
		router := setupRouter(mockRepo, logger)

		reqBody := identityhttp.RegisterRequest{
			Email:       "invalid-email",
			Username:    "in",
			DisplayName: "",
			Password:    "short",
		}
		jsonValue, _ := json.Marshal(reqBody)
		req, _ := http.NewRequest(http.MethodPost, "/auth/register", bytes.NewBuffer(jsonValue))
		w := httptest.NewRecorder()

		router.ServeHTTP(w, req)

		assert.Equal(t, http.StatusBadRequest, w.Code)
		mockRepo.AssertNotCalled(t, "Create")
	})

	t.Run("conflict duplicate email", func(t *testing.T) {
		mockRepo := new(MockUserRepository)
		router := setupRouter(mockRepo, logger)

		mockRepo.On("FindByEmail", mock.Anything, "test@example.com").Return(&domain.User{}, nil)

		reqBody := identityhttp.RegisterRequest{
			Email:       "test@example.com",
			Username:    "testuser",
			DisplayName: "Test User",
			Password:    "password123",
		}
		jsonValue, _ := json.Marshal(reqBody)
		req, _ := http.NewRequest(http.MethodPost, "/auth/register", bytes.NewBuffer(jsonValue))
		w := httptest.NewRecorder()

		router.ServeHTTP(w, req)

		assert.Equal(t, http.StatusConflict, w.Code)
		var response httpresponse.ErrorResponse
		err := json.Unmarshal(w.Body.Bytes(), &response)
		assert.NoError(t, err)
		assert.False(t, response.Success)
		assert.Equal(t, "USER_ALREADY_EXISTS", response.Error.Code)
		mockRepo.AssertExpectations(t)
	})
}
