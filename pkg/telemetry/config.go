package telemetry

// Config menyimpan konfigurasi untuk OpenTelemetry.
type Config struct {
	// ServiceName adalah nama service yang akan muncul di traces dan metrics.
	ServiceName string

	// ServiceVersion adalah versi service saat ini (misal "v0.1.0").
	ServiceVersion string

	// Environment adalah environment runtime (development, staging, production).
	Environment string

	// OTLPEndpoint adalah alamat OTLP collector (misal "localhost:4318").
	// Kosong = exporter OTLP tidak diaktifkan (hanya Prometheus).
	OTLPEndpoint string

	// EnableTraces mengaktifkan tracing exporter.
	EnableTraces bool

	// EnableMetrics mengaktifkan metrics exporter.
	EnableMetrics bool

	// PrometheusPort adalah port untuk Prometheus scrape endpoint.
	// Default: 9090. Hanya dipakai jika EnableMetrics = true.
	PrometheusPort string
}
