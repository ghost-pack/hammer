package cloudbuild

import (
	"context"

	"github.com/ghost-pack/hammer/internal/observability/tracing"
	otelCodes "go.opentelemetry.io/otel/codes"
)

func (p *Pipeline) createOrUpdateTrigger(ctx context.Context) error {
	ctx, span := tracing.Tracer("creating cloud build trigger").Start(ctx, "creating cloud build trigger")
	defer span.End()

	var properties Properties
	err := parseCloudBuildPath(p, &properties)
	if err != nil {
		span.RecordError(err)
		span.SetStatus(otelCodes.Error, err.Error())
		return err
	}

	err = p.cloudBuildClient.CreateOrUpdateCloudBuildTrigger(ctx, "hammer-bootstrap", "40324185623", "global", properties.Path, "webhook", p.component.Name)
	if err != nil {
		span.RecordError(err)
		span.SetStatus(otelCodes.Error, err.Error())
		return err
	}

	span.SetStatus(otelCodes.Ok, "")
	return nil
}
