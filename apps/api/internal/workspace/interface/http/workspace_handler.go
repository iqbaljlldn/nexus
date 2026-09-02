package http

import (
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"github.com/iqbaljlldn/nexus/apps/api/internal/platform/middleware"
	"github.com/iqbaljlldn/nexus/apps/api/internal/workspace/application"
	"github.com/iqbaljlldn/nexus/apps/api/internal/workspace/interface/dto"
	"github.com/iqbaljlldn/nexus/pkg/contextutil"
	pkgerrors "github.com/iqbaljlldn/nexus/pkg/errors"
	"github.com/iqbaljlldn/nexus/pkg/httpresponse"
)

type WorkspaceHandler struct {
	workspaceService *application.WorkspaceService
	inviteService    *application.InviteService
	baseURL          string
}

func NewWorkspaceHandler(wsService *application.WorkspaceService, inviteService *application.InviteService, baseURL string) *WorkspaceHandler {
	if baseURL == "" {
		baseURL = "http://localhost:3000"
	}
	return &WorkspaceHandler{
		workspaceService: wsService,
		inviteService:    inviteService,
		baseURL:          strings.TrimRight(baseURL, "/"),
	}
}

func (h *WorkspaceHandler) RegisterRoutes(router *gin.RouterGroup) {
	protected := router.Group("")
	protected.Use(middleware.Auth())

	protected.POST("/workspaces", h.Create)
	protected.GET("/workspaces", h.List)
	protected.POST("/workspaces/:id/invites", h.CreateInvite)
	protected.POST("/invites/:code/redeem", h.RedeemInvite)
}

func (h *WorkspaceHandler) Create(c *gin.Context) {
	userID, err := contextutil.UserID(c.Request.Context())
	if err != nil {
		httpresponse.Error(c, &pkgerrors.DomainError{
			Code:    pkgerrors.CodeUserUnauthorized,
			Message: "User tidak terautentikasi.",
			Err:     err,
		})
		return
	}

	var req *dto.CreateWorkspaceRequest

	if err := c.ShouldBindJSON(&req); err != nil {
		httpresponse.Error(c, err)
		return
	}

	workspace, err := h.workspaceService.Create(c.Request.Context(), userID, req.Name, *req.IconURL)
	if err != nil {
		httpresponse.Error(c, err)
		return
	}

	httpresponse.Created(c, workspace)
}

func (h *WorkspaceHandler) List(c *gin.Context) {
	userID, err := contextutil.UserID(c.Request.Context())
	if err != nil {
		httpresponse.Error(c, &pkgerrors.DomainError{
			Code:    pkgerrors.CodeUserUnauthorized,
			Message: "User tidak terautentikasi.",
			Err:     err,
		})
		return
	}

	var req *dto.ListWorkspacesRequest

	if err := c.ShouldBindJSON(&req); err != nil {
		httpresponse.Error(c, err)
		return
	}

	res, meta, err := h.workspaceService.ListByUserID(c.Request.Context(), userID, req)
	if err != nil {
		httpresponse.Error(c, err)
		return
	}

	httpresponse.SuccessWithMeta(c, 200, res, meta)
}

func (h *WorkspaceHandler) CreateInvite(c *gin.Context) {
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

	var req dto.CreateInviteRequest
	if c.Request.ContentLength > 0 {
		if err := c.ShouldBindJSON(&req); err != nil {
			httpresponse.Error(c, err)
			return
		}
	}

	// TODO: Task 3.5.x - Check permission MANAGE_INVITES using Permission Resolver once implemented

	userID, err := contextutil.UserID(c.Request.Context())
	if err != nil {
		httpresponse.Error(c, &pkgerrors.DomainError{
			Code:    pkgerrors.CodeUserUnauthorized,
			Message: "User tidak terautentikasi.",
			Err:     err,
		})
		return
	}

	ctx := contextutil.WithUserID(c.Request.Context(), userID)

	var expiresAt *time.Time
	if req.ExpiresInHours != nil {
		t := time.Now().Add(time.Duration(*req.ExpiresInHours) * time.Hour)
		expiresAt = &t
	}

	createReq := dto.CreateInviteReq{
		WorkspaceID: workspaceID,
		MaxUses:     req.MaxUses,
		ExpiresAt:   expiresAt,
	}

	invite, err := h.inviteService.Create(ctx, createReq)
	if err != nil {
		httpresponse.Error(c, err)
		return
	}

	resp := dto.CreateInviteResponse{
		Code: invite.Code,
		URL:  fmt.Sprintf("%s/invite/%s", h.baseURL, invite.Code),
	}

	httpresponse.Created(c, resp)
}

func (h *WorkspaceHandler) RedeemInvite(c *gin.Context) {
	idempotencyKey := c.GetHeader("Idempotency-Key")
	if idempotencyKey == "" {
		httpresponse.Error(c, &pkgerrors.DomainError{
			Code:    pkgerrors.CodeMissingRequiredField,
			Message: "Header Idempotency-Key wajib diisi.",
			Err:     errors.New("missing idempotency-key header"),
		})
		return
	}

	code := c.Param("code")
	if code == "" {
		httpresponse.Error(c, &pkgerrors.DomainError{
			Code:    pkgerrors.CodeMissingRequiredField,
			Message: "Kode undangan wajib diisi.",
			Err:     errors.New("missing invite code"),
		})
		return
	}

	userID, err := contextutil.UserID(c.Request.Context())
	if err != nil {
		httpresponse.Error(c, &pkgerrors.DomainError{
			Code:    pkgerrors.CodeUserUnauthorized,
			Message: "User tidak terautentikasi.",
			Err:     err,
		})
		return
	}

	member, err := h.inviteService.Redeem(c.Request.Context(), code, userID)
	if err != nil {
		httpresponse.Error(c, err)
		return
	}

	resp := dto.RedeemInviteResponse{
		WorkspaceID: member.WorkspaceID.String(),
		MemberID:    member.ID.String(),
	}

	httpresponse.OK(c, resp)
}
