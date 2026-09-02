package middleware

import (
	"context"
	"time"

	"github.com/gin-gonic/gin"
)

const (
	RequestStartTimeKey = "request_start_time"
	ResponseTimeHeader  = "X-Response-Time"
)

type durationWriter struct {
	gin.ResponseWriter
	startTime time.Time
	written   bool
}

func (w *durationWriter) WriteHeader(code int) {
	if !w.written {
		w.written = true
		duration := time.Since(w.startTime)
		if w.Header().Get(ResponseTimeHeader) == "" {
			w.Header().Set(ResponseTimeHeader, duration.String())
		}
	}
	w.ResponseWriter.WriteHeader(code)
}

func (w *durationWriter) Write(b []byte) (int, error) {
	if !w.written {
		w.written = true
		duration := time.Since(w.startTime)
		if w.Header().Get(ResponseTimeHeader) == "" {
			w.Header().Set(ResponseTimeHeader, duration.String())
		}
	}
	return w.ResponseWriter.Write(b)
}

// RequestDurationMiddleware calculates request execution duration
// and injects the X-Response-Time header into the HTTP response.
func RequestDurationMiddleware(c *gin.Context) {
	start := time.Now()
	c.Set(RequestStartTimeKey, start)

	ctx := context.WithValue(c.Request.Context(), RequestStartTimeKey, start)
	c.Request = c.Request.WithContext(ctx)

	writer := &durationWriter{
		ResponseWriter: c.Writer,
		startTime:      start,
	}
	c.Writer = writer

	c.Next()
}

// GetRequestDuration calculates and returns elapsed duration for current request in gin context.
func GetRequestDuration(c *gin.Context) time.Duration {
	if startVal, exists := c.Get(RequestStartTimeKey); exists {
		if startTime, ok := startVal.(time.Time); ok {
			return time.Since(startTime)
		}
	}
	return 0
}
