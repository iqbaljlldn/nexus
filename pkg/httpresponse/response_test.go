package httpresponse_test

import (
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/gin-gonic/gin"
	pkgerrors "github.com/iqbaljlldn/nexus/pkg/errors"
	"github.com/iqbaljlldn/nexus/pkg/httpresponse"
	"github.com/stretchr/testify/assert"
)

func TestResponseTimeHeader(t *testing.T) {
	gin.SetMode(gin.TestMode)

	t.Run("OK helper sets X-Response-Time when request_start_time exists", func(t *testing.T) {
		r := gin.New()
		r.Use(func(c *gin.Context) {
			c.Set(httpresponse.RequestStartTimeKey, time.Now().Add(-10*time.Millisecond))
			c.Next()
		})
		r.GET("/ok", func(c *gin.Context) {
			httpresponse.OK(c, gin.H{"data": "test"})
		})

		w := httptest.NewRecorder()
		req, _ := http.NewRequest(http.MethodGet, "/ok", nil)
		r.ServeHTTP(w, req)

		assert.Equal(t, http.StatusOK, w.Code)
		assert.NotEmpty(t, w.Header().Get(httpresponse.ResponseTimeHeader))
	})

	t.Run("Error helper sets X-Response-Time header", func(t *testing.T) {
		r := gin.New()
		r.Use(func(c *gin.Context) {
			c.Set(httpresponse.RequestStartTimeKey, time.Now().Add(-5*time.Millisecond))
			c.Next()
		})
		r.GET("/err", func(c *gin.Context) {
			httpresponse.Error(c, &pkgerrors.DomainError{
				Code:    pkgerrors.CodeInvalidCredentials,
				Message: "invalid creds",
				Err:     errors.New("bad creds"),
			})
		})

		w := httptest.NewRecorder()
		req, _ := http.NewRequest(http.MethodGet, "/err", nil)
		r.ServeHTTP(w, req)

		assert.Equal(t, http.StatusBadRequest, w.Code)
		assert.NotEmpty(t, w.Header().Get(httpresponse.ResponseTimeHeader))
	})
}
