package middleware

import (
	"errors"
	"net/http"
	"strings"

	"github.com/gin-gonic/gin"
	pkgerrors "github.com/iqbaljlldn/nexus/pkg/errors"
	"github.com/iqbaljlldn/nexus/pkg/httpresponse"
	"github.com/iqbaljlldn/nexus/pkg/jwt"
)

// Auth is a gin middleware that parses the Authorization header and verifies the JWT.
func Auth() gin.HandlerFunc {
	return func(c *gin.Context) {
		authHeader := c.GetHeader("Authorization")
		if authHeader == "" {
			err := &pkgerrors.DomainError{
				Code:    pkgerrors.CodeUserUnauthorized,
				Message: "missing authorization header",
				Err:     errors.New("missing authorization header"),
			}
			c.AbortWithStatusJSON(http.StatusUnauthorized, httpresponse.ErrorResponse{
				Success: false,
				Error: httpresponse.ErrorDetail{
					Code:    err.Code,
					Message: err.Message,
				},
			})
			return
		}

		parts := strings.SplitN(authHeader, " ", 2)
		if len(parts) != 2 || strings.ToLower(parts[0]) != "bearer" {
			err := &pkgerrors.DomainError{
				Code:    pkgerrors.CodeUserUnauthorized,
				Message: "invalid authorization header format",
				Err:     errors.New("invalid authorization header format"),
			}
			c.AbortWithStatusJSON(http.StatusUnauthorized, httpresponse.ErrorResponse{
				Success: false,
				Error: httpresponse.ErrorDetail{
					Code:    err.Code,
					Message: err.Message,
				},
			})
			return
		}

		tokenStr := parts[1]
		claims := &jwt.BaseClaims{}
		err := jwt.Verify(tokenStr, claims)
		if err != nil {
			var code string
			var message string
			if errors.Is(err, jwt.ErrExpiredToken) {
				code = pkgerrors.CodeTokenExpired
				message = "token expired"
			} else {
				code = pkgerrors.CodeTokenInvalid
				message = "invalid token"
			}

			domainErr := &pkgerrors.DomainError{
				Code:    code,
				Message: message,
				Err:     err,
			}
			c.AbortWithStatusJSON(http.StatusUnauthorized, httpresponse.ErrorResponse{
				Success: false,
				Error: httpresponse.ErrorDetail{
					Code:    domainErr.Code,
					Message: domainErr.Message,
				},
			})
			return
		}

		// Set user_id in the gin context
		c.Set("user_id", claims.UserID)
		c.Next()
	}
}
