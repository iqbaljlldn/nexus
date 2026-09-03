package http

import (
	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"github.com/iqbaljlldn/nexus/apps/api/internal/platform/middleware"
	"github.com/iqbaljlldn/nexus/apps/api/internal/role/application"
	"github.com/iqbaljlldn/nexus/apps/api/internal/role/interface/dto"
	"github.com/iqbaljlldn/nexus/pkg/contextutil"
	pkgerrors "github.com/iqbaljlldn/nexus/pkg/errors"
	"github.com/iqbaljlldn/nexus/pkg/httpresponse"
)

type RoleHandler struct {
	roleService *application.RoleService
}

func NewRoleHandler(roleService *application.RoleService) *RoleHandler {
	return &RoleHandler{
		roleService: roleService,
	}
}

func (h *RoleHandler) RegisterRoutes(router *gin.RouterGroup) {
	protected := router.Group("")
	protected.Use(middleware.Auth())

	protected.POST("/workspaces/:id/roles", h.Create)
	protected.PATCH("/workspaces/:id/members/:memberId/roles", h.AssignRoles)
}

func (h *RoleHandler) Create(c *gin.Context) {
	_, err := contextutil.UserID(c.Request.Context())
	if err != nil {
		httpresponse.Error(c, &pkgerrors.DomainError{
			Code:    pkgerrors.CodeUserUnauthorized,
			Message: "User tidak terautentikasi.",
			Err:     err,
		})
		return
	}

	workspaceIDStr := c.Param("id")
	workspaceID, err := uuid.Parse(workspaceIDStr)
	if err != nil {
		httpresponse.Error(c, &pkgerrors.DomainError{
			Code:    pkgerrors.CodeInvalidFieldFormat,
			Message: "ID Workspace tidak valid.",
			Err:     err,
		})
		return
	}

	var req dto.CreateRoleRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		httpresponse.Error(c, err)
		return
	}

	// TODO(task-3.5.x): Check permission MANAGE_ROLES using Permission Resolver once implemented.
	// Currently any authenticated member can create a role; this MUST be gated by permission
	// resolver before merging to production.

	role, err := h.roleService.Create(c.Request.Context(), workspaceID, req.Name, req.PermissionBitmask, req.Position)
	if err != nil {
		httpresponse.Error(c, err)
		return
	}

	httpresponse.Created(c, dto.FromRole(role))
}

func (h *RoleHandler) AssignRoles(c *gin.Context) {
	_, err := contextutil.UserID(c.Request.Context())
	if err != nil {
		httpresponse.Error(c, &pkgerrors.DomainError{
			Code:    pkgerrors.CodeUserUnauthorized,
			Message: "User tidak terautentikasi.",
			Err:     err,
		})
		return
	}

	workspaceIDStr := c.Param("id")
	workspaceID, err := uuid.Parse(workspaceIDStr)
	if err != nil {
		httpresponse.Error(c, &pkgerrors.DomainError{
			Code:    pkgerrors.CodeInvalidFieldFormat,
			Message: "ID Workspace tidak valid.",
			Err:     err,
		})
		return
	}

	memberIDStr := c.Param("memberId")
	memberID, err := uuid.Parse(memberIDStr)
	if err != nil {
		httpresponse.Error(c, &pkgerrors.DomainError{
			Code:    pkgerrors.CodeInvalidFieldFormat,
			Message: "ID Member tidak valid.",
			Err:     err,
		})
		return
	}

	var req dto.AssignRolesRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		httpresponse.Error(c, err)
		return
	}

	// TODO(task-3.5.x): Check permission MANAGE_ROLES using Permission Resolver once implemented.
	// Currently any authenticated member can assign roles; this MUST be gated by permission
	// resolver before merging to production.

	userID, _ := contextutil.UserID(c.Request.Context()) // already checked above
	if err := h.roleService.AssignRoles(c.Request.Context(), workspaceID, memberID, userID, req.RoleIDs); err != nil {
		httpresponse.Error(c, err)
		return
	}

	httpresponse.OK(c, gin.H{"message": "Role berhasil diperbarui."})
}
