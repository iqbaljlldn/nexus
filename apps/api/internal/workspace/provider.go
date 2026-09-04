package workspace

import (
	"github.com/google/wire"
	"github.com/iqbaljlldn/nexus/apps/api/internal/workspace/application"
	"github.com/iqbaljlldn/nexus/apps/api/internal/workspace/infrastructure"
	"github.com/iqbaljlldn/nexus/apps/api/internal/workspace/interface/http"
)

var ProviderSet = wire.NewSet(
	infrastructure.NewPostgresWorkspaceRepository,
	infrastructure.NewPostgresInviteRepository,
	infrastructure.NewPostgresTransactionManager,
	infrastructure.NewPostgresChannelOverrideRepository,
	wire.Bind(new(application.TransactionManager), new(*infrastructure.PostgresTransactionManager)),
	application.NewWorkspaceService,
	application.NewInviteService,
	application.NewPermissionResolver,
	application.NewCachedPermissionResolver,
	http.NewWorkspaceHandler,
)
