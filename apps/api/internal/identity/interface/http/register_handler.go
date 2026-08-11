package http

import (
	"errors"

	"github.com/gin-gonic/gin"
	"github.com/iqbaljlldn/nexus/apps/api/internal/identity/application"
	"github.com/iqbaljlldn/nexus/apps/api/internal/identity/domain"
	pkgerrors "github.com/iqbaljlldn/nexus/pkg/errors"
	"github.com/iqbaljlldn/nexus/pkg/httpresponse"
)

type RegisterHandler struct {
	authService *application.AuthService
}

func NewRegisterHandler(authService *application.AuthService) *RegisterHandler {
	return &RegisterHandler{
		authService: authService,
	}
}

func (h *RegisterHandler) RegisterRoutes(router *gin.RouterGroup) {
	router.POST("/auth/register", h.Register)
}

type RegisterRequest struct {
	Email       string `json:"email" binding:"required,email"`
	Username    string `json:"username" binding:"required,min=3,max=32,alphanum"`
	DisplayName string `json:"display_name" binding:"required,min=1,max=100"`
	Password    string `json:"password" binding:"required,min=8"`
}

type RegisterResponse struct {
	ID          string `json:"id"`
	Email       string `json:"email"`
	Username    string `json:"username"`
	DisplayName string `json:"display_name"`
}

func (h *RegisterHandler) Register(c *gin.Context) {
	var req RegisterRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(400, gin.H{"success": false, "error": gin.H{"code": "VALIDATION_ERROR", "message": "Invalid input format"}})
		return
	}

	user, err := h.authService.Register(c.Request.Context(), req.Email, req.Username, req.DisplayName, req.Password)
	if err != nil {
		var domainErr *pkgerrors.DomainError

		if errors.Is(err, domain.ErrDuplicateEmail) {
			domainErr = &pkgerrors.DomainError{
				Code:    pkgerrors.CodeUserAlreadyExists,
				Message: "Email already in use",
				Err:     err,
			}
		} else if errors.Is(err, domain.ErrDuplicateUsername) {
			domainErr = &pkgerrors.DomainError{
				Code:    pkgerrors.CodeUserAlreadyExists,
				Message: "Username already in use",
				Err:     err,
			}
		} else if errors.Is(err, domain.ErrInvalidEmail) || errors.Is(err, domain.ErrInvalidUsername) {
			domainErr = &pkgerrors.DomainError{
				Code:    pkgerrors.CodeInvalidFieldFormat,
				Message: err.Error(),
				Err:     err,
			}
		} else {
			_ = c.Error(err)
			return
		}

		httpresponse.Error(c, domainErr)
		return
	}

	resp := RegisterResponse{
		ID:          user.ID.String(),
		Email:       user.Email.String(),
		Username:    user.Username.String(),
		DisplayName: user.DisplayName,
	}

	httpresponse.Created(c, resp)
}
