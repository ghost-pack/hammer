package gcp

import (
	"context"
)

var constraints = []string{
	"compute.requireOsLogin",
	"iam.managed.disableServiceAccountKeyCreation",
}

func (p *Provisioner) applyOrgPolicies(ctx context.Context) error {
	for _, env := range p.tenant.Spec.Environments {
		for _, constraint := range constraints {
			err := p.clients.OrgPolicy.EnforcePolicy(ctx, "projects/"+p.newState.Projects[env].ProjectNumber, constraint)
			if err != nil {
				return err
			}
		}
	}
	p.newState.OrgPolicies = constraints
	return nil
}
