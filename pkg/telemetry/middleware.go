package telemetry

import (
	"fmt"
	"time"

	"github.com/gin-gonic/gin"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/metric"
	"go.opentelemetry.io/contrib/instrumentation/github.com/gin-gonic/gin/otelgin"
)

// TraceMiddleware mengembalikan Gin middleware dari otelgin
// yang secara otomatis membuat span untuk setiap HTTP request,
// mengekstrak/propagasi W3C TraceContext header, dan mencatat
// atribut HTTP standar (method, status_code, url, dsb.).
func TraceMiddleware(serviceName string) gin.HandlerFunc {
	return otelgin.Middleware(serviceName)
}

// MetricsMiddleware mengembalikan Gin middleware yang merekam
// HTTP metrics (duration histogram, request counter, active connections)
// menggunakan instrument dari Metrics struct.
func MetricsMiddleware(m *Metrics) gin.HandlerFunc {
	return func(c *gin.Context) {
		start := time.Now()

		// Increment active connections.
		m.ActiveConnections.Add(c.Request.Context(), 1)

		c.Next()

		// Decrement active connections.
		m.ActiveConnections.Add(c.Request.Context(), -1)

		// Record metrics setelah request selesai.
		duration := RecordDuration(start)
		status := c.Writer.Status()

		path := c.FullPath()
		if path == "" {
			path = c.Request.URL.Path
		}

		attrs := metric.WithAttributes(
			attribute.String("http.method", c.Request.Method),
			attribute.String("http.route", path),
			attribute.Int("http.status_code", status),
		)

		m.HTTPRequestDuration.Record(c.Request.Context(), duration, attrs)
		m.HTTPRequestTotal.Add(c.Request.Context(), 1, attrs)
	}
}

// SetupMiddlewares adalah convenience function yang mengembalikan
// kedua middleware (trace + metrics) sekaligus.
// Jika metrics nil (belum diinisialisasi), hanya trace middleware yang dikembalikan.
func SetupMiddlewares(serviceName string, m *Metrics) []gin.HandlerFunc {
	middlewares := []gin.HandlerFunc{
		TraceMiddleware(serviceName),
	}

	if m != nil {
		middlewares = append(middlewares, MetricsMiddleware(m))
	}

	return middlewares
}

// MustNewMetrics sama seperti NewMetrics tapi panic jika gagal.
// Hanya digunakan di main.go/entrypoint di mana kegagalan inisialisasi
// berarti aplikasi tidak bisa berjalan.
func MustNewMetrics(serviceName string) *Metrics {
	m, err := NewMetrics(serviceName)
	if err != nil {
		panic(fmt.Sprintf("telemetry: failed to create metrics: %v", err))
	}
	return m
}
