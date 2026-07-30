package health

import (
	"nexus-be/internal/health/application"
	httpHandler "nexus-be/internal/health/transport/http"
	"nexus-be/pkg/router"

	"github.com/google/wire"
)

var ProviderSet = wire.NewSet(
	application.NewService,
	httpHandler.NewHandler,
	wire.Bind(new(router.ModuleRouter), new(*httpHandler.Handler)),
)
