package gcp

import (
	"context"
)

func (p *Provisioner) enableApis(ctx context.Context) error {
	for _, env := range p.tenant.Spec.Environments {
		err := p.clients.ServiceUsage.EnableAPIs(ctx, p.newState.Projects[env].ProjectID, p.tenant.Spec.AllowedApis)
		if err != nil {
			return err
		}
	}
	p.newState.AllowedApis = p.tenant.Spec.AllowedApis
	return nil
}
