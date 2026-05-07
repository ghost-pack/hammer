package tracing

import (
	"context"
	"testing"

	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/trace"
)

func TestInitNoneIsNoOp(t *testing.T) {
	cfg := Config{Exporter: ExporterNone}
	shutdown, err := InitWithConfig(context.Background(), cfg)
	if err != nil {
		t.Fatalf("init failed: %v", err)
	}
	defer shutdown(context.Background())

	ctx, span := Start(context.Background(), "test-span")
	defer span.End()

	sc := trace.SpanContextFromContext(ctx)
	// Noop tracer produces invalid span contexts.
	if sc.IsValid() {
		t.Errorf("expected invalid span context with no-op exporter, got valid")
	}
}

func TestInitStdoutEmitsValidSpans(t *testing.T) {
	cfg := Config{
		Exporter:    ExporterStdout,
		ServiceName: "test",
		SampleRatio: 1.0,
	}
	shutdown, err := InitWithConfig(context.Background(), cfg)
	if err != nil {
		t.Fatalf("init failed: %v", err)
	}
	defer shutdown(context.Background())

	tracer := otel.Tracer("test")
	ctx, span := tracer.Start(context.Background(), "root")

	sc := trace.SpanContextFromContext(ctx)
	if !sc.IsValid() {
		t.Errorf("span context should be valid")
	}
	if !sc.TraceID().IsValid() {
		t.Errorf("trace id should be valid")
	}
	span.End()
}

func TestPickExporterDefaultsToStdout(t *testing.T) {
	cfg := Config{}
	got := pickExporter("stdout", cfg)
	if got != ExporterStdout {
		t.Errorf("expected stdout default, got %q", got)
	}
}

func TestPickExporterRespectsExplicit(t *testing.T) {
	cfg := Config{}
	got := pickExporter("gcp", cfg)
	if got != ExporterGCP {
		t.Errorf("expected gcp, got %q", got)
	}
}
