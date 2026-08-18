package gcp

import (
	"context"
	"fmt"
)

const platformTenant = "hammer-central"

func (p *Provisioner) ensureServiceAccounts(ctx context.Context) error {
	if p.tenant.Metadata.Name != platformTenant {
		for _, sa := range p.tenant.Spec.ServiceAccounts {
			if len(sa.Roles.Organization) > 0 {
				return fmt.Errorf(
					"tenant %q is not permitted to have org-level roles on service account %q — only %s can have org roles",
					p.tenant.Metadata.Name, sa.Name, platformTenant,
				)
			}
		}
	}

	for _, env := range p.tenant.Spec.Environments {
		serviceAccounts := map[string]ProvisionedServiceAccount{}

		for _, serviceAccount := range p.tenant.Spec.ServiceAccounts {
			email, err := p.clients.IAM.EnsureServiceAccountExists(ctx, p.newState.Projects[env].ProjectID, serviceAccount.Name, serviceAccount.Name)
			if err != nil {
				return err
			}
			serviceAccounts[serviceAccount.Name] = ProvisionedServiceAccount{
				Email: email,
			}
		}

		p.newState.Projects[env] = ProvisionedProject{
			ProjectID:       p.tenant.Metadata.Name + "-" + env,
			ProjectNumber:   p.newState.Projects[env].ProjectNumber,
			ServiceAccounts: serviceAccounts,
		}
	}
	return nil
}
