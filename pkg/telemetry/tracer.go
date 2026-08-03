package telemetry

import (
	"context"
	"net/http"

	pkgerrors "github.com/iqbaljlldn/nexus/pkg/errors"

	"go.opentelemetry.io/otel"
	sdkmetric "go.opentelemetry.io/otel/sdk/metric"
	"go.opentelemetry.io/otel/sdk/resource"
	sdktrace "go.opentelemetry.io/otel/sdk/trace"

	"github.com/prometheus/client_golang/prometheus/promhttp"
)

// Shutdown adalah tipe fungsi untuk mematikan seluruh provider dengan graceful.
type Shutdown func(context.Context) error

// Init menginisialisasi seluruh OpenTelemetry stack:
//   - Resource (identitas service)
//   - Propagator (W3C TraceContext + Baggage)
//   - TracerProvider (jika EnableTraces = true)
//   - MeterProvider (jika EnableMetrics = true, dengan Prometheus + OTLP opsional)
//
// Mengembalikan fungsi Shutdown yang WAJIB dipanggil saat graceful shutdown
// (misal di defer atau signal handler di main.go).
func Init(ctx context.Context, cfg Config) (Shutdown, error) {
	var shutdownFuncs []func(context.Context) error

	// 1. Resource
	res, err := newResource(ctx, cfg)
	if err != nil {
		res = fallbackResource(cfg)
	}

	// 2. Propagator
	otel.SetTextMapPropagator(newPropagator())

	// 3. TracerProvider
	if cfg.EnableTraces && cfg.OTLPEndpoint != "" {
		tp, err := initTracerProvider(ctx, cfg, res)
		if err != nil {
			return noopShutdown, &pkgerrors.InfrastructureError{
				Message: "telemetry: init tracer",
				Err:     err,
			}
		}
		otel.SetTracerProvider(tp)
		shutdownFuncs = append(shutdownFuncs, tp.Shutdown)
	}

	// 4. MeterProvider
	if cfg.EnableMetrics {
		mp, err := initMeterProvider(ctx, cfg, res)
		if err != nil {
			return noopShutdown, &pkgerrors.InfrastructureError{
				Message: "telemetry: init meter",
				Err:     err,
			}
		}
		otel.SetMeterProvider(mp)
		shutdownFuncs = append(shutdownFuncs, mp.Shutdown)

		// Jalankan Prometheus scrape endpoint di goroutine terpisah.
		go servePrometheus(cfg)
	}

	// Gabungkan semua shutdown menjadi satu fungsi.
	shutdown := func(ctx context.Context) error {
		var firstErr error
		for _, fn := range shutdownFuncs {
			if err := fn(ctx); err != nil && firstErr == nil {
				firstErr = err
			}
		}
		return firstErr
	}

	return shutdown, nil
}

// initTracerProvider membuat TracerProvider dengan OTLP HTTP exporter
// dan BatchSpanProcessor untuk performa tinggi.
func initTracerProvider(
	ctx context.Context,
	cfg Config,
	res *resource.Resource,
) (*sdktrace.TracerProvider, error) {
	exporter, err := newTraceExporter(ctx, cfg)
	if err != nil {
		return nil, err
	}

	tp := sdktrace.NewTracerProvider(
		sdktrace.WithResource(res),
		sdktrace.WithBatcher(exporter),
	)

	return tp, nil
}

// initMeterProvider membuat MeterProvider dengan:
//   - Prometheus exporter (pull-based, untuk scraping oleh Prometheus)
//   - OTLP metric exporter (push-based, opsional jika endpoint tersedia)
func initMeterProvider(
	ctx context.Context,
	cfg Config,
	res *resource.Resource,
) (*sdkmetric.MeterProvider, error) {
	// Prometheus reader (wajib jika metrics aktif).
	promExporter, err := newPrometheusExporter()
	if err != nil {
		return nil, err
	}

	opts := []sdkmetric.Option{
		sdkmetric.WithResource(res),
		sdkmetric.WithReader(promExporter),
	}

	// OTLP metric exporter (opsional, hanya jika endpoint tersedia).
	if cfg.OTLPEndpoint != "" {
		otlpExporter, err := newMetricExporter(ctx, cfg)
		if err != nil {
			return nil, err
		}
		opts = append(opts, sdkmetric.WithReader(
			sdkmetric.NewPeriodicReader(otlpExporter),
		))
	}

	mp := sdkmetric.NewMeterProvider(opts...)

	return mp, nil
}

func noopShutdown(_ context.Context) error {
	return nil
}

// servePrometheus menjalankan HTTP server terpisah untuk endpoint /metrics.
// Endpoint ini akan di-scrape oleh Prometheus secara periodik.
func servePrometheus(cfg Config) {
	port := cfg.PrometheusPort
	if port == "" {
		port = "9090"
	}

	mux := http.NewServeMux()
	mux.Handle("/metrics", promhttp.Handler())

	// Blocking call — dijalankan di goroutine oleh Init().
	_ = http.ListenAndServe(":"+port, mux) //nolint:gosec // this is just internal prometheus metric server
}
