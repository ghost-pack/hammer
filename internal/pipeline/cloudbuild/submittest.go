package cloudbuild

import (
	"context"
	"fmt"

	"github.com/ghost-pack/hammer/internal/observability/tracing"
	"go.opentelemetry.io/otel/attribute"
	otelCodes "go.opentelemetry.io/otel/codes"
	"go.opentelemetry.io/otel/trace"
)

func (p *Pipeline) submitTest(ctx context.Context) error {
	ctx, span := tracing.Tracer("gcloud builds submit test").Start(ctx, "gcloud builds submit test",
		trace.WithAttributes(
			attribute.String("cmd", "gcloud"),
			attribute.StringSlice("args", []string{"builds", "submit"})))
	defer span.End()

	var properties Properties
	if err := p.component.Properties.Decode(&properties); err != nil {
		errorDecodingOamProperties := fmt.Errorf("decoding properties: %w", err)
		span.RecordError(errorDecodingOamProperties)
		span.SetStatus(otelCodes.Error, errorDecodingOamProperties.Error())
		return errorDecodingOamProperties
	}
	if properties.Path == "" {
		properties.Path = "./cloudbuild.yaml"
	}
	if properties.TestPath == "" {
		noCloudBuildTestErrors := fmt.Errorf("cloud build test required")
		span.RecordError(noCloudBuildTestErrors)
		span.SetStatus(otelCodes.Error, noCloudBuildTestErrors.Error())
		return noCloudBuildTestErrors
	}

	err := p.cloudBuildClient.TestCloudBuild(ctx, "cloud-build-pipeline-396819", "global", properties.Path, properties.TestPath)
	if err != nil {
		span.RecordError(err)
		span.SetStatus(otelCodes.Error, err.Error())
		return err
	}

	span.SetStatus(otelCodes.Ok, "")
	return nil
}
