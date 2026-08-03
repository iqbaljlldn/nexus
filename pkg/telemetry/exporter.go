package telemetry

import (
	"context"

	pkgerrors "github.com/iqbaljlldn/nexus/pkg/errors"

	"go.opentelemetry.io/otel/exporters/otlp/otlpmetric/otlpmetrichttp"
	"go.opentelemetry.io/otel/exporters/otlp/otlptrace/otlptracehttp"
	"go.opentelemetry.io/otel/exporters/prometheus"
	sdkmetric "go.opentelemetry.io/otel/sdk/metric"
	sdktrace "go.opentelemetry.io/otel/sdk/trace"
)

// newTraceExporter membuat OTLP HTTP trace exporter.
// Trace akan dikirimkan ke collector (Jaeger/Tempo/dsb) via HTTP.
func newTraceExporter(ctx context.Context, cfg Config) (sdktrace.SpanExporter, error) {
	opts := []otlptracehttp.Option{
		otlptracehttp.WithEndpoint(cfg.OTLPEndpoint),
	}

	// Gunakan insecure untuk development.
	if cfg.Environment == "development" {
		opts = append(opts, otlptracehttp.WithInsecure())
	}

	exporter, err := otlptracehttp.New(ctx, opts...)
	if err != nil {
		return nil, &pkgerrors.InfrastructureError{
			Message: "telemetry: creating trace exporter",
			Err:     err,
		}
	}

	return exporter, nil
}

// newMetricExporter membuat OTLP HTTP metric exporter.
func newMetricExporter(ctx context.Context, cfg Config) (sdkmetric.Exporter, error) {
	opts := []otlpmetrichttp.Option{
		otlpmetrichttp.WithEndpoint(cfg.OTLPEndpoint),
	}

	if cfg.Environment == "development" {
		opts = append(opts, otlpmetrichttp.WithInsecure())
	}

	exporter, err := otlpmetrichttp.New(ctx, opts...)
	if err != nil {
		return nil, &pkgerrors.InfrastructureError{
			Message: "telemetry: creating metric exporter",
			Err:     err,
		}
	}

	return exporter, nil
}

// newPrometheusExporter membuat Prometheus metric reader.
// Prometheus akan melakukan pull (scrape) dari endpoint /metrics.
func newPrometheusExporter() (*prometheus.Exporter, error) {
	exporter, err := prometheus.New()
	if err != nil {
		return nil, &pkgerrors.InfrastructureError{
			Message: "telemetry: creating prometheus exporter",
			Err:     err,
		}
	}

	return exporter, nil
}
