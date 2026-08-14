//go:build wireinject
// +build wireinject

package main

import (
	"github.com/gin-gonic/gin"
	"github.com/google/wire"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/redis/go-redis/v9"
	"go.uber.org/zap"

	"github.com/iqbaljlldn/nexus/apps/api/internal/health"
	healthhttp "github.com/iqbaljlldn/nexus/apps/api/internal/health/transport/http"
	"github.com/iqbaljlldn/nexus/apps/api/internal/identity"
	identityhttp "github.com/iqbaljlldn/nexus/apps/api/internal/identity/interface/http"
	"github.com/iqbaljlldn/nexus/pkg/router"
)

func provideRouters(healthRouter *healthhttp.Handler, identityRouter *identityhttp.AuthHandler) []router.ModuleRouter {
	return []router.ModuleRouter{
		healthRouter,
		identityRouter,
	}
}

func InitializeRouter(log *zap.Logger, db *pgxpool.Pool, redisClient *redis.Client) *gin.Engine {
	wire.Build(
		health.ProviderSet,
		identity.ProviderSet,
		provideRouters,
		NewRouter,
	)
	return &gin.Engine{}
}
