package http_test

import (
	"bytes"
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/alicebob/miniredis/v2"
	"github.com/gin-gonic/gin"
	"github.com/golang-jwt/jwt/v5"
	"github.com/google/uuid"
	"github.com/iqbaljlldn/nexus/apps/api/internal/identity/application"
	"github.com/iqbaljlldn/nexus/apps/api/internal/identity/domain"
	identityhttp "github.com/iqbaljlldn/nexus/apps/api/internal/identity/interface/http"
	"github.com/iqbaljlldn/nexus/apps/api/internal/platform/middleware"
	"github.com/iqbaljlldn/nexus/pkg/contextutil"
	pkgerrors "github.com/iqbaljlldn/nexus/pkg/errors"
	"github.com/iqbaljlldn/nexus/pkg/httpresponse"
	"github.com/iqbaljlldn/nexus/pkg/passwordhash"
	"github.com/iqbaljlldn/nexus/pkg/ratelimit"
	"github.com/redis/go-redis/v9"
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
	handler := identityhttp.NewAuthHandler(authService, nil)
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
		jsonValue, _ := json.Marshal(reqBody) //nolint:gosec // G117: test-only, intentionally marshaling password field
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
		jsonValue, _ := json.Marshal(reqBody) //nolint:gosec // G117: test-only, intentionally marshaling password field
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
		jsonValue, _ := json.Marshal(reqBody) //nolint:gosec // G117: test-only, intentionally marshaling password field
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

func (m *MockSessionRepository) RotateRefreshToken(ctx context.Context, oldSessionID, newSessionID, newTokenHash string) error {
	args := m.Called(ctx, oldSessionID, newSessionID, newTokenHash)
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

func (m *MockSessionRepository) RevokeAllSessions(ctx context.Context) error {
	args := m.Called(ctx)
	return args.Error(0)
}

func (m *MockSessionRepository) GetActiveSessions(ctx context.Context) ([]domain.Session, error) {
	args := m.Called(ctx)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).([]domain.Session), args.Error(1)
}

func (m *MockSessionRepository) RevokeSessionById(ctx context.Context, id uuid.UUID) (*domain.Session, error) {
	args := m.Called(ctx, id)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*domain.Session), args.Error(1)
}

func (m *MockSessionRepository) FindById(ctx context.Context, id uuid.UUID) (*domain.Session, error) {
	args := m.Called(ctx, id)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*domain.Session), args.Error(1)
}

type MockTokenManager struct {
	mock.Mock
}

func (m *MockTokenManager) GenerateToken(userID, sessionID, tokenType string, duration time.Duration, deviceInfo domain.DeviceInfo) (string, error) {
	args := m.Called(userID, sessionID, tokenType, duration, deviceInfo)
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
		req.AddCookie(&http.Cookie{Name: "csrf_token", Value: "test-csrf-cookie"}) //nolint:gosec // G124: test-only cookie
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
			ID:               "11111111-1111-1111-1111-111111111111",
			UserID:           "test-user-id",
			RefreshTokenHash: "old-hash",
			IPAddress:        "127.0.0.1",
			UserAgent:        "TestBrowser",
			ExpiresAt:        time.Now().Add(1 * time.Hour),
		}

		hashedToken, _ := passwordhash.Hash("valid-refresh-token")
		validSession.RefreshTokenHash = hashedToken

		mockTokenManager.On("ParseToken", "valid-refresh-token", "refresh").Return(&domain.Claims{
			RegisteredClaims: jwt.RegisteredClaims{ID: "11111111-1111-1111-1111-111111111111"},
		}, nil)
		mockSessionRepo.On("FindById", mock.Anything, uuid.MustParse("11111111-1111-1111-1111-111111111111")).Return(validSession, nil)
		mockTokenManager.On("GenerateToken", "test-user-id", mock.Anything, "access", mock.Anything, mock.Anything).Return("new-access-token", nil)
		mockTokenManager.On("GenerateToken", "test-user-id", mock.Anything, "refresh", mock.Anything, mock.Anything).Return("new-refresh-token", nil)
		mockSessionRepo.On("RotateRefreshToken", mock.Anything, "11111111-1111-1111-1111-111111111111", mock.Anything, mock.Anything).Return(nil)

		req, _ := http.NewRequest(http.MethodPost, "/auth/refresh", nil)
		req.Header.Set("X-CSRF-Token", "valid-csrf-token")
		req.AddCookie(&http.Cookie{Name: "csrf_token", Value: "valid-csrf-token"})       //nolint:gosec // G124: test-only cookie
		req.AddCookie(&http.Cookie{Name: "refresh_token", Value: "valid-refresh-token"}) //nolint:gosec // G124: test-only cookie
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

		mockTokenManager.On("ParseToken", "expired-token", "refresh").Return((*domain.Claims)(nil), domain.ErrInvalidToken)

		req, _ := http.NewRequest(http.MethodPost, "/auth/refresh", nil)
		req.Header.Set("X-CSRF-Token", "valid-csrf-token")
		req.AddCookie(&http.Cookie{Name: "csrf_token", Value: "valid-csrf-token"}) //nolint:gosec // G124: test-only cookie
		req.AddCookie(&http.Cookie{Name: "refresh_token", Value: "expired-token"}) //nolint:gosec // G124: test-only cookie
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

func TestAuthHandler_Logout(t *testing.T) {
	logger := zaptest.NewLogger(t)

	t.Run("without CSRF header -> 403", func(t *testing.T) {
		mockUserRepo := new(MockUserRepository)
		mockSessionRepo := new(MockSessionRepository)
		mockTokenManager := new(MockTokenManager)
		router := setupRouter(mockUserRepo, mockSessionRepo, mockTokenManager, logger)

		req, _ := http.NewRequest(http.MethodPost, "/auth/logout", nil)
		req.AddCookie(&http.Cookie{Name: "csrf_token", Value: "test-csrf-cookie"}) //nolint:gosec // G124: test-only cookie
		w := httptest.NewRecorder()

		router.ServeHTTP(w, req)

		assert.Equal(t, http.StatusForbidden, w.Code)
	})

	t.Run("with matching CSRF header -> 204 with cleared cookies", func(t *testing.T) {
		mockUserRepo := new(MockUserRepository)
		mockSessionRepo := new(MockSessionRepository)
		mockTokenManager := new(MockTokenManager)
		router := setupRouter(mockUserRepo, mockSessionRepo, mockTokenManager, logger)

		validSession := &domain.Session{
			ID:               "11111111-1111-1111-1111-111111111111",
			UserID:           "test-user-id",
			RefreshTokenHash: "old-hash",
			IPAddress:        "127.0.0.1",
			UserAgent:        "TestBrowser",
			ExpiresAt:        time.Now().Add(1 * time.Hour),
		}

		hashedToken, _ := passwordhash.Hash("valid-refresh-token")
		validSession.RefreshTokenHash = hashedToken

		mockTokenManager.On("ParseToken", "valid-refresh-token", "refresh").Return(&domain.Claims{
			RegisteredClaims: jwt.RegisteredClaims{ID: "11111111-1111-1111-1111-111111111111"},
		}, nil)
		mockSessionRepo.On("FindById", mock.Anything, uuid.MustParse("11111111-1111-1111-1111-111111111111")).Return(validSession, nil)
		mockSessionRepo.On("RevokeSessionById", mock.Anything, uuid.MustParse("11111111-1111-1111-1111-111111111111")).Return(validSession, nil)

		req, _ := http.NewRequest(http.MethodPost, "/auth/logout", nil)
		req.Header.Set("X-CSRF-Token", "valid-csrf-token")
		req.AddCookie(&http.Cookie{Name: "csrf_token", Value: "valid-csrf-token"})       //nolint:gosec // G124: test-only cookie
		req.AddCookie(&http.Cookie{Name: "refresh_token", Value: "valid-refresh-token"}) //nolint:gosec // G124: test-only cookie
		w := httptest.NewRecorder()

		router.ServeHTTP(w, req)

		assert.Equal(t, http.StatusNoContent, w.Code)

		// Check cookies are cleared
		cookies := w.Result().Cookies()
		assert.Len(t, cookies, 2) // refresh_token and csrf_token

		var foundRefresh, foundCsrf bool
		for _, c := range cookies {
			if c.Name == "refresh_token" {
				foundRefresh = true
				assert.Equal(t, "", c.Value)  // Value should be empty
				assert.Equal(t, -1, c.MaxAge) // MaxAge should be -1
			}
			if c.Name == "csrf_token" {
				foundCsrf = true
				assert.Equal(t, "", c.Value)  // Value should be empty
				assert.Equal(t, -1, c.MaxAge) // MaxAge should be -1
			}
		}
		assert.True(t, foundRefresh)
		assert.True(t, foundCsrf)
	})
}

func TestAuthHandler_ListSessions(t *testing.T) {
	logger := zaptest.NewLogger(t)

	t.Run("success", func(t *testing.T) {
		mockUserRepo := new(MockUserRepository)
		mockSessionRepo := new(MockSessionRepository)
		mockTokenManager := new(MockTokenManager)
		_ = setupRouter(mockUserRepo, mockSessionRepo, mockTokenManager, logger)

		sessions := []domain.Session{
			{
				ID:               "session-1",
				UserID:           "test-user-id",
				RefreshTokenHash: "hash-1", // Should not be in response
				UserAgent:        "Chrome",
				IPAddress:        "192.168.1.1",
				CreatedAt:        time.Now(),
			},
			{
				ID:               "session-2",
				UserID:           "test-user-id",
				RefreshTokenHash: "hash-2", // Should not be in response
				UserAgent:        "Firefox",
				IPAddress:        "192.168.1.2",
				CreatedAt:        time.Now(),
			},
		}

		mockSessionRepo.On("GetActiveSessions", mock.Anything).Return(sessions, nil)

		// Create a mock token that the auth middleware will verify
		token := "mock-token"
		mockTokenManager.On("ParseToken", token, "access").Return(&domain.Claims{
			UserID: "test-user-id",
		}, nil)

		// For Auth middleware to work, it needs jwt.Verify to pass, but since jwt is in pkg/jwt and uses a global secret,
		// we can't easily mock it without setting the env. In Gin test, we can just bypass the middleware or set a valid token.
		// Wait, Auth middleware parses with jwt.Verify, so we must set NEXUS_API_JWT_SECRET and generate a real token.
		// Since we didn't do this, we can just test the handler function directly, or let the router test run if we can.
		// Actually, in TestAuthHandler_ListSessions, we just want to ensure DTO is mapped correctly.
		w := httptest.NewRecorder()
		c, _ := gin.CreateTestContext(w)
		c.Request, _ = http.NewRequest(http.MethodGet, "/auth/sessions", nil)
		// Inject the UserID directly to simulate the middleware
		c.Request = c.Request.WithContext(contextutil.WithUserID(c.Request.Context(), uuid.MustParse("00000000-0000-0000-0000-000000000000")))

		authService := application.NewAuthService(mockUserRepo, mockSessionRepo, mockTokenManager, logger)
		handler := identityhttp.NewAuthHandler(authService, nil)

		handler.ListSessions(c)

		assert.Equal(t, http.StatusOK, w.Code)
		var response httpresponse.SuccessResponse
		err := json.Unmarshal(w.Body.Bytes(), &response)
		assert.NoError(t, err)

		sessionList := response.Data.([]interface{})
		assert.Len(t, sessionList, 2)

		session1 := sessionList[0].(map[string]interface{})
		assert.Equal(t, "session-1", session1["id"])
		assert.Equal(t, "Chrome", session1["user_agent"])
		assert.Equal(t, "192.168.1.1", session1["ip_address"])
		assert.NotContains(t, session1, "refresh_token_hash")
		assert.NotContains(t, session1, "RefreshTokenHash")
	})
}

func TestAuthHandler_LogoutAll(t *testing.T) {
	logger := zaptest.NewLogger(t)

	t.Run("success -> 204 with cleared cookies", func(t *testing.T) {
		mockUserRepo := new(MockUserRepository)
		mockSessionRepo := new(MockSessionRepository)
		mockTokenManager := new(MockTokenManager)

		mockSessionRepo.On("RevokeAllSessions", mock.Anything).Return(nil)

		w := httptest.NewRecorder()
		c, _ := gin.CreateTestContext(w)
		c.Request, _ = http.NewRequest(http.MethodPost, "/auth/logout-all", nil)
		c.Request = c.Request.WithContext(contextutil.WithUserID(c.Request.Context(), uuid.MustParse("00000000-0000-0000-0000-000000000000")))

		authService := application.NewAuthService(mockUserRepo, mockSessionRepo, mockTokenManager, logger)
		handler := identityhttp.NewAuthHandler(authService, nil)

		handler.LogoutAll(c)

		assert.Equal(t, http.StatusNoContent, c.Writer.Status())

		cookies := w.Result().Cookies()
		assert.Len(t, cookies, 2)

		var foundRefresh, foundCsrf bool
		for _, cookie := range cookies {
			if cookie.Name == "refresh_token" {
				foundRefresh = true
				assert.Equal(t, "", cookie.Value)
				assert.Equal(t, -1, cookie.MaxAge)
			}
			if cookie.Name == "csrf_token" {
				foundCsrf = true
				assert.Equal(t, "", cookie.Value)
				assert.Equal(t, -1, cookie.MaxAge)
			}
		}
		assert.True(t, foundRefresh)
		assert.True(t, foundCsrf)
	})
}

func TestAuthHandler_RevokeSessionById(t *testing.T) {
	logger := zaptest.NewLogger(t)

	t.Run("success -> 200 with cleared cookies", func(t *testing.T) {
		mockUserRepo := new(MockUserRepository)
		mockSessionRepo := new(MockSessionRepository)
		mockTokenManager := new(MockTokenManager)

		userID := uuid.MustParse("00000000-0000-0000-0000-000000000000")
		sessionID := uuid.MustParse("11111111-1111-1111-1111-111111111111")

		session := &domain.Session{
			ID:     sessionID.String(),
			UserID: userID.String(),
		}

		mockSessionRepo.On("FindById", mock.Anything, sessionID).Return(session, nil)
		mockSessionRepo.On("RevokeSessionById", mock.Anything, sessionID).Return(session, nil)

		w := httptest.NewRecorder()
		c, _ := gin.CreateTestContext(w)
		c.Request, _ = http.NewRequest(http.MethodPost, "/auth/sessions/"+sessionID.String()+"/revoke", nil)
		c.Request = c.Request.WithContext(contextutil.WithUserID(c.Request.Context(), userID))
		c.Params = gin.Params{{Key: "id", Value: sessionID.String()}}

		authService := application.NewAuthService(mockUserRepo, mockSessionRepo, mockTokenManager, logger)
		handler := identityhttp.NewAuthHandler(authService, nil)

		handler.RevokeSessionById(c)

		assert.Equal(t, http.StatusOK, c.Writer.Status())

		cookies := w.Result().Cookies()
		assert.Len(t, cookies, 2)

		var foundRefresh, foundCsrf bool
		for _, cookie := range cookies {
			if cookie.Name == "refresh_token" {
				foundRefresh = true
				assert.Equal(t, "", cookie.Value)
				assert.Equal(t, -1, cookie.MaxAge)
			}
			if cookie.Name == "csrf_token" {
				foundCsrf = true
				assert.Equal(t, "", cookie.Value)
				assert.Equal(t, -1, cookie.MaxAge)
			}
		}
		assert.True(t, foundRefresh)
		assert.True(t, foundCsrf)
	})

	t.Run("forbidden -> 403 when revoking others session", func(t *testing.T) {
		mockUserRepo := new(MockUserRepository)
		mockSessionRepo := new(MockSessionRepository)
		mockTokenManager := new(MockTokenManager)

		userID := uuid.MustParse("00000000-0000-0000-0000-000000000000")
		otherUserID := "22222222-2222-2222-2222-222222222222"
		sessionID := uuid.MustParse("11111111-1111-1111-1111-111111111111")

		session := &domain.Session{
			ID:     sessionID.String(),
			UserID: otherUserID,
		}

		mockSessionRepo.On("FindById", mock.Anything, sessionID).Return(session, nil)

		w := httptest.NewRecorder()
		c, _ := gin.CreateTestContext(w)
		c.Request, _ = http.NewRequest(http.MethodPost, "/auth/sessions/"+sessionID.String()+"/revoke", nil)
		c.Request = c.Request.WithContext(contextutil.WithUserID(c.Request.Context(), userID))
		c.Params = gin.Params{{Key: "id", Value: sessionID.String()}}

		authService := application.NewAuthService(mockUserRepo, mockSessionRepo, mockTokenManager, logger)
		handler := identityhttp.NewAuthHandler(authService, nil)

		handler.RevokeSessionById(c)

		assert.Equal(t, http.StatusForbidden, c.Writer.Status())
		mockSessionRepo.AssertNotCalled(t, "RevokeSessionById")
	})
}

func setupRouterWithRateLimiter(userRepo domain.UserRepository, sessionRepo domain.SessionRepository, tokenManager domain.TokenManager, logger *zap.Logger, redisClient *redis.Client) *gin.Engine {
	gin.SetMode(gin.TestMode)

	r := gin.New()
	r.Use(func(c *gin.Context) {
		c.Next()
		if len(c.Errors) > 0 {
			err := c.Errors.Last().Err
			httpresponse.Error(c, err)
		}
	})

	limiter := ratelimit.New(redisClient)
	loginRateLimiter := middleware.NewLoginRateLimiter(limiter, redisClient)

	authService := application.NewAuthService(userRepo, sessionRepo, tokenManager, logger)
	handler := identityhttp.NewAuthHandler(authService, loginRateLimiter)
	handler.RegisterRoutes(r.Group("/"))

	return r
}

func TestAuthHandler_Login_RateLimiting(t *testing.T) {
	logger := zaptest.NewLogger(t)
	mr := miniredis.RunT(t)
	redisClient := redis.NewClient(&redis.Options{Addr: mr.Addr()})
	t.Cleanup(func() { _ = redisClient.Close() })

	t.Run("progressive lockout after 5 failed attempts", func(t *testing.T) {
		mr.FlushAll()
		mockRepo := new(MockUserRepository)
		router := setupRouterWithRateLimiter(mockRepo, nil, nil, logger, redisClient)

		// Setup mock to always return invalid credentials
		mockRepo.On("FindByEmail", mock.Anything, "test@example.com").Return(&domain.User{
			PasswordHash: "some-hash",
		}, nil)

		reqBody := identityhttp.LoginRequest{
			Identifier: "test@example.com",
			Password:   "wrongpassword",
		}
		jsonValue, _ := json.Marshal(reqBody) //nolint:gosec // G117: test-only

		// 1 to 5 attempts should return 400 Invalid Credentials
		for i := 0; i < 5; i++ {
			req, _ := http.NewRequest(http.MethodPost, "/auth/login", bytes.NewBuffer(jsonValue))
			w := httptest.NewRecorder()
			router.ServeHTTP(w, req)
			assert.Equal(t, http.StatusBadRequest, w.Code, "attempt %d", i+1)

			var response httpresponse.ErrorResponse
			err := json.Unmarshal(w.Body.Bytes(), &response)
			assert.NoError(t, err)
			assert.Equal(t, pkgerrors.CodeInvalidCredentials, response.Error.Code)
		}

		// 6th attempt should return 429 Rate Limit Exceeded
		req, _ := http.NewRequest(http.MethodPost, "/auth/login", bytes.NewBuffer(jsonValue))
		w := httptest.NewRecorder()
		router.ServeHTTP(w, req)

		assert.Equal(t, http.StatusTooManyRequests, w.Code)
		assert.Equal(t, "300", w.Header().Get("Retry-After")) // First lockout is 5 mins (300 secs)

		var response httpresponse.ErrorResponse
		err := json.Unmarshal(w.Body.Bytes(), &response)
		assert.NoError(t, err)
		assert.Equal(t, pkgerrors.CodeRateLimitExceeded, response.Error.Code)

		// Move forward in time by 5 minutes + 1 sec
		mr.FastForward(5*time.Minute + time.Second)

		// 7th attempt (after lockout expires) should be allowed to try, but fail auth again
		req, _ = http.NewRequest(http.MethodPost, "/auth/login", bytes.NewBuffer(jsonValue))
		w = httptest.NewRecorder()
		router.ServeHTTP(w, req)
		assert.Equal(t, http.StatusBadRequest, w.Code)

		// 8th attempt should immediately hit the 2nd lockout tier (15 minutes)
		// because the sliding window counter reset, but the active lockout will trigger again
		// Wait, the logic is: after lockout, the sliding window is cleared. So it takes another 5 attempts?
		// No, let's verify how the middleware behaves. If they fail 5 MORE times, they get a 15 min lockout.
		for i := 0; i < 4; i++ {
			req, _ = http.NewRequest(http.MethodPost, "/auth/login", bytes.NewBuffer(jsonValue))
			w = httptest.NewRecorder()
			router.ServeHTTP(w, req)
			assert.Equal(t, http.StatusBadRequest, w.Code)
		}

		// Now it should hit the 2nd tier
		req, _ = http.NewRequest(http.MethodPost, "/auth/login", bytes.NewBuffer(jsonValue))
		w = httptest.NewRecorder()
		router.ServeHTTP(w, req)
		assert.Equal(t, http.StatusTooManyRequests, w.Code)
		assert.Equal(t, "900", w.Header().Get("Retry-After")) // 15 mins (900 secs)
	})
}
