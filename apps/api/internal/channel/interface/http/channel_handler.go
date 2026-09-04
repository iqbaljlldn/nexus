package http

import (
	"errors"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"github.com/iqbaljlldn/nexus/apps/api/internal/channel/application"
	"github.com/iqbaljlldn/nexus/apps/api/internal/channel/interface/dto"
	"github.com/iqbaljlldn/nexus/apps/api/internal/platform/middleware"
	roleDomain "github.com/iqbaljlldn/nexus/apps/api/internal/role/domain"
	"github.com/iqbaljlldn/nexus/pkg/contextutil"
	pkgerrors "github.com/iqbaljlldn/nexus/pkg/errors"
	"github.com/iqbaljlldn/nexus/pkg/httpresponse"
)

type ChannelHandler struct {
	channelService *application.ChannelService
	permResolver   PermissionResolver
}

func NewChannelHandler(channelService *application.ChannelService, permResolver PermissionResolver) *ChannelHandler {
	return &ChannelHandler{
		channelService: channelService,
		permResolver:   permResolver,
	}
}

func (h *ChannelHandler) RegisterRoutes(router *gin.RouterGroup) {
	protected := router.Group("")
	protected.Use(middleware.Auth())

	protected.POST("/workspaces/:id/channels", h.CreateTextChannel)
	protected.GET("/workspaces/:id/channels", h.ListByWorkspace)
	protected.PATCH("/channels/:id/permission-overrides", h.PatchPermissionOverrides)
}

func (h *ChannelHandler) CreateTextChannel(c *gin.Context) {
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

	var req dto.CreateChannelRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		httpresponse.Error(c, err)
		return
	}

	allowed, err := h.permResolver.Resolve(c.Request.Context(), userID, workspaceID, uuid.Nil, roleDomain.PermManageChannels)
	if err != nil {
		httpresponse.Error(c, err)
		return
	}
	if !allowed {
		httpresponse.Error(c, &pkgerrors.DomainError{
			Code:    pkgerrors.CodeForbidden,
			Message: "Anda tidak memiliki izin untuk mengelola channel di workspace ini.",
			Err:     errors.New("forbidden: missing MANAGE_CHANNELS permission"),
		})
		return
	}

	ch, err := h.channelService.CreateTextChannel(c.Request.Context(), workspaceID, req.Name, req.CategoryID)
	if err != nil {
		httpresponse.Error(c, err)
		return
	}

	httpresponse.Created(c, dto.FromChannel(ch))
}

func (h *ChannelHandler) PatchPermissionOverrides(c *gin.Context) {
	userID, err := contextutil.UserID(c.Request.Context())
	if err != nil {
		httpresponse.Error(c, &pkgerrors.DomainError{
			Code:    pkgerrors.CodeUserUnauthorized,
			Message: "User tidak terautentikasi.",
			Err:     err,
		})
		return
	}

	channelIDStr := c.Param("id")
	channelID, err := uuid.Parse(channelIDStr)
	if err != nil {
		httpresponse.Error(c, &pkgerrors.DomainError{
			Code:    pkgerrors.CodeInvalidFieldFormat,
			Message: "ID Channel tidak valid.",
			Err:     err,
		})
		return
	}

	var req dto.PatchPermissionOverrideRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		httpresponse.Error(c, err)
		return
	}

	if (req.RoleID == nil && req.MemberID == nil) || (req.RoleID != nil && req.MemberID != nil) {
		httpresponse.Error(c, &pkgerrors.DomainError{
			Code:    pkgerrors.CodeInvalidFieldFormat,
			Message: "Harus mengisi salah satu dari role_id atau member_id, tidak boleh keduanya.",
			Err:     errors.New("xor constraint failed in handler"),
		})
		return
	}

	ch, err := h.channelService.GetByID(c.Request.Context(), channelID)
	if err != nil {
		httpresponse.Error(c, err)
		return
	}
	if ch.WorkspaceID == nil {
		httpresponse.Error(c, &pkgerrors.DomainError{
			Code:    pkgerrors.CodeInvalidFieldFormat,
			Message: "Tidak bisa set permission override untuk channel DM.",
			Err:     errors.New("cannot override permission on DM channel"),
		})
		return
	}

	// Validate MANAGE_ROLES for this channel/workspace
	allowed, err := h.permResolver.Resolve(c.Request.Context(), userID, *ch.WorkspaceID, channelID, roleDomain.PermManageRoles)
	if err != nil {
		httpresponse.Error(c, err)
		return
	}
	if !allowed {
		httpresponse.Error(c, &pkgerrors.DomainError{
			Code:    pkgerrors.CodeForbidden,
			Message: "Anda tidak memiliki izin untuk mengelola peran di channel ini.",
			Err:     errors.New("forbidden: missing MANAGE_ROLES permission"),
		})
		return
	}

	if err := h.channelService.PatchPermissionOverride(c.Request.Context(), channelID, req.RoleID, req.MemberID, req.AllowBitmask, req.DenyBitmask); err != nil {
		httpresponse.Error(c, err)
		return
	}

	httpresponse.OK(c, gin.H{"message": "Permission override berhasil disimpan."})
}

func (h *ChannelHandler) ListByWorkspace(c *gin.Context) {
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

	channels, err := h.channelService.ListByWorkspaceID(c.Request.Context(), workspaceID)
	if err != nil {
		httpresponse.Error(c, err)
		return
	}

	dtos := make([]*dto.ChannelResponse, 0, len(channels))
	for i := range channels {
		dtos = append(dtos, dto.FromChannel(&channels[i]))
	}

	httpresponse.OK(c, dtos)
}
