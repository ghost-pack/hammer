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
	if properties.Tests == nil {
		noCloudBuildTestErrors := fmt.Errorf("cloud build tests required")
		span.RecordError(noCloudBuildTestErrors)
		span.SetStatus(otelCodes.Error, noCloudBuildTestErrors.Error())
		return noCloudBuildTestErrors
	}

	for _, ph := range properties.Tests {
		required := ph.Required != nil && *ph.Required
		err := p.cloudBuildClient.TestCloudBuild(ctx, "cloud-build-pipeline-396819", "global", properties.Path, ph.Path)
		if err != nil && required {
			span.RecordError(err)
			span.SetStatus(otelCodes.Error, err.Error())
			return err
		} else if err == nil && !required {
			testFailureError := fmt.Errorf("test was supposed to fail")
			span.RecordError(testFailureError)
			span.SetStatus(otelCodes.Error, testFailureError.Error())
			return testFailureError
		}
	}

	span.SetStatus(otelCodes.Ok, "")
	return nil
}
