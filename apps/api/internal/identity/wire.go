package identity

import (
	"database/sql"
	"github.com/google/wire"
	"github.com/iqbaljlldn/nexus/apps/api/internal/identity/application"
	"github.com/iqbaljlldn/nexus/apps/api/internal/identity/domain"
	"github.com/iqbaljlldn/nexus/apps/api/internal/identity/infrastructure"
	identityhttp "github.com/iqbaljlldn/nexus/apps/api/internal/identity/interface/http"
	"github.com/iqbaljlldn/nexus/apps/api/internal/platform/middleware"
	"github.com/iqbaljlldn/nexus/pkg/ratelimit"

	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/jackc/pgx/v5/stdlib"
	"github.com/redis/go-redis/v9"
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

func ProvideRateLimiter(client *redis.Client) *ratelimit.RateLimiter {
	return ratelimit.New(client)
}

func ProvideLoginRateLimiter(limiter *ratelimit.RateLimiter, client *redis.Client) *middleware.LoginRateLimiter {
	return middleware.NewLoginRateLimiter(limiter, client)
}

// ProviderSet is the Wire provider set for the identity module.
var ProviderSet = wire.NewSet(
	ProvideDB,
	ProvideQuerier,
	ProvideTokenManager,
	ProvideRateLimiter,
	ProvideLoginRateLimiter,
	infrastructure.NewPostgresUserRepository,
	infrastructure.NewPostgresSessionRepository,
	application.NewAuthService,
	identityhttp.NewAuthHandler,
)
