package telemetry

import (
	"fmt"
	"time"

	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/metric"
)

// Metrics menyimpan kumpulan instrument metric yang umum digunakan
// lintas domain. Domain-specific metrics sebaiknya didefinisikan
// di masing-masing domain package, bukan di sini.
type Metrics struct {
	// HTTPRequestDuration mengukur durasi setiap HTTP request (histogram).
	HTTPRequestDuration metric.Float64Histogram

	// HTTPRequestTotal menghitung total HTTP request per status code (counter).
	HTTPRequestTotal metric.Int64Counter

	// ActiveConnections mengukur jumlah koneksi aktif saat ini (gauge via UpDownCounter).
	ActiveConnections metric.Int64UpDownCounter
}

// NewMetrics membuat instrument-instrument metric global.
// Dipanggil setelah Init() agar MeterProvider sudah ter-set.
func NewMetrics(serviceName string) (*Metrics, error) {
	meter := otel.Meter(serviceName)

	reqDuration, err := meter.Float64Histogram(
		"http.server.request.duration",
		metric.WithDescription("Duration of HTTP server requests in seconds"),
		metric.WithUnit("s"),
		metric.WithExplicitBucketBoundaries(
			0.005, 0.01, 0.025, 0.05, 0.1, 0.25, 0.5, 1.0, 2.5, 5.0, 10.0,
		),
	)
	if err != nil {
		return nil, fmt.Errorf("telemetry: creating request duration histogram: %w", err)
	}

	reqTotal, err := meter.Int64Counter(
		"http.server.request.total",
		metric.WithDescription("Total number of HTTP server requests"),
	)
	if err != nil {
		return nil, fmt.Errorf("telemetry: creating request total counter: %w", err)
	}

	activeConns, err := meter.Int64UpDownCounter(
		"http.server.active_connections",
		metric.WithDescription("Number of active HTTP connections"),
	)
	if err != nil {
		return nil, fmt.Errorf("telemetry: creating active connections gauge: %w", err)
	}

	return &Metrics{
		HTTPRequestDuration: reqDuration,
		HTTPRequestTotal:    reqTotal,
		ActiveConnections:   activeConns,
	}, nil
}

// RecordDuration adalah helper untuk merekam durasi dalam detik.
func RecordDuration(start time.Time) float64 {
	return time.Since(start).Seconds()
}
