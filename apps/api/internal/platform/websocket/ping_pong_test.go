package websocket_test

import (
	"context"
	"strings"
	"sync/atomic"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/gorilla/websocket"
	ws "github.com/iqbaljlldn/nexus/apps/api/internal/platform/websocket"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.uber.org/zap"
)

func TestConnection_PingPong_Heartbeat(t *testing.T) {
	logger := zap.NewNop()
	server, registry, handler := setupTestWSServer(t, logger)

	// Short ping/pong interval for quick testing
	cfg := &ws.ConnectionConfig{
		WriteWait:      500 * time.Millisecond,
		PongWait:       1000 * time.Millisecond,
		PingPeriod:     300 * time.Millisecond,
		MaxMessageSize: 8192,
		SendBufferSize: 64,
	}
	handler.SetConnectionConfig(cfg)

	userID := uuid.New()
	token := createValidToken(t, userID)
	wsURL := "ws" + strings.TrimPrefix(server.URL, "http") + "/api/v1/ws?token=" + token

	dialer := websocket.DefaultDialer
	clientConn, _, err := dialer.Dial(wsURL, nil)
	require.NoError(t, err)
	defer func() {
		_ = clientConn.Close()
	}()

	var pingCount int32
	clientConn.SetPingHandler(func(appData string) error {
		atomic.AddInt32(&pingCount, 1)
		// Send Pong in response to Ping
		return clientConn.WriteControl(websocket.PongMessage, []byte(appData), time.Now().Add(500*time.Millisecond))
	})

	// Start client read loop to process control messages
	go func() {
		for {
			if _, _, err := clientConn.ReadMessage(); err != nil {
				return
			}
		}
	}()

	// Wait for multiple pings to be received and answered
	assert.Eventually(t, func() bool {
		return atomic.LoadInt32(&pingCount) >= 2
	}, 2*time.Second, 50*time.Millisecond)

	// Connection should still be active in registry because pongs were returned
	assert.True(t, registry.IsUserConnected(userID))
	assert.Equal(t, 1, registry.GetActiveConnectionsCount())
}

func TestConnection_PongTimeout_AutoCleanedUp(t *testing.T) {
	logger := zap.NewNop()
	server, registry, handler := setupTestWSServer(t, logger)

	// Fast timeout: pong wait 300ms, ping period 100ms
	cfg := &ws.ConnectionConfig{
		WriteWait:      200 * time.Millisecond,
		PongWait:       300 * time.Millisecond,
		PingPeriod:     100 * time.Millisecond,
		MaxMessageSize: 8192,
		SendBufferSize: 64,
	}
	handler.SetConnectionConfig(cfg)

	userID := uuid.New()
	token := createValidToken(t, userID)
	wsURL := "ws" + strings.TrimPrefix(server.URL, "http") + "/api/v1/ws?token=" + token

	dialer := websocket.DefaultDialer
	clientConn, _, err := dialer.Dial(wsURL, nil)
	require.NoError(t, err)
	defer func() {
		_ = clientConn.Close()
	}()

	// Intercept ping and deliberately DO NOT send pong back
	clientConn.SetPingHandler(func(string) error {
		return nil // Ignore ping, don't write pong
	})

	// Read loop so client receives ping frames
	go func() {
		for {
			if _, _, err := clientConn.ReadMessage(); err != nil {
				return
			}
		}
	}()

	// Connection should be automatically terminated by the server due to pong timeout
	assert.Eventually(t, func() bool {
		return !registry.IsUserConnected(userID) && registry.GetActiveConnectionsCount() == 0
	}, 2*time.Second, 50*time.Millisecond)
}

func TestConnectionRegistry_CloseAllGracefully(t *testing.T) {
	logger := zap.NewNop()
	server, registry, _ := setupTestWSServer(t, logger)

	u1 := uuid.New()
	u2 := uuid.New()
	t1 := createValidToken(t, u1)
	t2 := createValidToken(t, u2)

	wsBase := "ws" + strings.TrimPrefix(server.URL, "http") + "/api/v1/ws?token="

	c1, _, err := websocket.DefaultDialer.Dial(wsBase+t1, nil)
	require.NoError(t, err)
	defer func() {
		_ = c1.Close()
	}()

	c2, _, err := websocket.DefaultDialer.Dial(wsBase+t2, nil)
	require.NoError(t, err)
	defer func() {
		_ = c2.Close()
	}()

	assert.Eventually(t, func() bool {
		return registry.GetActiveConnectionsCount() == 2
	}, 1*time.Second, 10*time.Millisecond)

	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer cancel()

	registry.CloseAllGracefully(ctx)

	assert.Equal(t, 0, registry.GetActiveConnectionsCount())
}
