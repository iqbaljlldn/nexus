package middleware_test

import (
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/iqbaljlldn/nexus/apps/api/internal/platform/middleware"
	"github.com/stretchr/testify/assert"
)

func TestRequestDurationMiddleware(t *testing.T) {
	gin.SetMode(gin.TestMode)

	r := gin.New()
	r.Use(middleware.RequestDurationMiddleware)

	var recordedDuration time.Duration

	r.GET("/test", func(c *gin.Context) {
		time.Sleep(5 * time.Millisecond)
		recordedDuration = middleware.GetRequestDuration(c)
		c.JSON(http.StatusOK, gin.H{"message": "ok"})
	})

	w := httptest.NewRecorder()
	req, _ := http.NewRequest(http.MethodGet, "/test", nil)
	r.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)
	assert.NotEmpty(t, w.Header().Get("X-Response-Time"))
	assert.True(t, recordedDuration > 0, "recorded duration should be greater than zero")
}
