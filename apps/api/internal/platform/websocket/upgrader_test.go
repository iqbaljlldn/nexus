package websocket_test

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/gin-gonic/gin"
	jwt_lib "github.com/golang-jwt/jwt/v5"
	"github.com/google/uuid"
	"github.com/gorilla/websocket"
	ws "github.com/iqbaljlldn/nexus/apps/api/internal/platform/websocket"
	"github.com/iqbaljlldn/nexus/pkg/jwt"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.uber.org/zap"
)

func setupTestWSServer(t *testing.T, logger *zap.Logger) (*httptest.Server, *ws.ConnectionRegistry, *ws.Handler) {
	gin.SetMode(gin.TestMode)
	r := gin.New()

	registry := ws.NewConnectionRegistry(logger)
	handler := ws.NewHandler(registry, logger)

	api := r.Group("/api/v1")
	handler.RegisterRoutes(api)

	server := httptest.NewServer(r)
	t.Cleanup(func() {
		server.Close()
	})

	return server, registry, handler
}

func createValidToken(t *testing.T, userID uuid.UUID) string {
	claims := jwt.BaseClaims{
		UserID: userID.String(),
		RegisteredClaims: jwt_lib.RegisteredClaims{
			ExpiresAt: jwt_lib.NewNumericDate(time.Now().Add(1 * time.Hour)),
			IssuedAt:  jwt_lib.NewNumericDate(time.Now()),
		},
	}
	token, err := jwt.Sign(claims)
	require.NoError(t, err)
	return token
}

func createExpiredToken(t *testing.T, userID uuid.UUID) string {
	claims := jwt.BaseClaims{
		UserID: userID.String(),
		RegisteredClaims: jwt_lib.RegisteredClaims{
			ExpiresAt: jwt_lib.NewNumericDate(time.Now().Add(-1 * time.Hour)),
			IssuedAt:  jwt_lib.NewNumericDate(time.Now().Add(-2 * time.Hour)),
		},
	}
	token, err := jwt.Sign(claims)
	require.NoError(t, err)
	return token
}

func TestWebSocketUpgrade_ValidToken(t *testing.T) {
	logger := zap.NewNop()
	server, registry, _ := setupTestWSServer(t, logger)

	userID := uuid.New()
	token := createValidToken(t, userID)

	wsURL := "ws" + strings.TrimPrefix(server.URL, "http") + "/api/v1/ws?token=" + token

	dialer := websocket.DefaultDialer
	conn, resp, err := dialer.Dial(wsURL, nil)
	require.NoError(t, err)
	defer func() {
		_ = conn.Close()
	}()
	require.Equal(t, http.StatusSwitchingProtocols, resp.StatusCode)

	// Verify connection is registered
	assert.Eventually(t, func() bool {
		return registry.GetUserConnectionsCount(userID) == 1
	}, 500*time.Millisecond, 10*time.Millisecond)

	assert.True(t, registry.IsUserConnected(userID))
	assert.Equal(t, 1, registry.GetActiveConnectionsCount())
}

func TestWebSocketUpgrade_MissingToken(t *testing.T) {
	logger := zap.NewNop()
	server, registry, _ := setupTestWSServer(t, logger)

	wsURL := "ws" + strings.TrimPrefix(server.URL, "http") + "/api/v1/ws"

	dialer := websocket.DefaultDialer
	conn, _, err := dialer.Dial(wsURL, nil)
	require.NoError(t, err, "Upgrade handshake itself should succeed before sending 4001 close frame")
	defer func() {
		_ = conn.Close()
	}()

	// Connection should receive close code 4001
	_, _, err = conn.ReadMessage()
	require.Error(t, err)

	var closeErr *websocket.CloseError
	require.ErrorAs(t, err, &closeErr)
	assert.Equal(t, ws.CloseCodeUnauthorized, closeErr.Code)
	assert.Equal(t, 0, registry.GetActiveConnectionsCount())
}

func TestWebSocketUpgrade_InvalidToken(t *testing.T) {
	logger := zap.NewNop()
	server, registry, _ := setupTestWSServer(t, logger)

	wsURL := "ws" + strings.TrimPrefix(server.URL, "http") + "/api/v1/ws?token=invalid.jwt.token"

	dialer := websocket.DefaultDialer
	conn, _, err := dialer.Dial(wsURL, nil)
	require.NoError(t, err)
	defer func() {
		_ = conn.Close()
	}()

	_, _, err = conn.ReadMessage()
	require.Error(t, err)

	var closeErr *websocket.CloseError
	require.ErrorAs(t, err, &closeErr)
	assert.Equal(t, ws.CloseCodeUnauthorized, closeErr.Code)
	assert.Equal(t, 0, registry.GetActiveConnectionsCount())
}

func TestWebSocketUpgrade_ExpiredToken(t *testing.T) {
	logger := zap.NewNop()
	server, registry, _ := setupTestWSServer(t, logger)

	userID := uuid.New()
	token := createExpiredToken(t, userID)

	wsURL := "ws" + strings.TrimPrefix(server.URL, "http") + "/api/v1/ws?token=" + token

	dialer := websocket.DefaultDialer
	conn, _, err := dialer.Dial(wsURL, nil)
	require.NoError(t, err)
	defer func() {
		_ = conn.Close()
	}()

	_, _, err = conn.ReadMessage()
	require.Error(t, err)

	var closeErr *websocket.CloseError
	require.ErrorAs(t, err, &closeErr)
	assert.Equal(t, ws.CloseCodeUnauthorized, closeErr.Code)
	assert.Equal(t, 0, registry.GetActiveConnectionsCount())
}

func TestWebSocketUpgrade_OriginValidation(t *testing.T) {
	t.Setenv("CORS_ALLOWED_ORIGINS", "https://app.nexus.dev,https://nexus.dev")

	logger := zap.NewNop()
	server, _, _ := setupTestWSServer(t, logger)

	userID := uuid.New()
	token := createValidToken(t, userID)
	wsURL := "ws" + strings.TrimPrefix(server.URL, "http") + "/api/v1/ws?token=" + token

	t.Run("Allowed Origin", func(t *testing.T) {
		headers := http.Header{}
		headers.Set("Origin", "https://app.nexus.dev")

		conn, resp, err := websocket.DefaultDialer.Dial(wsURL, headers)
		require.NoError(t, err)
		defer func() {
			_ = conn.Close()
		}()
		assert.Equal(t, http.StatusSwitchingProtocols, resp.StatusCode)
	})

	t.Run("Disallowed Origin", func(t *testing.T) {
		headers := http.Header{}
		headers.Set("Origin", "https://malicious-site.com")

		_, resp, err := websocket.DefaultDialer.Dial(wsURL, headers)
		require.Error(t, err)
		assert.Equal(t, http.StatusForbidden, resp.StatusCode)
	})
}
