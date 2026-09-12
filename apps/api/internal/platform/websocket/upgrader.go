package websocket

import (
	"net/http"
	"os"
	"strings"

	"github.com/gorilla/websocket"
)

const (
	// CloseCodeUnauthorized is the custom WebSocket close code for authentication failure (Security Design §7).
	CloseCodeUnauthorized = 4001

	// Buffer configurations
	defaultReadBufferSize  = 4096
	defaultWriteBufferSize = 4096
)

// CheckOriginFunc returns whether a given request origin is allowed based on CORS_ALLOWED_ORIGINS.
func CheckOriginFunc(r *http.Request) bool {
	origin := r.Header.Get("Origin")
	if origin == "" {
		return true
	}

	allowedOriginsEnv := os.Getenv("CORS_ALLOWED_ORIGINS")
	if allowedOriginsEnv == "" {
		// Default fallback for dev environment: allow requesting origin
		return true
	}

	allowedOrigins := strings.Split(allowedOriginsEnv, ",")
	for _, allowed := range allowedOrigins {
		trimmed := strings.TrimSpace(allowed)
		if trimmed == "*" || trimmed == origin {
			return true
		}
	}

	return false
}

// NewUpgrader creates a new websocket.Upgrader with configured buffers and origin check.
func NewUpgrader() *websocket.Upgrader {
	return &websocket.Upgrader{
		ReadBufferSize:  defaultReadBufferSize,
		WriteBufferSize: defaultWriteBufferSize,
		CheckOrigin:     CheckOriginFunc,
	}
}
