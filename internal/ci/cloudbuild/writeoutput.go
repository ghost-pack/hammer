package cloudbuild

import (
	"context"
	"fmt"
	"os"

	"github.com/ghost-pack/hammer/internal/observability/tracing"
	"go.opentelemetry.io/otel/attribute"
	otelCodes "go.opentelemetry.io/otel/codes"
	"go.opentelemetry.io/otel/trace"
)

func (p *Pipeline) writeOutput(ctx context.Context) (err error) {
	ctx, span := tracing.Tracer("write cloudbuild output").Start(ctx, "upload cloudbuild output",
		trace.WithAttributes(
			attribute.String("bucket", p.releaseBucket),
			attribute.String("folder", fmt.Sprintf("%s/deployments/cloudbuild/%s.yaml", p.app.Metadata.Name, p.shortCommitSha))))
	defer span.End()
	defer func() {
		if err != nil {
			span.RecordError(err)
			span.SetStatus(otelCodes.Error, err.Error())
		} else {
			span.SetStatus(otelCodes.Ok, "Completed writing cloudbuild output")
		}
		span.End()
	}()

	props, err := parseCloudBuildPath(p)
	if err != nil {
		return err
	}
	cloudBuildYamlBytes, err := os.ReadFile(props.Path)
	if err != nil {
		return err
	}
	if err := p.cloudStorageClient.WriteObject(
		ctx,
		p.releaseBucket,
		fmt.Sprintf("%s/deployments/cloudbuild/%s.yaml", p.app.Metadata.Name, p.shortCommitSha),
		cloudBuildYamlBytes,
		nil,
	); err != nil {
		return fmt.Errorf("writing cloudbuild yaml to release bucket: %w", err)
	}

	return nil
}
