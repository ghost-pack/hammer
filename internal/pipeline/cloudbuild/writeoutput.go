package cloudbuild

import (
	"context"
)

func (p *Pipeline) writeOutput(ctx context.Context) error {
	//ctx, span := tracing.Tracer("write cloudbuild output").Start(ctx, "upload cloudbuild output",
	//	trace.WithAttributes(
	//		attribute.String("bucket", "hammer-release"),
	//		attribute.String("folder", fmt.Sprintf("%s/", p.Metadata.Name))))
	//defer span.End()
	//
	//p.newState.AppliedAt = time.Now().UTC()
	//
	//meta := map[string]string{
	//	"tenant-name": p.tenant.Metadata.Name,
	//	"applied-at":  p.newState.AppliedAt.Format(time.RFC3339),
	//}
	//
	//// write state.json
	//stateData, err := json.MarshalIndent(p.newState, "", "  ")
	//if err != nil {
	//	span.RecordError(err)
	//	span.SetStatus(otelCodes.Error, err.Error())
	//	return fmt.Errorf("marshalling tenant state: %w", err)
	//}
	//if err := p.clients.CloudStorage.WriteObject(
	//	ctx,
	//	p.registryBucket,
	//	fmt.Sprintf("tenants/%s/state.json", p.tenant.Metadata.Name),
	//	stateData,
	//	meta,
	//); err != nil {
	//	span.RecordError(err)
	//	span.SetStatus(otelCodes.Error, err.Error())
	//	return fmt.Errorf("writing state to registry: %w", err)
	//}
	//
	//// write spec.yaml — audit trail of what was applied
	//specData, err := yaml.Marshal(p.tenant)
	//if err != nil {
	//	span.RecordError(err)
	//	span.SetStatus(otelCodes.Error, err.Error())
	//	return fmt.Errorf("marshalling tenant spec: %w", err)
	//}
	//if err := p.clients.CloudStorage.WriteObject(
	//	ctx,
	//	p.registryBucket,
	//	fmt.Sprintf("tenants/%s/spec.yaml", p.tenant.Metadata.Name),
	//	specData,
	//	meta,
	//); err != nil {
	//	span.RecordError(err)
	//	span.SetStatus(otelCodes.Error, err.Error())
	//	return fmt.Errorf("writing spec to registry: %w", err)
	//}
	//
	//span.SetStatus(otelCodes.Ok, "tenant state and spec written")
	return nil
}
