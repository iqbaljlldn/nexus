package middleware

import (
	"net/http"
	"os"
	"strings"

	"github.com/gin-gonic/gin"
)

// CORSMiddleware handles Cross-Origin Resource Sharing (CORS) preflight and header policies.
// Allowed origins can be configured via environment variable CORS_ALLOWED_ORIGINS (comma-separated).
// If CORS_ALLOWED_ORIGINS is empty, it defaults to allowing the requesting origin.
func CORSMiddleware() gin.HandlerFunc {
	allowedOriginsEnv := os.Getenv("CORS_ALLOWED_ORIGINS")
	var allowedOrigins []string
	if allowedOriginsEnv != "" {
		allowedOrigins = strings.Split(allowedOriginsEnv, ",")
		for i := range allowedOrigins {
			allowedOrigins[i] = strings.TrimSpace(allowedOrigins[i])
		}
	}

	return func(c *gin.Context) {
		origin := c.Request.Header.Get("Origin")

		if origin != "" {
			allowOrigin := false
			if len(allowedOrigins) > 0 {
				for _, o := range allowedOrigins {
					if o == "*" || o == origin {
						allowOrigin = true
						break
					}
				}
			} else {
				// Default fallback: allow requesting origin for dev / single domain
				allowOrigin = true
			}

			if allowOrigin {
				c.Writer.Header().Set("Access-Control-Allow-Origin", origin)
				c.Writer.Header().Set("Access-Control-Allow-Credentials", "true")
				c.Writer.Header().Set("Access-Control-Allow-Headers", "Content-Type, Content-Length, Accept-Encoding, X-CSRF-Token, Authorization, accept, origin, Cache-Control, X-Requested-With, Idempotency-Key")
				c.Writer.Header().Set("Access-Control-Allow-Methods", "POST, OPTIONS, GET, PUT, DELETE, PATCH")
				c.Writer.Header().Set("Access-Control-Max-Age", "86400")
			}
		}

		if c.Request.Method == http.MethodOptions {
			c.AbortWithStatus(http.StatusNoContent)
			return
		}

		c.Next()
	}
}
