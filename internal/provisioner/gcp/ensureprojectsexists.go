package gcp

import (
	"context"
)

func (p *Provisioner) ensureProjectsExists(ctx context.Context) error {
	p.newState.Projects = map[string]ProvisionedProject{}

	for _, env := range p.tenant.Spec.Environments {
		projectNumber, err := p.clients.ResourceManager.EnsureProjectExists(ctx, p.tenant.Metadata.Name+"-"+env, p.tenant.Metadata.Name+"-"+env, p.newState.Parent)
		if err != nil {
			return err
		}
		p.newState.Projects[env] = ProvisionedProject{
			ProjectID:     p.tenant.Metadata.Name + "-" + env,
			ProjectNumber: projectNumber,
		}
	}
	return nil
}
