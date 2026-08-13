package gcp

import (
	"context"
	"fmt"
)

func (p *Provisioner) applyRoleChanges(ctx context.Context) error {
	for _, env := range p.tenant.Spec.Environments {
		projectID := p.newState.Projects[env].ProjectID

		err := applySaPipelineRoles(ctx, p, env, projectID)
		if err != nil {
			return err
		}

		err = applyTenantDefinedServiceAccountRoles(ctx, p, env, projectID)
		if err != nil {
			return err
		}
	}

	return nil

}

func applyTenantDefinedServiceAccountRoles(ctx context.Context, p *Provisioner, env string, projectID string) error {
	// tenant-defined service accounts
	for _, saSpec := range p.tenant.Spec.ServiceAccounts {
		sa := p.newState.Projects[env].ServiceAccounts[saSpec.Name]

		var lastProjectRoles, lastOrgRoles []string
		if p.lastAppliedState != nil {
			if lastSA, ok := p.lastAppliedState.Projects[env].ServiceAccounts[saSpec.Name]; ok {
				lastProjectRoles = lastSA.ProjectRoles
				lastOrgRoles = lastSA.OrgRoles
			}
		}

		projectRolesToAdd, projectRolesToRemove := diffStringSlices(lastProjectRoles, saSpec.Roles.Project)
		if len(projectRolesToAdd) > 0 {
			if err := p.clients.IAM.BindProjectRoles(ctx, projectID, sa.Email, projectRolesToAdd); err != nil {
				return fmt.Errorf("binding %s project roles for %s: %w", saSpec.Name, env, err)
			}
		}
		if len(projectRolesToRemove) > 0 {
			if err := p.clients.IAM.UnbindProjectRoles(ctx, projectID, sa.Email, projectRolesToRemove); err != nil {
				return fmt.Errorf("unbinding %s project roles for %s: %w", saSpec.Name, env, err)
			}
		}

		orgRolesToAdd, orgRolesToRemove := diffStringSlices(lastOrgRoles, saSpec.Roles.Organization)
		if len(orgRolesToAdd) > 0 {
			if err := p.clients.IAM.BindOrgRoles(ctx, p.tenant.Spec.ParentFolder, sa.Email, orgRolesToAdd); err != nil {
				return fmt.Errorf("binding %s org roles for %s: %w", saSpec.Name, env, err)
			}
		}
		if len(orgRolesToRemove) > 0 {
			if err := p.clients.IAM.UnbindOrgRoles(ctx, p.tenant.Spec.ParentFolder, sa.Email, orgRolesToRemove); err != nil {
				return fmt.Errorf("unbinding %s org roles for %s: %w", saSpec.Name, env, err)
			}
		}

		project := p.newState.Projects[env]
		sa.ProjectRoles = saSpec.Roles.Project
		sa.OrgRoles = saSpec.Roles.Organization
		project.ServiceAccounts[saSpec.Name] = sa
		p.newState.Projects[env] = project
	}
	return nil
}

func applySaPipelineRoles(ctx context.Context, p *Provisioner, env string, projectID string) error {
	pipelineSAEmail := p.newState.Projects[env].ServiceAccounts["sa-pipeline"].Email

	pipelineRolesToAdd, err := convertApiToRole(p.apisToAdd)
	if err != nil {
		return fmt.Errorf("converting apis to add for sa-pipeline: %w", err)
	}
	pipelineRolesToAdd = append(pipelineRolesToAdd, alwaysOnRoles...)

	pipelineRolesToRemove, err := convertApiToRole(p.apisToRemove)
	if err != nil {
		return fmt.Errorf("converting apis to remove for sa-pipeline: %w", err)
	}

	if len(pipelineRolesToAdd) > 0 {
		if err := p.clients.IAM.BindProjectRoles(ctx, projectID, pipelineSAEmail, pipelineRolesToAdd); err != nil {
			return fmt.Errorf("binding sa-pipeline roles for %s: %w", env, err)
		}
	}
	if len(pipelineRolesToRemove) > 0 {
		if err := p.clients.IAM.UnbindProjectRoles(ctx, projectID, pipelineSAEmail, pipelineRolesToRemove); err != nil {
			return fmt.Errorf("unbinding sa-pipeline roles for %s: %w", env, err)
		}
	}

	err = setupSaOamOnTenantProject(ctx, p, env, projectID, pipelineSAEmail)
	if err != nil {
		return err
	}

	// compute final sa-pipeline roles for newState
	allPipelineRoles, err := convertApiToRole(p.tenant.Spec.AllowedApis)
	if err != nil {
		return fmt.Errorf("computing final sa-pipeline roles: %w", err)
	}
	allPipelineRoles = append(allPipelineRoles, alwaysOnRoles...)

	project := p.newState.Projects[env]
	pipelineSA := project.ServiceAccounts["sa-pipeline"]
	pipelineSA.ProjectRoles = allPipelineRoles
	project.ServiceAccounts["sa-pipeline"] = pipelineSA
	p.newState.Projects[env] = project
	return nil
}

func setupSaOamOnTenantProject(ctx context.Context, p *Provisioner, env string, projectID string, pipelineSAEmail string) error {
	// Allowing sa-oam to impersonate sa-pipeline
	if err := p.clients.IAM.AllowImpersonation(ctx, projectID, pipelineSAEmail, p.platformOAMServiceAccount); err != nil {
		return fmt.Errorf("granting sa-oam impersonation of sa-pipeline for %s: %w", env, err)
	}

	// Giving sa-oam the ability to create service accounts (for cloud run) on tenant projects,
	// as well as view cloud storage (for opentofu plan)
	if err := p.clients.IAM.BindProjectRoles(
		ctx,
		projectID,
		pipelineSAEmail,
		[]string{
			"roles/storage.objectViewer",
			"roles/iam.serviceAccountCreator",
			"roles/iam.serviceAccountAdmin",
		},
	); err != nil {
		return fmt.Errorf("binding sa-oam roles for %s: %w", env, err)
	}
	return nil
}

func diffStringSlices(old, new []string) (toAdd, toRemove []string) {
	oldSet := make(map[string]bool, len(old))
	for _, s := range old {
		oldSet[s] = true
	}
	newSet := make(map[string]bool, len(new))
	for _, s := range new {
		newSet[s] = true
		if !oldSet[s] {
			toAdd = append(toAdd, s)
		}
	}
	for _, s := range old {
		if !newSet[s] {
			toRemove = append(toRemove, s)
		}
	}
	return
}

func convertApiToRole(apis []string) ([]string, error) {
	var roles []string
	for _, api := range apis {
		rolesForApi, ok := apiToRoles[api]
		if !ok {
			return nil, fmt.Errorf("api %s is not supported", api)
		}
		if rolesForApi != nil {
			roles = append(roles, rolesForApi...)
		}
	}
	return roles, nil
}

var alwaysOnRoles = []string{
	"roles/iam.serviceAccountAdmin",
	"roles/iam.serviceAccountUser",
	"roles/resourcemanager.projectIamAdmin",
	"roles/serviceusage.serviceUsageConsumer",
}

// apiToRoles maps an enabled API to the roles terraform-sa needs to manage it
var apiToRoles = map[string][]string{
	"run.googleapis.com": {
		"roles/run.admin",
	},
	"cloudbuild.googleapis.com": {
		"roles/cloudbuild.builds.editor",
		"roles/cloudbuild.integrations.owner",
	},
	"artifactregistry.googleapis.com": {
		"roles/artifactregistry.admin",
	},
	"sqladmin.googleapis.com": {
		"roles/cloudsql.admin",
	},
	"secretmanager.googleapis.com": {
		"roles/secretmanager.admin",
	},
	"storage.googleapis.com": {
		"roles/storage.admin",
	},
	"logging.googleapis.com": {
		"roles/logging.admin",
	},
	"monitoring.googleapis.com": {
		"roles/monitoring.admin",
	},
	"cloudtrace.googleapis.com": {
		"roles/cloudtrace.agent",
	},
	"cloudscheduler.googleapis.com": {
		"roles/cloudscheduler.admin",
	},
	"cloudtasks.googleapis.com": {
		"roles/cloudtasks.admin",
	},
	"pubsub.googleapis.com": {
		"roles/pubsub.admin",
	},
	"firestore.googleapis.com": {
		"roles/datastore.owner",
	},
	"redis.googleapis.com": {
		"roles/redis.admin",
	},
	"vpcaccess.googleapis.com": {
		"roles/vpcaccess.admin",
	},
	"dns.googleapis.com": {
		"roles/dns.admin",
	},
	"bigquery.googleapis.com": {
		"roles/bigquery.admin",
	},
	"cloudresourcemanager.googleapis.com": {},
	"cloudbilling.googleapis.com":         {},
	"orgpolicy.googleapis.com":            {},
	"serviceusage.googleapis.com":         {},
}
