package gcp

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"

	"cloud.google.com/go/storage"
	"github.com/ghost-pack/hammer/internal/observability/tracing"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/codes"
	"go.opentelemetry.io/otel/trace"
)

func (p *Provisioner) loadLastAppliedState(ctx context.Context) error {
	ctx, span := tracing.Tracer("load old tenant state").Start(ctx, "load old tenant state",
		trace.WithAttributes(
			attribute.String("bucket", p.registryBucket),
			attribute.String("object", fmt.Sprintf("tenants/%s/state.json", p.tenant.Metadata.Name))))
	defer span.End()

	stateData, err := p.clients.CloudStorage.GetObject(
		ctx,
		p.registryBucket,
		fmt.Sprintf("tenants/%s/state.json", p.tenant.Metadata.Name),
	)
	if err != nil {
		if errors.Is(err, storage.ErrObjectNotExist) {
			span.SetStatus(codes.Ok, "no existing state — first run")
			return nil
		}
		span.RecordError(err)
		span.SetStatus(codes.Error, err.Error())
		return fmt.Errorf("loading last applied state: %w", err)
	}

	var oldState TenantState
	if err := json.Unmarshal(stateData, &oldState); err != nil {
		span.RecordError(err)
		span.SetStatus(codes.Error, err.Error())
		return fmt.Errorf("unmarshalling tenant state: %w", err)
	}

	p.lastAppliedState = &oldState
	span.SetStatus(codes.Ok, "loaded old tenant state")
	return nil
}
