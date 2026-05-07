// Package tracing configures OpenTelemetry tracing for hammer.
//
// It supports three exporter modes selected by OTEL_EXPORTER:
//   - "stdout"  : human-readable spans to stderr (default for local dev without an endpoint)
//   - "otlp"    : OTLP gRPC to OTEL_EXPORTER_OTLP_ENDPOINT (e.g., local Jaeger at localhost:4317)
//   - "gcp"     : Google Cloud Trace exporter (used in CI/Cloud Run)
//   - "none"    : no-op (disables tracing)
package tracing

import (
	"context"
	"errors"
	"fmt"

	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/propagation"
	sdktrace "go.opentelemetry.io/otel/sdk/trace"
	"go.opentelemetry.io/otel/trace"
)

// ShutdownFunc flushes pending spans and shuts down the provider.
// Always safe to call (no-op if Init returned an error or was never called).
type ShutdownFunc func(context.Context) error

// noopShutdown is returned when initialization fails or tracing is disabled,
// so callers can defer it without nil checks.
var noopShutdown ShutdownFunc = func(context.Context) error { return nil }

// Init configures the global OpenTelemetry tracer provider and propagators.
// Returns a shutdown function that flushes spans before exit.
//
// Usage:
//
//	shutdown, err := tracing.Init(ctx)
//	if err != nil { logger.Warn("tracing init failed", "err", err) }
//	defer shutdown(context.Background())
func Init(ctx context.Context) (ShutdownFunc, error) {
	return InitWithConfig(ctx, ConfigFromEnv())
}

// InitWithConfig is like Init but takes an explicit configuration.
func InitWithConfig(ctx context.Context, cfg Config) (ShutdownFunc, error) {
	if cfg.Exporter == ExporterNone {
		// Set the no-op tracer provider explicitly so calls to otel.Tracer
		// don't accidentally use a real one from elsewhere.
		otel.SetTracerProvider(trace.NewNoopTracerProvider())
		return noopShutdown, nil
	}

	res, err := newResource(ctx, cfg)
	if err != nil {
		return noopShutdown, fmt.Errorf("create resource: %w", err)
	}

	exporter, err := newExporter(ctx, cfg)
	if err != nil {
		return noopShutdown, fmt.Errorf("create exporter: %w", err)
	}

	tp := sdktrace.NewTracerProvider(
		sdktrace.WithBatcher(exporter),
		sdktrace.WithResource(res),
		sdktrace.WithSampler(newSampler(cfg)),
	)

	otel.SetTracerProvider(tp)
	otel.SetTextMapPropagator(newPropagator())

	shutdown := func(ctx context.Context) error {
		// TracerProvider.Shutdown also shuts down the exporter.
		return errors.Join(tp.Shutdown(ctx))
	}
	return shutdown, nil
}

// Tracer returns a named tracer from the global provider.
// Use the package import path as the name (convention).
//
//	tracer := tracing.Tracer("github.com/you/hammer/internal/scan")
func Tracer(name string) trace.Tracer {
	return otel.Tracer(name)
}

// Start is a convenience wrapper that gets a tracer and starts a span in one call.
// Most callsites should use this instead of Tracer + Start.
//
//	ctx, span := tracing.Start(ctx, "scan.trivy")
//	defer span.End()
func Start(ctx context.Context, spanName string, opts ...trace.SpanStartOption) (context.Context, trace.Span) {
	return otel.Tracer("hammer").Start(ctx, spanName, opts...)
}

func newPropagator() propagation.TextMapPropagator {
	return propagation.NewCompositeTextMapPropagator(
		propagation.TraceContext{},
		propagation.Baggage{},
	)
}

func newSampler(cfg Config) sdktrace.Sampler {
	if cfg.SampleRatio >= 1.0 {
		return sdktrace.AlwaysSample()
	}
	if cfg.SampleRatio <= 0 {
		return sdktrace.NeverSample()
	}
	return sdktrace.ParentBased(sdktrace.TraceIDRatioBased(cfg.SampleRatio))
}
