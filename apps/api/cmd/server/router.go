package main

import (
	"github.com/iqbaljlldn/nexus/pkg/router"

	"github.com/gin-gonic/gin"
	"go.uber.org/zap"

	_ "github.com/iqbaljlldn/nexus/apps/api/docs"
	swaggerFiles "github.com/swaggo/files"
	ginSwagger "github.com/swaggo/gin-swagger"
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

	r.GET("/swagger/*any", ginSwagger.WrapHandler(swaggerFiles.Handler))

	return r
}
