package main

import (
	"nexus-be/pkg/router"

	"github.com/gin-gonic/gin"
	"go.uber.org/zap"
)

func NewRouter(
	log *zap.Logger,
	routers []router.ModuleRouter,

) *gin.Engine {
	r := gin.New()

	api := r.Group("/api/v1")

	for _, router := range routers {
		router.RegisterRoutes(api)
	}

	return r
}
