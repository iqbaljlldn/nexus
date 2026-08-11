package identity

import (
	"github.com/google/wire"
	"github.com/iqbaljlldn/nexus/apps/api/internal/identity/application"
	"github.com/iqbaljlldn/nexus/apps/api/internal/identity/infrastructure"
	identityhttp "github.com/iqbaljlldn/nexus/apps/api/internal/identity/interface/http"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/jackc/pgx/v5/stdlib"
)

func ProvideQuerier(pool *pgxpool.Pool) infrastructure.Querier {
	db := stdlib.OpenDBFromPool(pool)
	return infrastructure.New(db)
}

// ProviderSet is the Wire provider set for the identity module.
var ProviderSet = wire.NewSet(
	ProvideQuerier,
	infrastructure.NewPostgresUserRepository,
	application.NewAuthService,
	identityhttp.NewRegisterHandler,
)
