package tracing

import (
	"context"
	"fmt"
	"os"

	texporter "github.com/GoogleCloudPlatform/opentelemetry-operations-go/exporter/trace"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/exporters/otlp/otlptrace/otlptracegrpc"
	"go.opentelemetry.io/otel/exporters/stdout/stdouttrace"
	"go.opentelemetry.io/otel/sdk/resource"
	sdktrace "go.opentelemetry.io/otel/sdk/trace"
	semconv "go.opentelemetry.io/otel/semconv/v1.26.0"
)

// newResource builds the OTel resource (set of attributes describing this process).
func newResource(ctx context.Context, cfg Config) (*resource.Resource, error) {
	baseAttrs := []attribute.KeyValue{
		semconv.ServiceName(cfg.ServiceName),
		semconv.ServiceVersion(cfg.ServiceVersion),
		semconv.DeploymentEnvironment(cfg.Environment),
		semconv.ServiceInstanceID(hostnameOr("unknown")),
	}
	if cfg.Commit != "" {
		// No semantic convention for commit; use a custom key.
		baseAttrs = append(baseAttrs, attribute.String("vcs.revision", cfg.Commit))
	}

	return resource.New(ctx,
		resource.WithFromEnv(),      // OTEL_RESOURCE_ATTRIBUTES env var
		resource.WithTelemetrySDK(), // SDK info
		resource.WithProcess(),      // process info (pid, exe, etc.)
		resource.WithHost(),         // host info
		resource.WithAttributes(baseAttrs...),
	)
}

func hostnameOr(fallback string) string {
	if h, err := os.Hostname(); err == nil && h != "" {
		return h
	}
	return fallback
}

// newExporter constructs the configured exporter.
func newExporter(ctx context.Context, cfg Config) (sdktrace.SpanExporter, error) {
	switch cfg.Exporter {
	case ExporterStdout:
		return stdouttrace.New(stdouttrace.WithPrettyPrint())

	case ExporterOTLP:
		opts := []otlptracegrpc.Option{
			otlptracegrpc.WithEndpoint(cfg.OTLPEndpoint),
		}
		if cfg.OTLPInsecure {
			opts = append(opts, otlptracegrpc.WithInsecure())
		}
		return otlptracegrpc.New(ctx, opts...)

	case ExporterGCP:
		if cfg.GCPProject == "" {
			return nil, fmt.Errorf("gcp exporter selected but GOOGLE_CLOUD_PROJECT not set")
		}
		return texporter.New(texporter.WithProjectID(cfg.GCPProject))

	default:
		return nil, fmt.Errorf("unknown exporter: %q", cfg.Exporter)
	}
}
