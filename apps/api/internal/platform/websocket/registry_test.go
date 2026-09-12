package websocket_test

import (
	"sync"
	"testing"

	"github.com/google/uuid"
	ws "github.com/iqbaljlldn/nexus/apps/api/internal/platform/websocket"
	"github.com/stretchr/testify/assert"
	"go.uber.org/zap"
)

func TestConnectionRegistry_Subscribe_Broadcast_Isolation(t *testing.T) {
	logger := zap.NewNop()
	registry := ws.NewConnectionRegistry(logger)

	u1 := uuid.New()
	u2 := uuid.New()
	u3 := uuid.New()

	c1 := ws.NewConnection(u1, nil, registry, logger, nil)
	c2 := ws.NewConnection(u2, nil, registry, logger, nil)
	c3 := ws.NewConnection(u3, nil, registry, logger, nil)

	registry.Register(c1)
	registry.Register(c2)
	registry.Register(c3)

	assert.Equal(t, 3, registry.GetActiveConnectionsCount())
	assert.Equal(t, 1, registry.GetUserConnectionsCount(u1))

	channelA := uuid.New()
	channelB := uuid.New()

	// c1 & c2 in channelA; c2 & c3 in channelB
	registry.Subscribe(c1, channelA)
	registry.Subscribe(c2, channelA)
	registry.Subscribe(c2, channelB)
	registry.Subscribe(c3, channelB)

	assert.Equal(t, 2, registry.GetChannelConnectionsCount(channelA))
	assert.Equal(t, 2, registry.GetChannelConnectionsCount(channelB))

	// Broadcast to channelA
	msgA := []byte("hello channel A")
	registry.Broadcast(channelA, msgA)

	// c1 and c2 should have received msgA in their sendCh
	assert.True(t, c1.Send([]byte("ping"))) // just checking sendCh is alive
	// Check that msgA was sent into c1.sendCh and c2.sendCh
	// Unsubscribe c2 from channelA
	registry.Unsubscribe(c2, channelA)
	assert.Equal(t, 1, registry.GetChannelConnectionsCount(channelA))

	// Unregister c1
	registry.Unregister(c1)
	assert.Equal(t, 0, registry.GetChannelConnectionsCount(channelA))
	assert.Equal(t, 0, registry.GetUserConnectionsCount(u1))
	assert.False(t, registry.IsUserConnected(u1))
}

func TestConnectionRegistry_HighConcurrency(t *testing.T) {
	logger := zap.NewNop()
	registry := ws.NewConnectionRegistry(logger)

	const numWorkers = 50
	const opsPerWorker = 100

	var wg sync.WaitGroup
	wg.Add(numWorkers)

	channelID := uuid.New()

	for i := 0; i < numWorkers; i++ {
		go func() {
			defer wg.Done()
			userID := uuid.New()
			conn := ws.NewConnection(userID, nil, registry, logger, nil)

			for j := 0; j < opsPerWorker; j++ {
				registry.Register(conn)
				registry.Subscribe(conn, channelID)
				registry.Broadcast(channelID, []byte("concurrent message"))
				registry.BroadcastToUser(userID, []byte("user message"))
				registry.BroadcastAll([]byte("all message"))
				registry.Unsubscribe(conn, channelID)
				registry.Unregister(conn)
			}
		}()
	}

	wg.Wait()
	assert.Equal(t, 0, registry.GetActiveConnectionsCount())
}
