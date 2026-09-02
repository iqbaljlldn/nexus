package http

import (
	"errors"
	"fmt"
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"github.com/iqbaljlldn/nexus/apps/api/internal/identity/application"
	"github.com/iqbaljlldn/nexus/apps/api/internal/identity/domain"
	"github.com/iqbaljlldn/nexus/apps/api/internal/platform/middleware"
	pkgerrors "github.com/iqbaljlldn/nexus/pkg/errors"
	"github.com/iqbaljlldn/nexus/pkg/httpresponse"
)

type AuthHandler struct {
	authService      *application.AuthService
	loginRateLimiter *middleware.LoginRateLimiter
}

func NewAuthHandler(authService *application.AuthService, loginRateLimiter *middleware.LoginRateLimiter) *AuthHandler {
	return &AuthHandler{
		authService:      authService,
		loginRateLimiter: loginRateLimiter,
	}
}

func (h *AuthHandler) RegisterRoutes(router *gin.RouterGroup) {
	router.POST("/auth/register", h.Register)
	router.POST("/auth/login", h.Login)
	router.POST("/auth/refresh", h.RefreshToken)
	router.POST("/auth/logout", h.Logout)

	protected := router.Group("/auth")
	protected.Use(middleware.Auth())
	protected.POST("/logout-all", h.LogoutAll)
	protected.GET("/sessions", h.ListSessions)
	protected.POST("/sessions/:id/revoke", h.RevokeSessionById)
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

type SessionResponse struct {
	ID        string `json:"id"`
	UserAgent string `json:"user_agent"`
	IPAddress string `json:"ip_address"`
	CreatedAt string `json:"created_at"`
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

	// Check if the identifier is currently locked out
	if h.loginRateLimiter != nil {
		result, err := h.loginRateLimiter.CheckLoginAllowed(c.Request.Context(), req.Identifier)
		if err != nil {
			_ = c.Error(fmt.Errorf("rate limit check failed: %w", err))
			return
		}
		if !result.Allowed {
			retryAfterSecs := int(result.RetryAfter.Seconds())
			if retryAfterSecs < 1 {
				retryAfterSecs = 1
			}
			c.Header("Retry-After", fmt.Sprintf("%d", retryAfterSecs))
			httpresponse.Error(c, &pkgerrors.DomainError{
				Code:    pkgerrors.CodeRateLimitExceeded,
				Message: "Too many failed login attempts. Please try again later.",
				Err:     errors.New("login rate limit exceeded"),
			})
			return
		}
	}

	deviceInfo := req.DeviceInfo
	if deviceInfo == nil {
		deviceInfo = &domain.DeviceInfo{
			DeviceID:  "unknown",
			IPAddress: c.ClientIP(),
			UserAgent: c.Request.UserAgent(),
		}
	}

	tokenPair, _, err := h.authService.Login(c.Request.Context(), req.Identifier, req.Password, deviceInfo)
	if err != nil {
		// Record failed attempt for rate limiting
		if h.loginRateLimiter != nil && errors.Is(err, domain.ErrInvalidCredentials) {
			result, rlErr := h.loginRateLimiter.RecordFailedAttempt(c.Request.Context(), req.Identifier)
			if rlErr == nil && !result.Allowed {
				// Lockout was just triggered by this attempt
				retryAfterSecs := int(result.RetryAfter.Seconds())
				if retryAfterSecs < 1 {
					retryAfterSecs = 1
				}
				c.Header("Retry-After", fmt.Sprintf("%d", retryAfterSecs))
				httpresponse.Error(c, &pkgerrors.DomainError{
					Code:    pkgerrors.CodeRateLimitExceeded,
					Message: "Too many failed login attempts. Please try again later.",
					Err:     errors.New("login rate limit exceeded"),
				})
				return
			}
		}

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

	// Login succeeded — reset rate limit state for this identifier
	if h.loginRateLimiter != nil {
		_ = h.loginRateLimiter.ResetOnSuccess(c.Request.Context(), req.Identifier)
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

func (h *AuthHandler) Logout(c *gin.Context) {
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

	err = h.authService.Logout(c.Request.Context(), refreshToken)
	if err != nil {
		httpresponse.Error(c, err)
		return
	}

	// Delete cookies
	c.SetCookie("refresh_token", "", -1, "/", "", true, true)
	c.SetCookie("csrf_token", "", -1, "/", "", true, false)

	httpresponse.NoContent(c)
}

func (h *AuthHandler) LogoutAll(c *gin.Context) {
	e := h.authService.LogoutAll(c.Request.Context())
	if e != nil {
		httpresponse.Error(c, e)
		return
	}

	c.SetCookie("refresh_token", "", -1, "/", "", true, true)
	c.SetCookie("csrf_token", "", -1, "/", "", true, false)

	c.Status(http.StatusNoContent)
}

func (h *AuthHandler) ListSessions(c *gin.Context) {
	sessions, err := h.authService.GetActiveSessions(c.Request.Context())
	if err != nil {
		httpresponse.Error(c, err)
		return
	}

	sessionResp := make([]SessionResponse, 0, len(sessions))
	for _, s := range sessions {
		sessionResp = append(sessionResp, SessionResponse{
			ID:        s.ID,
			UserAgent: s.UserAgent,
			IPAddress: s.IPAddress,
			CreatedAt: s.CreatedAt.Format("2006-01-02T15:04:05Z07:00"),
		})
	}

	httpresponse.OK(c, sessionResp)
}

func (h *AuthHandler) RevokeSessionById(c *gin.Context) {
	sessionID := c.Param("id")

	if sessionID == "" {
		httpresponse.Error(c, &pkgerrors.DomainError{
			Code:    pkgerrors.CodeInvalidFieldFormat,
			Message: "Session ID not found",
			Err:     errors.New("session id not found"),
		})
		return
	}

	session, err := h.authService.RevokeSessionById(c.Request.Context(), sessionID)
	if err != nil {
		var domainErr *pkgerrors.DomainError

		if errors.Is(err, domain.ErrInvalidToken) {
			domainErr = &pkgerrors.DomainError{
				Code:    pkgerrors.CodeTokenInvalid,
				Message: "Invalid or expired token",
				Err:     err,
			}
		} else if errors.Is(err, domain.ErrUnauthorizedSession) {
			domainErr = &pkgerrors.DomainError{
				Code:    pkgerrors.CodeForbidden,
				Message: "Forbidden to revoke this session",
				Err:     err,
			}
		} else {
			_ = c.Error(err)
			return
		}

		httpresponse.Error(c, domainErr)
		return
	}

	if session.ID == sessionID {
		c.SetCookie("refresh_token", "", -1, "/", "", true, true)
		c.SetCookie("csrf_token", "", -1, "/", "", true, false)
	}

	httpresponse.OK(c, session)
}
