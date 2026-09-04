package channel

import (
	"database/sql"
	"github.com/google/wire"
	"github.com/iqbaljlldn/nexus/apps/api/internal/channel/application"
	"github.com/iqbaljlldn/nexus/apps/api/internal/channel/domain"
	"github.com/iqbaljlldn/nexus/apps/api/internal/channel/infrastructure"
	"github.com/iqbaljlldn/nexus/apps/api/internal/channel/interface/http"
)

func ProvideDB(db *sql.DB) infrastructure.DBTX {
	return db
}

var ProviderSet = wire.NewSet(
	ProvideDB,
	infrastructure.NewPostgresChannelRepository,
	wire.Bind(new(domain.ChannelRepository), new(*infrastructure.PostgresChannelRepository)),
	application.NewChannelService,
	http.NewChannelHandler,
)
