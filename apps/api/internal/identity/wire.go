package identity

import (
	"os"

	"github.com/google/wire"
	"github.com/iqbaljlldn/nexus/apps/api/internal/identity/application"
	"github.com/iqbaljlldn/nexus/apps/api/internal/identity/domain"
	"github.com/iqbaljlldn/nexus/apps/api/internal/identity/infrastructure"
	identityhttp "github.com/iqbaljlldn/nexus/apps/api/internal/identity/interface/http"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/jackc/pgx/v5/stdlib"
)

func ProvideQuerier(pool *pgxpool.Pool) infrastructure.Querier {
	db := stdlib.OpenDBFromPool(pool)
	return infrastructure.New(db)
}

func ProvideTokenManager() domain.TokenManager {
	secret := os.Getenv("NEXUS_API_JWT_SECRET")
	if secret == "" {
		secret = "default_secret_for_local_dev" // Fallback for local testing
	}
	return infrastructure.NewJWTTokenManager(secret, "nexus-api", "nexus-client")
}

// ProviderSet is the Wire provider set for the identity module.
var ProviderSet = wire.NewSet(
	ProvideQuerier,
	ProvideTokenManager,
	infrastructure.NewPostgresUserRepository,
	infrastructure.NewPostgresSessionRepository,
	application.NewAuthService,
	identityhttp.NewAuthHandler,
)
