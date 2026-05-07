package logging

import (
	"context"
	"log/slog"

	"go.opentelemetry.io/otel/trace"
)

type traceHandler struct {
	slog.Handler
	project string
}

func (h *traceHandler) Handle(ctx context.Context, r slog.Record) error {
	span := trace.SpanContextFromContext(ctx)
	if span.IsValid() {
		traceID := span.TraceID().String()
		spanID := span.SpanID().String()

		if h.project != "" {
			r.AddAttrs(slog.String("logging.googleapis.com/trace", "projects/"+h.project+"/traces/"+traceID))
		} else {
			r.AddAttrs(slog.String("trace_id", traceID))
		}
		r.AddAttrs(slog.String("logging.googleapis.com/spanId", spanID))
		r.AddAttrs(slog.Bool("logging.googleapis.com/trace_sampled", span.IsSampled()))
	}
	return h.Handler.Handle(ctx, r)
}

func (h *traceHandler) WithAttr(attrs []slog.Attr) slog.Handler {
	return &traceHandler{Handler: h.Handler.WithAttrs(attrs), project: h.project}
}

func (h *traceHandler) WithGroup(name string) slog.Handler {
	return &traceHandler{Handler: h.Handler.WithGroup(name), project: h.project}
}
