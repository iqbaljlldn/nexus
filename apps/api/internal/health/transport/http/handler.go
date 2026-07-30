package http

import (
	"net/http"
	"nexus-be/internal/health/application"

	"github.com/gin-gonic/gin"
)

type Handler struct {
	service application.Service
}

func NewHandler(s application.Service) *Handler {
	return &Handler{
		service: s,
	}
}

func (h *Handler) Health(c *gin.Context) {
	resp, err := h.service.Check(c.Request.Context())
	if err != nil {
		c.JSON(
			http.StatusInternalServerError,
			gin.H{
				"message": err.Error(),
			},
		)
		return
	}
	c.JSON(http.StatusOK, resp)
}
