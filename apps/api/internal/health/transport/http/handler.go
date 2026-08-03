package http

import (
	"github.com/iqbaljlldn/nexus/apps/api/internal/health/application"
	"github.com/iqbaljlldn/nexus/pkg/httpresponse"

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
