package http_test

import (
	"bytes"
	"database/sql"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"os"
	"testing"

	"github.com/alicebob/miniredis/v2"
	"github.com/gin-gonic/gin"
	"github.com/iqbaljlldn/nexus/apps/api/internal/identity/application"
	"github.com/iqbaljlldn/nexus/apps/api/internal/identity/infrastructure"
	identityhttp "github.com/iqbaljlldn/nexus/apps/api/internal/identity/interface/http"
	"github.com/iqbaljlldn/nexus/apps/api/internal/platform/middleware"
	"github.com/iqbaljlldn/nexus/pkg/httpresponse"
	"github.com/iqbaljlldn/nexus/pkg/ratelimit"
	_ "github.com/jackc/pgx/v5/stdlib"
	"github.com/redis/go-redis/v9"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.uber.org/zap/zaptest"
)

func setupIntegrationEnvironment(t *testing.T) (*gin.Engine, *sql.DB, *miniredis.Miniredis) {
	t.Helper()
	dsn := os.Getenv("TEST_DATABASE_URL")
	if dsn == "" {
		t.Skip("TEST_DATABASE_URL not set, skipping integration test")
	}

	// 1. Setup Postgres
	db, err := sql.Open("pgx", dsn)
	require.NoError(t, err)
	t.Cleanup(func() { _ = db.Close() })

	// Clean tables before test
	_, err = db.Exec("TRUNCATE TABLE users, sessions CASCADE")
	require.NoError(t, err)

	// 2. Setup Redis (Miniredis)
	mr := miniredis.RunT(t)
	redisClient := redis.NewClient(&redis.Options{Addr: mr.Addr()})
	t.Cleanup(func() { _ = redisClient.Close() })

	// 3. Setup Repositories
	querier := infrastructure.New(db)
	userRepo := infrastructure.NewPostgresUserRepository(querier)
	sessionRepo := infrastructure.NewPostgresSessionRepository(db)

	// 4. Setup Services & Rate Limiter
	logger := zaptest.NewLogger(t)
	tokenManager := infrastructure.NewJWTTokenManager("nexus-api", "nexus-client")

	// Ensure the pkg/jwt has a secret for middleware verification
	_ = os.Setenv("NEXUS_API_JWT_SECRET", "test-integration-secret-key-that-is-long-enough")

	limiter := ratelimit.New(redisClient)
	loginRateLimiter := middleware.NewLoginRateLimiter(limiter, redisClient)

	authService := application.NewAuthService(userRepo, sessionRepo, tokenManager, logger)
	handler := identityhttp.NewAuthHandler(authService, loginRateLimiter)

	// 5. Setup Gin Router
	gin.SetMode(gin.TestMode)
	r := gin.New()
	r.Use(func(c *gin.Context) {
		c.Next()
		if len(c.Errors) > 0 {
			err := c.Errors.Last().Err
			httpresponse.Error(c, err)
		}
	})

	handler.RegisterRoutes(r.Group("/api/v1"))
	return r, db, mr
}

// getCookie is a helper to find a cookie by name from the response
func getCookie(res *http.Response, name string) *http.Cookie {
	for _, c := range res.Cookies() {
		if c.Name == name {
			return c
		}
	}
	return nil
}

func TestAuthFlow_EndToEnd(t *testing.T) {
	router, _, _ := setupIntegrationEnvironment(t)
	var accessToken string
	var refreshTokenCookie *http.Cookie
	var csrfTokenCookie *http.Cookie
	var csrfTokenValue string

	t.Run("1. Register", func(t *testing.T) {
		reqBody := identityhttp.RegisterRequest{
			Email:       "e2e@example.com",
			Username:    "e2euser",
			DisplayName: "E2E User",
			Password:    "password123",
		}
		jsonValue, _ := json.Marshal(reqBody) //nolint:gosec // G117: test-only
		req, _ := http.NewRequest(http.MethodPost, "/api/v1/auth/register", bytes.NewBuffer(jsonValue))
		w := httptest.NewRecorder()
		router.ServeHTTP(w, req)

		assert.Equal(t, http.StatusCreated, w.Code)
	})

	t.Run("2. Login", func(t *testing.T) {
		reqBody := identityhttp.LoginRequest{
			Identifier: "e2e@example.com",
			Password:   "password123",
		}
		jsonValue, _ := json.Marshal(reqBody) //nolint:gosec // G117: test-only
		req, _ := http.NewRequest(http.MethodPost, "/api/v1/auth/login", bytes.NewBuffer(jsonValue))
		w := httptest.NewRecorder()
		router.ServeHTTP(w, req)

		require.Equal(t, http.StatusOK, w.Code)

		var response httpresponse.SuccessResponse
		err := json.Unmarshal(w.Body.Bytes(), &response)
		require.NoError(t, err)

		data := response.Data.(map[string]interface{})
		accessToken = data["access_token"].(string)
		require.NotEmpty(t, accessToken)

		res := w.Result()
		refreshTokenCookie = getCookie(res, "refresh_token")
		require.NotNil(t, refreshTokenCookie)
		require.NotEmpty(t, refreshTokenCookie.Value)

		csrfTokenCookie = getCookie(res, "csrf_token")
		require.NotNil(t, csrfTokenCookie)
		csrfTokenValue = csrfTokenCookie.Value
		require.NotEmpty(t, csrfTokenValue)
	})

	t.Run("3. Protected Access", func(t *testing.T) {
		req, _ := http.NewRequest(http.MethodGet, "/api/v1/auth/sessions", nil)
		req.Header.Set("Authorization", "Bearer "+accessToken)
		w := httptest.NewRecorder()
		router.ServeHTTP(w, req)

		require.Equal(t, http.StatusOK, w.Code)
		var response httpresponse.SuccessResponse
		err := json.Unmarshal(w.Body.Bytes(), &response)
		require.NoError(t, err)

		sessions := response.Data.([]interface{})
		assert.Len(t, sessions, 1, "Should have 1 active session")
	})

	var oldRefreshTokenCookie *http.Cookie

	t.Run("4. Refresh Token", func(t *testing.T) {
		oldRefreshTokenCookie = refreshTokenCookie // Save to test replay later

		req, _ := http.NewRequest(http.MethodPost, "/api/v1/auth/refresh", nil)
		// Set cookies
		req.AddCookie(refreshTokenCookie)
		req.AddCookie(csrfTokenCookie)
		// Set CSRF header
		req.Header.Set("X-CSRF-Token", csrfTokenValue)

		w := httptest.NewRecorder()
		router.ServeHTTP(w, req)

		require.Equal(t, http.StatusOK, w.Code)

		var response httpresponse.SuccessResponse
		err := json.Unmarshal(w.Body.Bytes(), &response)
		require.NoError(t, err)

		data := response.Data.(map[string]interface{})
		accessToken = data["access_token"].(string)
		require.NotEmpty(t, accessToken)

		// Capture new cookies
		res := w.Result()
		newRefreshCookie := getCookie(res, "refresh_token")
		require.NotNil(t, newRefreshCookie)

		newCsrfCookie := getCookie(res, "csrf_token")
		require.NotNil(t, newCsrfCookie)

		// Ensure tokens rotated
		assert.NotEqual(t, refreshTokenCookie.Value, newRefreshCookie.Value)

		refreshTokenCookie = newRefreshCookie
		csrfTokenCookie = newCsrfCookie
		csrfTokenValue = newCsrfCookie.Value
	})

	t.Run("5. Verify Old Token Revoked", func(t *testing.T) {
		req, _ := http.NewRequest(http.MethodPost, "/api/v1/auth/refresh", nil)
		// Try using the OLD refresh token to simulate replay attack or stolen token
		req.AddCookie(oldRefreshTokenCookie)
		req.AddCookie(csrfTokenCookie)
		req.Header.Set("X-CSRF-Token", csrfTokenValue)

		w := httptest.NewRecorder()
		router.ServeHTTP(w, req)

		require.Equal(t, http.StatusUnauthorized, w.Code)
	})

	t.Run("6. Logout All", func(t *testing.T) {
		req, _ := http.NewRequest(http.MethodPost, "/api/v1/auth/logout-all", nil)
		req.Header.Set("Authorization", "Bearer "+accessToken) // Protected endpoint
		req.AddCookie(csrfTokenCookie)                         // Not strictly required by logout-all, but good practice

		w := httptest.NewRecorder()
		router.ServeHTTP(w, req)

		require.Equal(t, http.StatusNoContent, w.Code)

		res := w.Result()
		clearedRefresh := getCookie(res, "refresh_token")
		require.NotNil(t, clearedRefresh)
		assert.Equal(t, "", clearedRefresh.Value)
		assert.Equal(t, -1, clearedRefresh.MaxAge)
	})

	t.Run("7. Verify Logout", func(t *testing.T) {
		// Try to refresh using the newest token that was just logged out
		req, _ := http.NewRequest(http.MethodPost, "/api/v1/auth/refresh", nil)
		req.AddCookie(refreshTokenCookie)
		req.AddCookie(csrfTokenCookie)
		req.Header.Set("X-CSRF-Token", csrfTokenValue)

		w := httptest.NewRecorder()
		router.ServeHTTP(w, req)

		// Should be unauthorized since session was revoked by logout-all
		require.Equal(t, http.StatusUnauthorized, w.Code)
	})
}
