package gcp

import "time"

type TenantState struct {
	Name           string                        `json:"name"`
	AppliedAt      time.Time                     `json:"appliedAt"`
	Parent         string                        `json:"parent"`
	BillingAccount string                        `json:"billingAccount"`
	Projects       map[string]ProvisionedProject `json:"projects"` // env → project
	AllowedApis    []string                      `json:"allowedAPIs"`
	OrgPolicies    []string                      `json:"orgPolicies"`
}

type ProvisionedProject struct {
	ProjectID       string            `json:"projectId"`
	ServiceAccounts map[string]string `json:"serviceAccounts"` // name → email
}
