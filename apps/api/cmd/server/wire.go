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
	"github.com/iqbaljlldn/nexus/pkg/router"
)

func provideRouters(healthRouter router.ModuleRouter) []router.ModuleRouter {
	return []router.ModuleRouter{
		healthRouter,
	}
}

func InitializeRouter(log *zap.Logger, db *pgxpool.Pool, redisClient *redis.Client) *gin.Engine {
	wire.Build(
		health.ProviderSet,
		provideRouters,
		NewRouter,
	)
	return &gin.Engine{}
}
