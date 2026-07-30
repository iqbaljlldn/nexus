//go:build wireinject
// +build wireinject

package main

import (
	"github.com/gin-gonic/gin"
	"github.com/google/wire"
	"go.uber.org/zap"

	"nexus-be/internal/health"
	"nexus-be/pkg/router"
)

func provideRouters(healthRouter router.ModuleRouter) []router.ModuleRouter {
	return []router.ModuleRouter{
		healthRouter,
	}
}

func InitializeRouter(log *zap.Logger) *gin.Engine {
	wire.Build(
		health.ProviderSet,
		provideRouters,
		NewRouter,
	)
	return &gin.Engine{}
}
