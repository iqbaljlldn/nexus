package http

import (
	"net/http"

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

// Healthz godoc
// @Summary Liveness check
// @Description Returns 200 OK if the API server is alive
// @Tags Health
// @Produce json
// @Success 200 {object} application.HealthResponse
// @Router /healthz [get]
func (h *Handler) Healthz(c *gin.Context) {
	resp, err := h.service.Healthz(c.Request.Context())
	if err != nil {
		httpresponse.Error(c, err)
		return
	}
	httpresponse.OK(c, resp)
}

// Readyz godoc
// @Summary Readiness check
// @Description Returns 200 OK if the API and its dependencies (DB, Redis) are ready
// @Tags Health
// @Produce json
// @Success 200 {object} application.HealthResponse
// @Failure 503 {object} httpresponse.ErrorResponse
// @Router /readyz [get]
func (h *Handler) Readyz(c *gin.Context) {
	resp, err := h.service.Readyz(c.Request.Context())
	if err != nil {
		c.JSON(http.StatusServiceUnavailable, httpresponse.ErrorResponse{
			Success: false,
			Error: httpresponse.ErrorDetail{
				Code:    "SERVICE_UNAVAILABLE",
				Message: "Service is not ready.",
				Details: err.Error(),
			},
		})
		return
	}
	httpresponse.OK(c, resp)
}
