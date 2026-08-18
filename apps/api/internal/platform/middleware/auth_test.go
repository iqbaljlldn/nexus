package middleware_test

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"os"
	"testing"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/golang-jwt/jwt/v5"
	"github.com/iqbaljlldn/nexus/apps/api/internal/platform/middleware"
	"github.com/iqbaljlldn/nexus/pkg/httpresponse"
	pkgjwt "github.com/iqbaljlldn/nexus/pkg/jwt"
	"github.com/stretchr/testify/assert"
)

func setupRouter() *gin.Engine {
	gin.SetMode(gin.TestMode)
	r := gin.New()

	// Protected route
	r.GET("/protected", middleware.Auth(), func(c *gin.Context) {
		userID, exists := c.Get("user_id")
		if !exists {
			c.String(http.StatusInternalServerError, "user_id not found in context")
			return
		}
		c.String(http.StatusOK, userID.(string))
	})

	return r
}

func TestAuthMiddleware(t *testing.T) {
	os.Setenv("NEXUS_API_JWT_SECRET", "test_secret_key")
	defer os.Unsetenv("NEXUS_API_JWT_SECRET")

	router := setupRouter()

	t.Run("missing header", func(t *testing.T) {
		req, _ := http.NewRequest(http.MethodGet, "/protected", nil)
		w := httptest.NewRecorder()
		router.ServeHTTP(w, req)

		assert.Equal(t, http.StatusUnauthorized, w.Code)

		var response httpresponse.ErrorResponse
		err := json.Unmarshal(w.Body.Bytes(), &response)
		assert.NoError(t, err)
		assert.False(t, response.Success)
		assert.Equal(t, "USER_UNAUTHORIZED", response.Error.Code)
	})

	t.Run("malformed header", func(t *testing.T) {
		req, _ := http.NewRequest(http.MethodGet, "/protected", nil)
		req.Header.Set("Authorization", "InvalidFormat")
		w := httptest.NewRecorder()
		router.ServeHTTP(w, req)

		assert.Equal(t, http.StatusUnauthorized, w.Code)

		var response httpresponse.ErrorResponse
		err := json.Unmarshal(w.Body.Bytes(), &response)
		assert.NoError(t, err)
		assert.False(t, response.Success)
		assert.Equal(t, "USER_UNAUTHORIZED", response.Error.Code)
	})

	t.Run("token expired", func(t *testing.T) {
		claims := pkgjwt.BaseClaims{
			UserID: "user123",
			RegisteredClaims: jwt.RegisteredClaims{
				IssuedAt:  jwt.NewNumericDate(time.Now().Add(-30 * time.Minute)),
				ExpiresAt: jwt.NewNumericDate(time.Now().Add(-15 * time.Minute)),
			},
		}
		tokenStr, _ := pkgjwt.Sign(claims)

		req, _ := http.NewRequest(http.MethodGet, "/protected", nil)
		req.Header.Set("Authorization", "Bearer "+tokenStr)
		w := httptest.NewRecorder()
		router.ServeHTTP(w, req)

		assert.Equal(t, http.StatusUnauthorized, w.Code)

		var response httpresponse.ErrorResponse
		err := json.Unmarshal(w.Body.Bytes(), &response)
		assert.NoError(t, err)
		assert.False(t, response.Success)
		assert.Equal(t, "TOKEN_EXPIRED", response.Error.Code)
	})

	t.Run("valid token", func(t *testing.T) {
		claims := pkgjwt.BaseClaims{
			UserID: "user123",
			RegisteredClaims: jwt.RegisteredClaims{
				IssuedAt:  jwt.NewNumericDate(time.Now()),
				ExpiresAt: jwt.NewNumericDate(time.Now().Add(15 * time.Minute)),
			},
		}
		tokenStr, _ := pkgjwt.Sign(claims)

		req, _ := http.NewRequest(http.MethodGet, "/protected", nil)
		req.Header.Set("Authorization", "Bearer "+tokenStr)
		w := httptest.NewRecorder()
		router.ServeHTTP(w, req)

		assert.Equal(t, http.StatusOK, w.Code)
		assert.Equal(t, "user123", w.Body.String())
	})
}
