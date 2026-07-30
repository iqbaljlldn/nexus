package router

import "github.com/gin-gonic/gin"

type ModuleRouter interface {
	RegisterRoutes(router *gin.RouterGroup)
}
