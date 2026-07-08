package tenant

type Tenant struct {
	APIVersion string   `yaml:"apiVersion"`
	Kind       string   `yaml:"kind"`
	Metadata   Metadata `yaml:"metadata"`
	Spec       Spec     `yaml:"spec"`
}

type Metadata struct {
	Name        string            `yaml:"name"`
	Annotations map[string]string `yaml:"annotations,omitempty"`
}

type Spec struct {
	BillingAccount string   `yaml:"billingAccount"`
	ParentFolder   string   `yaml:"parentFolder"`
	AllowedApis    []string `yaml:"allowedAPIs"`
	Environments   []string `yaml:"environments"`
	OrgPolicies    []string `yaml:"orgPolicies"`
}
