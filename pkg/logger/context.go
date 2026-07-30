package logger

import (
	"context"

	"go.opentelemetry.io/otel/trace"
	"go.uber.org/zap"
)

type contextKey struct{}

var loggerContextKey = contextKey{}

func IntoContext(
	ctx context.Context,
	log *zap.Logger,
) context.Context {
	return context.WithValue(
		ctx,
		loggerContextKey,
		log,
	)
}

func FromContext(ctx context.Context, defaultLogger *zap.Logger) *zap.Logger {
	log := defaultLogger
	if l, ok := ctx.Value(loggerContextKey).(*zap.Logger); ok {
		log = l
	}

	span := trace.SpanContextFromContext(ctx)
	if span.IsValid() {
		log = log.With(
			zap.String(
				"trace_id",
				span.TraceID().String(),
			),
			zap.String(
				"span_id",
				span.SpanID().String(),
			),
			zap.Bool(
				"sampled",
				span.TraceFlags().IsSampled(),
			),
		)
	}

	return log
}
