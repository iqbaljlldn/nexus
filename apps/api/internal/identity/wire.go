package identity

import (
	"database/sql"
	"github.com/google/wire"
	"github.com/iqbaljlldn/nexus/apps/api/internal/identity/application"
	"github.com/iqbaljlldn/nexus/apps/api/internal/identity/domain"
	"github.com/iqbaljlldn/nexus/apps/api/internal/identity/infrastructure"
	identityhttp "github.com/iqbaljlldn/nexus/apps/api/internal/identity/interface/http"

	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/jackc/pgx/v5/stdlib"
)

func ProvideDB(pool *pgxpool.Pool) *sql.DB {
	return stdlib.OpenDBFromPool(pool)
}

func ProvideQuerier(db *sql.DB) infrastructure.Querier {
	return infrastructure.New(db)
}

func ProvideTokenManager() domain.TokenManager {
	return infrastructure.NewJWTTokenManager("nexus-api", "nexus-client")
}

// ProviderSet is the Wire provider set for the identity module.
var ProviderSet = wire.NewSet(
	ProvideDB,
	ProvideQuerier,
	ProvideTokenManager,
	infrastructure.NewPostgresUserRepository,
	infrastructure.NewPostgresSessionRepository,
	application.NewAuthService,
	identityhttp.NewAuthHandler,
)
