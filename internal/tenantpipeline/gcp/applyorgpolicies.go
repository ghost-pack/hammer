package gcp

import (
	"context"
)

var constraints = []string{
	"constraints/compute.requireOsLogin",
	"constraints/iam.disableServiceAccountKeyCreation",
}

func (p *Provisioner) applyOrgPolicies(ctx context.Context) error {
	for _, env := range p.tenant.Spec.Environments {
		for _, constraint := range constraints {
			err := p.clients.OrgPolicy.EnforcePolicy(ctx, p.newState.Projects[env].ProjectID, constraint)
			if err != nil {
				return err
			}
		}
	}
	p.newState.OrgPolicies = constraints
	return nil
}
