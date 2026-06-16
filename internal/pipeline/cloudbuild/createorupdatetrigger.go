package cloudbuild

import (
	"context"

	"github.com/ghost-pack/hammer/internal/observability/tracing"
	"go.opentelemetry.io/otel/attribute"
	otelCodes "go.opentelemetry.io/otel/codes"
	"go.opentelemetry.io/otel/trace"
)

func (p *Pipeline) createOrUpdateTrigger(ctx context.Context) error {
	ctx, span := tracing.Tracer("gcloud builds triggers").Start(ctx, "gcloud builds triggers",
		trace.WithAttributes(
			attribute.String("cmd", "gcloud"),
			attribute.StringSlice("args", []string{"builds", "triggers"})))
	defer span.End()

	var properties Properties
	err := parseCloudBuildPath(p, &properties)
	if err != nil {
		span.RecordError(err)
		span.SetStatus(otelCodes.Error, err.Error())
		return err
	}

	err = p.cloudBuildClient.CreateOrUpdateCloudBuildTrigger(ctx, "cloud-build-pipeline-396819", "212799175996", "global", properties.Path, p.component.Name)
	if err != nil {
		span.RecordError(err)
		span.SetStatus(otelCodes.Error, err.Error())
		return err
	}

	span.SetStatus(otelCodes.Ok, "")
	return nil
}
