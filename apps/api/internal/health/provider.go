package health

import (
	"github.com/iqbaljlldn/nexus/apps/api/internal/health/application"
	httpHandler "github.com/iqbaljlldn/nexus/apps/api/internal/health/transport/http"

	"github.com/google/wire"
)

var ProviderSet = wire.NewSet(
	application.NewService,
	httpHandler.NewHandler,
)
