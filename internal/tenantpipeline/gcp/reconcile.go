package gcp

import (
	"context"

	"github.com/ghost-pack/hammer/internal/observability/tracing"
	"go.opentelemetry.io/otel/codes"
)

func (p *Provisioner) reconcile(ctx context.Context) error {
	ctx, span := tracing.Tracer("reconcile tenant state").Start(ctx, "reconcile tenant state")
	defer span.End()

	p.newState.Name = p.tenant.Metadata.Name
	apisToAdd, apisToRemove := diffApis(p.lastAppliedState.AllowedApis, p.tenant.Spec.AllowedApis)
	p.apisToAdd = apisToAdd
	p.apisToRemove = apisToRemove

	span.SetStatus(codes.Ok, "Successfully reconciled tenant state")
	return nil
}

// Note: the plan here is to not actually disable apis. Just to remove roles from the sa-pipeline service account.
func diffApis(old, new []string) (apisToAdd, apisToRemove []string) {
	oldSet := make(map[string]bool, len(old))
	for _, api := range old {
		oldSet[api] = true
	}
	newSet := make(map[string]bool, len(new))
	for _, api := range new {
		newSet[api] = true
		if !oldSet[api] {
			apisToAdd = append(apisToAdd, api)
		}
	}
	for _, api := range old {
		if !newSet[api] {
			apisToRemove = append(apisToRemove, api)
		}
	}
	return

}
