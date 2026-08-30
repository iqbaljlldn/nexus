package http

import (
	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"github.com/iqbaljlldn/nexus/apps/api/internal/platform/middleware"
	"github.com/iqbaljlldn/nexus/apps/api/internal/workspace/application"
	"github.com/iqbaljlldn/nexus/apps/api/internal/workspace/interface/dto"
	"github.com/iqbaljlldn/nexus/pkg/httpresponse"
)

type WorkspaceHandler struct {
	workspaceService *application.WorkspaceService
}

func NewWorkspaceHandler(wsService *application.WorkspaceService) *WorkspaceHandler {
	return &WorkspaceHandler{workspaceService: wsService}
}

func (h *WorkspaceHandler) RegisterRoutes(router *gin.RouterGroup) {
	router.Use(middleware.Auth())
	router.POST("/workspaces", h.Create)
	router.GET("/workspaces", h.List)
}

func (h *WorkspaceHandler) Create(c *gin.Context) {
	var req *dto.CreateWorkspaceRequest

	if err := c.ShouldBindJSON(&req); err != nil {
		httpresponse.Error(c, err)
		return
	}

	workspace, err := h.workspaceService.Create(c, c.MustGet("user_id").(uuid.UUID), req.Name, *req.IconURL)
	if err != nil {
		httpresponse.Error(c, err)
		return
	}

	httpresponse.Created(c, workspace)
}

func (h *WorkspaceHandler) List(c *gin.Context) {
	var req *dto.ListWorkspacesRequest

	if err := c.ShouldBindJSON(&req); err != nil {
		httpresponse.Error(c, err)
		return
	}

	res, meta, err := h.workspaceService.ListByUserID(c, c.MustGet("user_id").(uuid.UUID), req)
	if err != nil {
		httpresponse.Error(c, err)
		return
	}

	httpresponse.SuccessWithMeta(c, 200, res, meta)
}
