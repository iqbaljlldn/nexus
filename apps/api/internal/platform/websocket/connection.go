package websocket

import (
	"sync"
	"time"

	"github.com/google/uuid"
	"github.com/gorilla/websocket"
	"go.uber.org/zap"
)

// ConnectionConfig holds timing and buffer configurations for a WebSocket connection.
type ConnectionConfig struct {
	WriteWait      time.Duration
	PongWait       time.Duration
	PingPeriod     time.Duration
	MaxMessageSize int64
	SendBufferSize int
}

// DefaultConnectionConfig defines standard production defaults.
var DefaultConnectionConfig = ConnectionConfig{
	WriteWait:      10 * time.Second,
	PongWait:       60 * time.Second,
	PingPeriod:     54 * time.Second,
	MaxMessageSize: 8192, // 8 KB frame size limit (Security Design §7)
	SendBufferSize: 64,   // Initial buffer size (LLD §2.9)
}

// MessageHandler is a callback function for incoming WebSocket messages from the client.
type MessageHandler func(conn *Connection, message []byte)

// Connection represents a single authenticated WebSocket connection.
// It enforces the single-writer principle: only writePump writes to the underlying websocket.Conn.
type Connection struct {
	ID       uuid.UUID
	UserID   uuid.UUID
	conn     *websocket.Conn
	sendCh   chan []byte
	registry *ConnectionRegistry
	logger   *zap.Logger
	cfg      ConnectionConfig

	mu        sync.Mutex
	channels  map[uuid.UUID]struct{}
	onMessage MessageHandler
	done      chan struct{}
	closeOnce sync.Once
	closed    bool
}

// NewConnection creates a new Connection instance.
func NewConnection(
	userID uuid.UUID,
	conn *websocket.Conn,
	registry *ConnectionRegistry,
	logger *zap.Logger,
	cfg *ConnectionConfig,
) *Connection {
	resolvedCfg := DefaultConnectionConfig
	if cfg != nil {
		if cfg.WriteWait > 0 {
			resolvedCfg.WriteWait = cfg.WriteWait
		}
		if cfg.PongWait > 0 {
			resolvedCfg.PongWait = cfg.PongWait
		}
		if cfg.PingPeriod > 0 {
			resolvedCfg.PingPeriod = cfg.PingPeriod
		}
		if cfg.MaxMessageSize > 0 {
			resolvedCfg.MaxMessageSize = cfg.MaxMessageSize
		}
		if cfg.SendBufferSize > 0 {
			resolvedCfg.SendBufferSize = cfg.SendBufferSize
		}
	}

	return &Connection{
		ID:       uuid.New(),
		UserID:   userID,
		conn:     conn,
		sendCh:   make(chan []byte, resolvedCfg.SendBufferSize),
		registry: registry,
		logger:   logger.With(zap.String("user_id", userID.String())),
		cfg:      resolvedCfg,
		channels: make(map[uuid.UUID]struct{}),
		done:     make(chan struct{}),
	}
}

// SetMessageHandler sets the callback for incoming client messages.
func (c *Connection) SetMessageHandler(handler MessageHandler) {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.onMessage = handler
}

// Start begins the reader and writer loops in separate goroutines.
func (c *Connection) Start() {
	if c.conn != nil {
		go c.writePump()
		go c.readPump()
	}
}

// readPump pumps messages from the websocket connection to the application.
// It also manages the read deadline and pong handling to detect dead connections.
func (c *Connection) readPump() {
	defer func() {
		c.Close()
	}()

	if c.conn == nil {
		return
	}

	c.conn.SetReadLimit(c.cfg.MaxMessageSize)
	_ = c.conn.SetReadDeadline(time.Now().Add(c.cfg.PongWait))
	c.conn.SetPongHandler(func(string) error {
		_ = c.conn.SetReadDeadline(time.Now().Add(c.cfg.PongWait))
		return nil
	})

	for {
		_, message, err := c.conn.ReadMessage()
		if err != nil {
			if websocket.IsUnexpectedCloseError(
				err,
				websocket.CloseGoingAway,
				websocket.CloseNormalClosure,
				websocket.CloseNoStatusReceived,
				CloseCodeUnauthorized,
			) {
				c.logger.Debug("websocket connection read error", zap.Error(err))
			}
			break
		}

		c.mu.Lock()
		handler := c.onMessage
		c.mu.Unlock()

		if handler != nil {
			handler(c, message)
		}
	}
}

// writePump pumps messages from the sendCh to the websocket connection.
// It also sends periodic ping frames to the client.
func (c *Connection) writePump() {
	ticker := time.NewTicker(c.cfg.PingPeriod)
	defer func() {
		ticker.Stop()
		c.Close()
	}()

	for {
		select {
		case msg, ok := <-c.sendCh:
			if c.conn == nil {
				return
			}
			_ = c.conn.SetWriteDeadline(time.Now().Add(c.cfg.WriteWait))
			if !ok {
				// sendCh was closed, send normal close frame
				_ = c.conn.WriteMessage(websocket.CloseMessage, []byte{})
				return
			}

			w, err := c.conn.NextWriter(websocket.TextMessage)
			if err != nil {
				return
			}
			if _, err := w.Write(msg); err != nil {
				return
			}
			if err := w.Close(); err != nil {
				return
			}

		case <-ticker.C:
			if c.conn == nil {
				return
			}
			_ = c.conn.SetWriteDeadline(time.Now().Add(c.cfg.WriteWait))
			if err := c.conn.WriteMessage(websocket.PingMessage, nil); err != nil {
				return
			}

		case <-c.done:
			return
		}
	}
}

// Close closes the connection, unregisters from registry, and releases resources cleanly.
func (c *Connection) Close() {
	c.closeOnce.Do(func() {
		c.mu.Lock()
		c.closed = true
		c.mu.Unlock()

		close(c.done)

		if c.registry != nil {
			c.registry.Unregister(c)
		}

		if c.conn != nil {
			_ = c.conn.Close()
		}
	})
}

// IsClosed returns whether the connection has been closed.
func (c *Connection) IsClosed() bool {
	c.mu.Lock()
	defer c.mu.Unlock()
	return c.closed
}

// Send attempts to enqueue a message to the connection's send buffer.
// Returns false if the buffer is full (slow consumer) or the connection is closed.
func (c *Connection) Send(msg []byte) bool {
	select {
	case <-c.done:
		return false
	default:
	}

	select {
	case c.sendCh <- msg:
		return true
	default:
		return false
	}
}
