package gcp

import "context"

func (p *Provisioner) linkBillingAccounts(ctx context.Context) error {
	p.newState.BillingAccount = p.tenant.Spec.BillingAccount

	for _, env := range p.tenant.Spec.Environments {
		err := p.clients.Billing.LinkBillingAccount(ctx, p.newState.Projects[env].ProjectID, p.newState.BillingAccount)
		if err != nil {
			return err
		}
	}
	return nil
}
