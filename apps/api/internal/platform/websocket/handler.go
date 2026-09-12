package websocket

import (
	"errors"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"github.com/gorilla/websocket"
	"github.com/iqbaljlldn/nexus/pkg/jwt"
	"github.com/iqbaljlldn/nexus/pkg/router"
	"go.uber.org/zap"
)

// Handler handles WebSocket upgrade HTTP requests and registers new connections.
type Handler struct {
	upgrader       *websocket.Upgrader
	registry       *ConnectionRegistry
	logger         *zap.Logger
	connConfig     *ConnectionConfig
	messageHandler MessageHandler
}

var _ router.ModuleRouter = (*Handler)(nil)

// NewHandler creates a new WebSocket Handler.
func NewHandler(
	registry *ConnectionRegistry,
	logger *zap.Logger,
) *Handler {
	return &Handler{
		upgrader:   NewUpgrader(),
		registry:   registry,
		logger:     logger,
		connConfig: &DefaultConnectionConfig,
	}
}

// SetConnectionConfig overrides the default connection config (useful for testing).
func (h *Handler) SetConnectionConfig(cfg *ConnectionConfig) {
	h.connConfig = cfg
}

// SetMessageHandler sets the message dispatcher for incoming frames from clients.
func (h *Handler) SetMessageHandler(handler MessageHandler) {
	h.messageHandler = handler
}

// RegisterRoutes registers the WebSocket route on the Gin RouterGroup.
func (h *Handler) RegisterRoutes(rg *gin.RouterGroup) {
	rg.GET("/ws", h.HandleUpgrade)
}

// HandleUpgrade handles the handshake, token authentication, and upgrade to WebSocket.
func (h *Handler) HandleUpgrade(c *gin.Context) {
	tokenStr := c.Query("token")

	var userID uuid.UUID
	var authErr error

	if tokenStr == "" {
		authErr = errors.New("missing token query parameter")
	} else {
		claims := &jwt.BaseClaims{}
		if err := jwt.Verify(tokenStr, claims); err != nil {
			authErr = err
		} else {
			parsedID, err := uuid.Parse(claims.UserID)
			if err != nil {
				authErr = err
			} else {
				userID = parsedID
			}
		}
	}

	// Check if auth failed
	if authErr != nil {
		h.logger.Warn("websocket auth failed", zap.Error(authErr))

		// Try to upgrade connection to send custom close code 4001 as specified in Security Design §7
		conn, err := h.upgrader.Upgrade(c.Writer, c.Request, nil)
		if err != nil {
			// Origin check failed or upgrade error (HTTP error status already sent by Gorilla)
			return
		}

		closeMsg := websocket.FormatCloseMessage(CloseCodeUnauthorized, "Unauthorized")
		_ = conn.WriteControl(
			websocket.CloseMessage,
			closeMsg,
			time.Now().Add(time.Second),
		)
		_ = conn.Close()
		return
	}

	// Successful authentication: upgrade connection
	conn, err := h.upgrader.Upgrade(c.Writer, c.Request, nil)
	if err != nil {
		h.logger.Error("websocket upgrade failed", zap.Error(err))
		return
	}

	wsConn := NewConnection(userID, conn, h.registry, h.logger, h.connConfig)
	if h.messageHandler != nil {
		wsConn.SetMessageHandler(h.messageHandler)
	}

	h.registry.Register(wsConn)
	wsConn.Start()

	h.logger.Debug("websocket connection established", zap.String("user_id", userID.String()), zap.String("conn_id", wsConn.ID.String()))
}
