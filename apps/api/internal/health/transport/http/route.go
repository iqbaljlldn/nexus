package http

import "github.com/gin-gonic/gin"

func (h *Handler) RegisterRoutes(router *gin.RouterGroup) {
	router.GET("/health", h.Health)
}
