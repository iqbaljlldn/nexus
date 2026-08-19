package http

import (
	"errors"

	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"github.com/iqbaljlldn/nexus/apps/api/internal/identity/application"
	"github.com/iqbaljlldn/nexus/apps/api/internal/identity/domain"
	pkgerrors "github.com/iqbaljlldn/nexus/pkg/errors"
	"github.com/iqbaljlldn/nexus/pkg/httpresponse"
)

type AuthHandler struct {
	authService *application.AuthService
}

func NewAuthHandler(authService *application.AuthService) *AuthHandler {
	return &AuthHandler{
		authService: authService,
	}
}

func (h *AuthHandler) RegisterRoutes(router *gin.RouterGroup) {
	router.POST("/auth/register", h.Register)
	router.POST("/auth/login", h.Login)
	router.POST("/auth/refresh", h.RefreshToken)
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

type LoginRequest struct {
	Identifier string             `json:"identifier" binding:"required"`
	Password   string             `json:"password" binding:"required"`
	DeviceInfo *domain.DeviceInfo `json:"device_info"`
}

type LoginResponse struct {
	AccessToken string `json:"access_token"`
	ExpiresIn   int    `json:"expires_in"` // in seconds
}

type RefreshTokenResponse struct {
	AccessToken string `json:"access_token"`
	ExpiresIn   int    `json:"expires_in"` // in seconds
}

func (h *AuthHandler) Register(c *gin.Context) {
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

func (h *AuthHandler) Login(c *gin.Context) {
	var req LoginRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(400, gin.H{"success": false, "error": gin.H{"code": "VALIDATION_ERROR", "message": "Invalid input format"}})
		return
	}

	tokenPair, _, err := h.authService.Login(c, req.Identifier, req.Password, req.DeviceInfo)
	if err != nil {
		var domainErr *pkgerrors.DomainError

		if errors.Is(err, domain.ErrUserNotFound) {
			domainErr = &pkgerrors.DomainError{
				Code:    pkgerrors.CodeUserNotFound,
				Message: "User not found",
				Err:     err,
			}
		} else if errors.Is(err, domain.ErrInvalidCredentials) {
			domainErr = &pkgerrors.DomainError{
				Code:    pkgerrors.CodeInvalidCredentials,
				Message: "Invalid credentials",
				Err:     err,
			}
		} else {
			_ = c.Error(err)
			return
		}

		httpresponse.Error(c, domainErr)
		return
	}

	// Set Refresh Token as HttpOnly Secure cookie
	c.SetSameSite(http.SameSiteStrictMode)
	c.SetCookie("refresh_token", tokenPair.RefreshToken, 86400, "/", "", true, true)

	// Set CSRF Token as non-HttpOnly Secure cookie
	csrfToken := uuid.NewString()
	c.SetCookie("csrf_token", csrfToken, 86400, "/", "", true, false)

	resp := LoginResponse{
		AccessToken: tokenPair.AccessToken,
		ExpiresIn:   900, // 15 minutes in seconds
	}

	httpresponse.OK(c, resp)
}

func (h *AuthHandler) RefreshToken(c *gin.Context) {
	csrfCookie, err := c.Cookie("csrf_token")
	if err != nil {
		httpresponse.Error(c, &pkgerrors.DomainError{
			Code:    pkgerrors.CodeForbidden,
			Message: "CSRF token missing",
			Err:     err,
		})
		return
	}

	csrfHeader := c.GetHeader("X-CSRF-Token")
	if csrfHeader == "" || csrfHeader != csrfCookie {
		httpresponse.Error(c, &pkgerrors.DomainError{
			Code:    pkgerrors.CodeForbidden,
			Message: "Invalid CSRF token",
			Err:     errors.New("csrf token mismatch"),
		})
		return
	}

	refreshToken, _ := c.Cookie("refresh_token")
	if refreshToken == "" {
		httpresponse.Error(c, &pkgerrors.DomainError{
			Code:    pkgerrors.CodeInvalidFieldFormat,
			Message: "Refresh token not found",
			Err:     errors.New("refresh token not found"),
		})
		return
	}

	tokenPair, err := h.authService.RefreshToken(c.Request.Context(), refreshToken)
	if err != nil {
		var domainErr *pkgerrors.DomainError

		if errors.Is(err, domain.ErrInvalidToken) {
			domainErr = &pkgerrors.DomainError{
				Code:    pkgerrors.CodeTokenInvalid,
				Message: "Invalid or expired token",
				Err:     err,
			}
		} else {
			_ = c.Error(err)
			return
		}

		httpresponse.Error(c, domainErr)
		return
	}

	// Set Refresh Token as HttpOnly Secure cookie
	c.SetSameSite(http.SameSiteStrictMode)
	c.SetCookie("refresh_token", tokenPair.RefreshToken, 86400, "/", "", true, true)

	// Set CSRF Token as non-HttpOnly Secure cookie
	newCsrfToken := uuid.NewString()
	c.SetCookie("csrf_token", newCsrfToken, 86400, "/", "", true, false)

	resp := RefreshTokenResponse{
		AccessToken: tokenPair.AccessToken,
		ExpiresIn:   900, // 15 minutes in seconds
	}

	httpresponse.OK(c, resp)
}
