package gcp

import "time"

type TenantState struct {
	Name           string                        `json:"name"`
	AppliedAt      time.Time                     `json:"appliedAt"`
	Parent         string                        `json:"parent"`
	BillingAccount string                        `json:"billingAccount"`
	Projects       map[string]ProvisionedProject `json:"projects"`
	AllowedApis    []string                      `json:"allowedAPIs"`
	OrgPolicies    []string                      `json:"orgPolicies"`
}

type ProvisionedProject struct {
	ProjectID       string                               `json:"projectId"`
	ProjectNumber   string                               `json:"projectNumber"`
	ServiceAccounts map[string]ProvisionedServiceAccount `json:"serviceAccounts"`
}

type ProvisionedServiceAccount struct {
	Email        string   `json:"email"`
	ProjectRoles []string `json:"projectRoles,omitempty"`
	OrgRoles     []string `json:"orgRoles,omitempty"`
}
