package gcp

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"github.com/ghost-pack/hammer/internal/observability/tracing"
	"go.opentelemetry.io/otel/attribute"
	otelCodes "go.opentelemetry.io/otel/codes"
	"go.opentelemetry.io/otel/trace"
	"gopkg.in/yaml.v3"
)

func (p *Provisioner) writeStateToRegistry(ctx context.Context) error {
	ctx, span := tracing.Tracer("upload tenant state").Start(ctx, "upload tenant state",
		trace.WithAttributes(
			attribute.String("bucket", p.registryBucket),
			attribute.String("folder", fmt.Sprintf("tenants/%s", p.tenant.Metadata.Name))))
	defer span.End()

	p.newState.AppliedAt = time.Now().UTC()

	meta := map[string]string{
		"tenant-name": p.tenant.Metadata.Name,
		"applied-at":  p.newState.AppliedAt.Format(time.RFC3339),
	}

	// write state.json
	stateData, err := json.MarshalIndent(p.newState, "", "  ")
	if err != nil {
		span.RecordError(err)
		span.SetStatus(otelCodes.Error, err.Error())
		return fmt.Errorf("marshalling tenant state: %w", err)
	}
	if err := p.clients.CloudStorage.WriteObject(
		ctx,
		p.registryBucket,
		fmt.Sprintf("tenants/%s/state.json", p.tenant.Metadata.Name),
		stateData,
		meta,
	); err != nil {
		span.RecordError(err)
		span.SetStatus(otelCodes.Error, err.Error())
		return fmt.Errorf("writing state to registry: %w", err)
	}

	// write spec.yaml — audit trail of what was applied
	specData, err := yaml.Marshal(p.tenant)
	if err != nil {
		span.RecordError(err)
		span.SetStatus(otelCodes.Error, err.Error())
		return fmt.Errorf("marshalling tenant spec: %w", err)
	}
	if err := p.clients.CloudStorage.WriteObject(
		ctx,
		p.registryBucket,
		fmt.Sprintf("tenants/%s/spec.yaml", p.tenant.Metadata.Name),
		specData,
		meta,
	); err != nil {
		span.RecordError(err)
		span.SetStatus(otelCodes.Error, err.Error())
		return fmt.Errorf("writing spec to registry: %w", err)
	}

	span.SetStatus(otelCodes.Ok, "tenant state and spec written")
	return nil
}
