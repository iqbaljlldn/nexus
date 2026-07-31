package http

import (
	"nexus-be/internal/health/application"
	"nexus-be/pkg/httpresponse"

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
		httpresponse.Error(c, err)
		return
	}
	httpresponse.OK(c, resp)
}

