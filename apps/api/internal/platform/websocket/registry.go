package websocket

import (
	"context"
	"sync"
	"time"

	"github.com/google/uuid"
	"go.uber.org/zap"
)

// ConnectionRegistry manages active WebSocket connections, channel subscriptions, and broadcasts.
// It implements single-instance connection registry as defined in LLD §2.9.
type ConnectionRegistry struct {
	mu        sync.RWMutex
	byChannel map[uuid.UUID]map[*Connection]struct{}
	byUser    map[uuid.UUID]map[*Connection]struct{}
	conns     map[*Connection]struct{}
	logger    *zap.Logger
}

// NewConnectionRegistry creates a new ConnectionRegistry instance.
func NewConnectionRegistry(logger *zap.Logger) *ConnectionRegistry {
	return &ConnectionRegistry{
		byChannel: make(map[uuid.UUID]map[*Connection]struct{}),
		byUser:    make(map[uuid.UUID]map[*Connection]struct{}),
		conns:     make(map[*Connection]struct{}),
		logger:    logger,
	}
}

// Register registers a new active connection.
func (r *ConnectionRegistry) Register(c *Connection) {
	r.mu.Lock()
	defer r.mu.Unlock()

	r.conns[c] = struct{}{}

	userConns, exists := r.byUser[c.UserID]
	if !exists {
		userConns = make(map[*Connection]struct{})
		r.byUser[c.UserID] = userConns
	}
	userConns[c] = struct{}{}
}

// Unregister removes a connection and cleans up all its channel and user mappings.
func (r *ConnectionRegistry) Unregister(c *Connection) {
	r.mu.Lock()
	defer r.mu.Unlock()

	delete(r.conns, c)

	if userConns, exists := r.byUser[c.UserID]; exists {
		delete(userConns, c)
		if len(userConns) == 0 {
			delete(r.byUser, c.UserID)
		}
	}

	c.mu.Lock()
	channels := make([]uuid.UUID, 0, len(c.channels))
	for chID := range c.channels {
		channels = append(channels, chID)
	}
	c.channels = make(map[uuid.UUID]struct{})
	c.mu.Unlock()

	for _, chID := range channels {
		if chConns, exists := r.byChannel[chID]; exists {
			delete(chConns, c)
			if len(chConns) == 0 {
				delete(r.byChannel, chID)
			}
		}
	}
}

// Subscribe subscribes a connection to a specific channel.
func (r *ConnectionRegistry) Subscribe(c *Connection, channelID uuid.UUID) {
	r.mu.Lock()
	defer r.mu.Unlock()

	// Check if connection is still registered
	if _, exists := r.conns[c]; !exists {
		return
	}

	chConns, exists := r.byChannel[channelID]
	if !exists {
		chConns = make(map[*Connection]struct{})
		r.byChannel[channelID] = chConns
	}
	chConns[c] = struct{}{}

	c.mu.Lock()
	c.channels[channelID] = struct{}{}
	c.mu.Unlock()
}

// Unsubscribe removes a connection from a specific channel.
func (r *ConnectionRegistry) Unsubscribe(c *Connection, channelID uuid.UUID) {
	r.mu.Lock()
	defer r.mu.Unlock()

	if chConns, exists := r.byChannel[channelID]; exists {
		delete(chConns, c)
		if len(chConns) == 0 {
			delete(r.byChannel, channelID)
		}
	}

	c.mu.Lock()
	delete(c.channels, channelID)
	c.mu.Unlock()
}

// Broadcast sends a message to all connections subscribed to the specified channel.
// Conforms exactly to LLD §2.9: snapshots connections under lock, releases lock before I/O,
// and applies non-blocking send with slow-consumer protection.
func (r *ConnectionRegistry) Broadcast(channelID uuid.UUID, msg []byte) {
	r.mu.RLock()
	conns := r.byChannel[channelID]
	snapshot := make([]*Connection, 0, len(conns))
	for c := range conns {
		snapshot = append(snapshot, c)
	}
	r.mu.RUnlock() // Lock is released before I/O so registering/unregistering other connections is not blocked

	for _, c := range snapshot {
		select {
		case c.sendCh <- msg: // non-blocking send to per-connection buffered channel
		default:
			c.logger.Warn("send buffer full, dropping slow consumer connection", zap.String("user_id", c.UserID.String()))
			c.Close() // slow consumer protection — prevents one slow connection from blocking others
		}
	}
}

// BroadcastToUser sends a message to all active connections belonging to a specific user.
func (r *ConnectionRegistry) BroadcastToUser(userID uuid.UUID, msg []byte) {
	r.mu.RLock()
	conns := r.byUser[userID]
	snapshot := make([]*Connection, 0, len(conns))
	for c := range conns {
		snapshot = append(snapshot, c)
	}
	r.mu.RUnlock()

	for _, c := range snapshot {
		select {
		case c.sendCh <- msg:
		default:
			c.logger.Warn("send buffer full, dropping slow consumer connection", zap.String("user_id", c.UserID.String()))
			c.Close()
		}
	}
}

// BroadcastAll sends a message to all active connections.
func (r *ConnectionRegistry) BroadcastAll(msg []byte) {
	r.mu.RLock()
	snapshot := make([]*Connection, 0, len(r.conns))
	for c := range r.conns {
		snapshot = append(snapshot, c)
	}
	r.mu.RUnlock()

	for _, c := range snapshot {
		select {
		case c.sendCh <- msg:
		default:
			c.logger.Warn("send buffer full, dropping slow consumer connection", zap.String("user_id", c.UserID.String()))
			c.Close()
		}
	}
}

// CloseAllGracefully closes all active connections gracefully during server shutdown (LLD §3).
func (r *ConnectionRegistry) CloseAllGracefully(ctx context.Context) {
	r.mu.RLock()
	snapshot := make([]*Connection, 0, len(r.conns))
	for c := range r.conns {
		snapshot = append(snapshot, c)
	}
	r.mu.RUnlock()

	r.logger.Info("closing all websocket connections gracefully", zap.Int("count", len(snapshot)))

	for _, c := range snapshot {
		c.Close()
	}

	// Poll until all connections have finished unregistering or context expires
	ticker := time.NewTicker(20 * time.Millisecond)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			r.logger.Warn("graceful close timeout reached for websocket connections")
			return
		case <-ticker.C:
			r.mu.RLock()
			count := len(r.conns)
			r.mu.RUnlock()
			if count == 0 {
				r.logger.Info("all websocket connections closed successfully")
				return
			}
		}
	}
}

// GetActiveConnectionsCount returns the total number of currently active connections.
func (r *ConnectionRegistry) GetActiveConnectionsCount() int {
	r.mu.RLock()
	defer r.mu.RUnlock()
	return len(r.conns)
}

// GetUserConnectionsCount returns the number of active connections for a given user.
func (r *ConnectionRegistry) GetUserConnectionsCount(userID uuid.UUID) int {
	r.mu.RLock()
	defer r.mu.RUnlock()
	return len(r.byUser[userID])
}

// GetChannelConnectionsCount returns the number of connections subscribed to a channel.
func (r *ConnectionRegistry) GetChannelConnectionsCount(channelID uuid.UUID) int {
	r.mu.RLock()
	defer r.mu.RUnlock()
	return len(r.byChannel[channelID])
}

// IsUserConnected checks if a user has at least one active connection.
func (r *ConnectionRegistry) IsUserConnected(userID uuid.UUID) bool {
	r.mu.RLock()
	defer r.mu.RUnlock()
	return len(r.byUser[userID]) > 0
}
