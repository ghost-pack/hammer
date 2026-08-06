package cloudbuild

import (
	"context"

	"github.com/ghost-pack/hammer/internal/observability/tracing"
	otelCodes "go.opentelemetry.io/otel/codes"
)

func (p *Pipeline) createOrUpdateTrigger(ctx context.Context) error {
	ctx, span := tracing.Tracer("creating cloud build trigger").Start(ctx, "creating cloud build trigger")
	defer span.End()

	var props properties
	err := parseCloudBuildPath(p, &props)
	if err != nil {
		span.RecordError(err)
		span.SetStatus(otelCodes.Error, err.Error())
		return err
	}

	var triggerType string
	if props.TriggerType != "" {
		triggerType = props.TriggerType
	} else {
		triggerType = "webhook"
	}

	var pubSubTopic string
	if props.PubSubTopic != "" {
		pubSubTopic = props.PubSubTopic
	}

	err = p.cloudBuildClient.CreateOrUpdateCloudBuildTrigger(ctx, "hammer-central-prod", "598451979611", "global", props.Path, triggerType, p.component.Name, pubSubTopic, props.ManuallyApproved)
	if err != nil {
		span.RecordError(err)
		span.SetStatus(otelCodes.Error, err.Error())
		return err
	}

	span.SetStatus(otelCodes.Ok, "")
	return nil
}
