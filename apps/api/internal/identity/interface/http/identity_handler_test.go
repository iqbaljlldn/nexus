package http_test

import (
	"bytes"
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

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

func setupRouter(userRepo domain.UserRepository, sessionRepo domain.SessionRepository, tokenManager domain.TokenManager, logger *zap.Logger) *gin.Engine {
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

	authService := application.NewAuthService(userRepo, sessionRepo, tokenManager, logger)
	handler := identityhttp.NewAuthHandler(authService)
	handler.RegisterRoutes(r.Group("/"))

	return r
}

func TestRegisterHandler_Register(t *testing.T) {
	logger := zaptest.NewLogger(t)

	t.Run("success", func(t *testing.T) {
		mockRepo := new(MockUserRepository)
		router := setupRouter(mockRepo, nil, nil, logger)

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
		router := setupRouter(mockRepo, nil, nil, logger)

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
		router := setupRouter(mockRepo, nil, nil, logger)

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

type MockSessionRepository struct {
	mock.Mock
}

func (m *MockSessionRepository) Create(ctx context.Context, session *domain.Session) error {
	args := m.Called(ctx, session)
	return args.Error(0)
}

func (m *MockSessionRepository) RotateRefreshToken(ctx context.Context, oldTokenHash, newTokenHash string) error {
	args := m.Called(ctx, oldTokenHash, newTokenHash)
	return args.Error(0)
}

func (m *MockSessionRepository) FindByRefreshToken(ctx context.Context, refreshToken string) (*domain.Session, error) {
	args := m.Called(ctx, refreshToken)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*domain.Session), args.Error(1)
}

func (m *MockSessionRepository) RevokeSession(ctx context.Context, refreshToken string) error {
	args := m.Called(ctx, refreshToken)
	return args.Error(0)
}

type MockTokenManager struct {
	mock.Mock
}

func (m *MockTokenManager) GenerateToken(userID, tokenType string, duration time.Duration, deviceInfo domain.DeviceInfo) (string, error) {
	args := m.Called(userID, tokenType, duration, deviceInfo)
	return args.String(0), args.Error(1)
}

func (m *MockTokenManager) ParseToken(token, tokenType string) (*domain.Claims, error) {
	args := m.Called(token, tokenType)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*domain.Claims), args.Error(1)
}

func TestAuthHandler_RefreshToken(t *testing.T) {
	logger := zaptest.NewLogger(t)

	t.Run("without CSRF header -> 403", func(t *testing.T) {
		mockUserRepo := new(MockUserRepository)
		mockSessionRepo := new(MockSessionRepository)
		mockTokenManager := new(MockTokenManager)
		router := setupRouter(mockUserRepo, mockSessionRepo, mockTokenManager, logger)

		req, _ := http.NewRequest(http.MethodPost, "/auth/refresh", nil)
		req.AddCookie(&http.Cookie{Name: "csrf_token", Value: "test-csrf-cookie"})
		w := httptest.NewRecorder()

		router.ServeHTTP(w, req)

		assert.Equal(t, http.StatusForbidden, w.Code)
		var response httpresponse.ErrorResponse
		err := json.Unmarshal(w.Body.Bytes(), &response)
		assert.NoError(t, err)
		assert.False(t, response.Success)
		assert.Equal(t, "FORBIDDEN", response.Error.Code)
	})

	t.Run("with matching CSRF header -> 200 with new cookies", func(t *testing.T) {
		mockUserRepo := new(MockUserRepository)
		mockSessionRepo := new(MockSessionRepository)
		mockTokenManager := new(MockTokenManager)
		router := setupRouter(mockUserRepo, mockSessionRepo, mockTokenManager, logger)

		validSession := &domain.Session{
			UserID:           "test-user-id",
			RefreshTokenHash: "old-hash",
			IPAddress:        "127.0.0.1",
			UserAgent:        "TestBrowser",
			ExpiresAt:        time.Now().Add(1 * time.Hour),
		}

		mockSessionRepo.On("FindByRefreshToken", mock.Anything, "valid-refresh-token").Return(validSession, nil)
		mockTokenManager.On("GenerateToken", "test-user-id", "access", mock.Anything, mock.Anything).Return("new-access-token", nil)
		mockTokenManager.On("GenerateToken", "test-user-id", "refresh", mock.Anything, mock.Anything).Return("new-refresh-token", nil)
		mockSessionRepo.On("RotateRefreshToken", mock.Anything, "old-hash", mock.Anything).Return(nil)

		req, _ := http.NewRequest(http.MethodPost, "/auth/refresh", nil)
		req.Header.Set("X-CSRF-Token", "valid-csrf-token")
		req.AddCookie(&http.Cookie{Name: "csrf_token", Value: "valid-csrf-token"})
		req.AddCookie(&http.Cookie{Name: "refresh_token", Value: "valid-refresh-token"})
		w := httptest.NewRecorder()

		router.ServeHTTP(w, req)

		assert.Equal(t, http.StatusOK, w.Code)

		// Check response body
		var response httpresponse.SuccessResponse
		err := json.Unmarshal(w.Body.Bytes(), &response)
		assert.NoError(t, err)
		assert.True(t, response.Success)
		data := response.Data.(map[string]interface{})
		assert.Equal(t, "new-access-token", data["access_token"])

		// Check cookies
		cookies := w.Result().Cookies()
		assert.Len(t, cookies, 2) // refresh_token and csrf_token

		var foundRefresh, foundCsrf bool
		for _, c := range cookies {
			if c.Name == "refresh_token" {
				foundRefresh = true
				assert.Equal(t, "new-refresh-token", c.Value)
				assert.True(t, c.HttpOnly)
				assert.True(t, c.Secure)
			}
			if c.Name == "csrf_token" {
				foundCsrf = true
				assert.NotEmpty(t, c.Value)
				assert.False(t, c.HttpOnly)
				assert.True(t, c.Secure)
			}
		}
		assert.True(t, foundRefresh)
		assert.True(t, foundCsrf)
	})

	t.Run("expired or revoked refresh token -> 401", func(t *testing.T) {
		mockUserRepo := new(MockUserRepository)
		mockSessionRepo := new(MockSessionRepository)
		mockTokenManager := new(MockTokenManager)
		router := setupRouter(mockUserRepo, mockSessionRepo, mockTokenManager, logger)

		mockSessionRepo.On("FindByRefreshToken", mock.Anything, "expired-token").Return((*domain.Session)(nil), domain.ErrInvalidToken)

		req, _ := http.NewRequest(http.MethodPost, "/auth/refresh", nil)
		req.Header.Set("X-CSRF-Token", "valid-csrf-token")
		req.AddCookie(&http.Cookie{Name: "csrf_token", Value: "valid-csrf-token"})
		req.AddCookie(&http.Cookie{Name: "refresh_token", Value: "expired-token"})
		w := httptest.NewRecorder()

		router.ServeHTTP(w, req)

		assert.Equal(t, http.StatusUnauthorized, w.Code)
		var response httpresponse.ErrorResponse
		err := json.Unmarshal(w.Body.Bytes(), &response)
		assert.NoError(t, err)
		assert.False(t, response.Success)
		assert.Equal(t, "TOKEN_INVALID", response.Error.Code)
	})
}
