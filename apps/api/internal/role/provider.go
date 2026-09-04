package role

import (
	"github.com/google/wire"
	"github.com/iqbaljlldn/nexus/apps/api/internal/role/application"
	"github.com/iqbaljlldn/nexus/apps/api/internal/role/infrastructure"
	"github.com/iqbaljlldn/nexus/apps/api/internal/role/interface/http"
)

var ProviderSet = wire.NewSet(
	infrastructure.NewPostgresRoleRepository,
	application.NewRoleService,
	http.NewRoleHandler,
)
