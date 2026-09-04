package http

import (
	"errors"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"github.com/iqbaljlldn/nexus/apps/api/internal/platform/middleware"
	"github.com/iqbaljlldn/nexus/apps/api/internal/role/application"
	roleDomain "github.com/iqbaljlldn/nexus/apps/api/internal/role/domain"
	"github.com/iqbaljlldn/nexus/apps/api/internal/role/interface/dto"
	"github.com/iqbaljlldn/nexus/pkg/contextutil"
	pkgerrors "github.com/iqbaljlldn/nexus/pkg/errors"
	"github.com/iqbaljlldn/nexus/pkg/httpresponse"
)

type RoleHandler struct {
	roleService  *application.RoleService
	permResolver PermissionResolver
}

func NewRoleHandler(roleService *application.RoleService, permResolver PermissionResolver) *RoleHandler {
	return &RoleHandler{
		roleService:  roleService,
		permResolver: permResolver,
	}
}

func (h *RoleHandler) RegisterRoutes(router *gin.RouterGroup) {
	protected := router.Group("")
	protected.Use(middleware.Auth())

	protected.POST("/workspaces/:id/roles", h.Create)
	protected.PATCH("/workspaces/:id/members/:memberId/roles", h.AssignRoles)
}

func (h *RoleHandler) Create(c *gin.Context) {
	userID, err := contextutil.UserID(c.Request.Context())
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

	allowed, err := h.permResolver.Resolve(c.Request.Context(), userID, workspaceID, uuid.Nil, roleDomain.PermManageRoles)
	if err != nil {
		httpresponse.Error(c, err)
		return
	}
	if !allowed {
		httpresponse.Error(c, &pkgerrors.DomainError{
			Code:    pkgerrors.CodeForbidden,
			Message: "Anda tidak memiliki izin untuk mengelola role di workspace ini.",
			Err:     errors.New("forbidden: missing MANAGE_ROLES permission"),
		})
		return
	}

	role, err := h.roleService.Create(c.Request.Context(), workspaceID, req.Name, req.PermissionBitmask, req.Position)
	if err != nil {
		httpresponse.Error(c, err)
		return
	}

	httpresponse.Created(c, dto.FromRole(role))
}

func (h *RoleHandler) AssignRoles(c *gin.Context) {
	userID, err := contextutil.UserID(c.Request.Context())
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

	allowed, err := h.permResolver.Resolve(c.Request.Context(), userID, workspaceID, uuid.Nil, roleDomain.PermManageRoles)
	if err != nil {
		httpresponse.Error(c, err)
		return
	}
	if !allowed {
		httpresponse.Error(c, &pkgerrors.DomainError{
			Code:    pkgerrors.CodeForbidden,
			Message: "Anda tidak memiliki izin untuk mengubah role di workspace ini.",
			Err:     errors.New("forbidden: missing MANAGE_ROLES permission"),
		})
		return
	}

	if err := h.roleService.AssignRoles(c.Request.Context(), workspaceID, memberID, userID, req.RoleIDs); err != nil {
		httpresponse.Error(c, err)
		return
	}

	httpresponse.OK(c, gin.H{"message": "Role berhasil diperbarui."})
}
