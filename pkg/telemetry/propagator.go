package telemetry

import (
	"go.opentelemetry.io/otel/propagation"
)

// newPropagator membuat composite propagator yang mendukung
// W3C TraceContext dan Baggage. Propagator ini memastikan
// trace_id dan span_id terbawa di header HTTP lintas service.
func newPropagator() propagation.TextMapPropagator {
	return propagation.NewCompositeTextMapPropagator(
		propagation.TraceContext{},
		propagation.Baggage{},
	)
}
