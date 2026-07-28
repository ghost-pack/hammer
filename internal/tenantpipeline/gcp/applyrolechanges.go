package gcp

import (
	"context"
	"fmt"
)

func (p *Provisioner) applyRoleChanges(ctx context.Context) error {
	rolesToAdd, err := convertApiToRole(p.apisToAdd)
	rolesToAdd = append(rolesToAdd, alwaysOnRoles...)
	if err != nil {
		return err
	}
	rolesToRemove, err := convertApiToRole(p.apisToRemove)
	if err != nil {
		return err
	}
	for _, env := range p.tenant.Spec.Environments {
		projectID := p.newState.Projects[env].ProjectID
		pipelineSAEmail := fmt.Sprintf("sa-pipeline@%s.iam.gserviceaccount.com", projectID)

		if len(rolesToAdd) > 0 {
			if err := p.clients.IAM.BindProjectRoles(ctx, projectID, pipelineSAEmail, rolesToAdd); err != nil {
				return fmt.Errorf("binding roles for %s: %w", env, err)
			}
		}

		if len(rolesToRemove) > 0 {
			if err := p.clients.IAM.UnbindProjectRoles(ctx, projectID, pipelineSAEmail, rolesToRemove); err != nil {
				return fmt.Errorf("unbinding roles for %s: %w", env, err)
			}
		}
	}

	return nil
}

func convertApiToRole(apis []string) ([]string, error) {
	var roles []string
	for _, api := range apis {
		rolesForApi, ok := apiToRoles[api]
		if !ok {
			return nil, fmt.Errorf("api %s is not supported", api)
		}
		roles = append(roles, rolesForApi...)
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
}
