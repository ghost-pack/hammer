package gcp

import (
	"context"
)

func (p *Provisioner) ensureServiceAccounts(ctx context.Context) error {
	for _, env := range p.tenant.Spec.Environments {
		email, err := p.clients.IAM.EnsureServiceAccountExists(ctx, p.newState.Projects[env].ProjectID, "sa-pipeline", "sa-pipeline")
		if err != nil {
			return err
		}
		serviceAccounts := map[string]string{
			"sa-pipeline": email,
		}

		p.newState.Projects[env] = ProvisionedProject{
			ProjectID:       p.tenant.Metadata.Name + env,
			ServiceAccounts: serviceAccounts,
		}
	}
	return nil
}
