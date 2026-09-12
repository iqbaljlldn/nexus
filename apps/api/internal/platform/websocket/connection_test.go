package websocket_test

import (
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/gorilla/websocket"
	ws "github.com/iqbaljlldn/nexus/apps/api/internal/platform/websocket"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.uber.org/zap"
)

func TestConnection_SingleWriter_ConcurrentBroadcasts(t *testing.T) {
	logger := zap.NewNop()
	server, registry, handler := setupTestWSServer(t, logger)

	// Set a generous buffer so concurrency test tests single-writer without dropping
	cfg := &ws.ConnectionConfig{
		WriteWait:      2 * time.Second,
		PongWait:       5 * time.Second,
		PingPeriod:     4 * time.Second,
		MaxMessageSize: 8192,
		SendBufferSize: 1024,
	}
	handler.SetConnectionConfig(cfg)

	userID := uuid.New()
	token := createValidToken(t, userID)

	wsURL := "ws" + strings.TrimPrefix(server.URL, "http") + "/api/v1/ws?token=" + token
	clientConn, _, err := websocket.DefaultDialer.Dial(wsURL, nil)
	require.NoError(t, err)
	defer func() {
		_ = clientConn.Close()
	}()

	// Wait for connection to be registered
	assert.Eventually(t, func() bool {
		return registry.GetUserConnectionsCount(userID) == 1
	}, 1*time.Second, 10*time.Millisecond)

	const goroutines = 10
	const messagesPerGoroutine = 50

	var wg sync.WaitGroup
	receivedCount := 0
	var countMu sync.Mutex
	doneReading := make(chan struct{})

	// Client reader goroutine
	go func() {
		defer close(doneReading)
		for {
			_, _, readErr := clientConn.ReadMessage()
			if readErr != nil {
				return
			}
			countMu.Lock()
			receivedCount++
			countMu.Unlock()
		}
	}()

	// Spawn multiple concurrent goroutines broadcasting to the same user
	wg.Add(goroutines)
	for i := 0; i < goroutines; i++ {
		go func(gID int) {
			defer wg.Done()
			for m := 0; m < messagesPerGoroutine; m++ {
				registry.BroadcastToUser(userID, []byte(`{"event":"test","seq":1}`))
				time.Sleep(1 * time.Millisecond)
			}
		}(i)
	}

	wg.Wait()

	// Verify all messages received without race detector violations or data corruption
	assert.Eventually(t, func() bool {
		countMu.Lock()
		defer countMu.Unlock()
		return receivedCount == goroutines*messagesPerGoroutine
	}, 5*time.Second, 50*time.Millisecond)
}

func TestConnection_SlowConsumer_Dropped(t *testing.T) {
	logger := zap.NewNop()
	registry := ws.NewConnectionRegistry(logger)

	cfg := &ws.ConnectionConfig{
		SendBufferSize: 2, // Tiny buffer of 2 items
	}

	slowUserID := uuid.New()
	fastUserID := uuid.New()
	channelID := uuid.New()

	slowConn := ws.NewConnection(slowUserID, nil, registry, logger, cfg)
	fastConn := ws.NewConnection(fastUserID, nil, registry, logger, cfg)

	registry.Register(slowConn)
	registry.Register(fastConn)
	registry.Subscribe(slowConn, channelID)
	registry.Subscribe(fastConn, channelID)

	require.Equal(t, 2, registry.GetActiveConnectionsCount())
	require.Equal(t, 2, registry.GetChannelConnectionsCount(channelID))

	// Fill slowConn's send buffer completely
	require.True(t, slowConn.Send([]byte("msg1")))
	require.True(t, slowConn.Send([]byte("msg2")))
	require.False(t, slowConn.Send([]byte("overflow_attempt")), "Send should return false when buffer is full")

	// Now broadcast to channelID. fastConn will receive msg3, slowConn will hit default in select and be dropped
	registry.Broadcast(channelID, []byte("msg3"))

	// Verify slowConn was dropped and removed from registry
	assert.False(t, registry.IsUserConnected(slowUserID))
	assert.True(t, slowConn.IsClosed())
	assert.Equal(t, 1, registry.GetActiveConnectionsCount())
	assert.Equal(t, 1, registry.GetChannelConnectionsCount(channelID))

	// Fast conn is still connected and in registry
	assert.True(t, registry.IsUserConnected(fastUserID))
	assert.False(t, fastConn.IsClosed())
}
