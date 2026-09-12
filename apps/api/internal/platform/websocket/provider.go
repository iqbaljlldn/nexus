package websocket

import "github.com/google/wire"

// ProviderSet is the Wire provider set for WebSocket infrastructure.
var ProviderSet = wire.NewSet(
	NewConnectionRegistry,
	NewHandler,
)
