package observability

import (
	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/trace"
)

func Tracer(pkg string) trace.Tracer {
	return otel.Tracer("hammer-pipeline/" + pkg)
}
