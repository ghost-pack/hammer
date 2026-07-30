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
	BillingAccount  string               `yaml:"billingAccount"`
	ParentFolder    string               `yaml:"parentFolder"`
	AllowedApis     []string             `yaml:"allowedAPIs"`
	Environments    []string             `yaml:"environments"`
	ServiceAccounts []ServiceAccountSpec `yaml:"serviceAccounts,omitempty"`
}

type ServiceAccountSpec struct {
	Name        string        `yaml:"name"`
	Description string        `yaml:"description"`
	Roles       SARoleBinding `yaml:"roles"`
}

type SARoleBinding struct {
	Project      []string `yaml:"project,omitempty"`
	Organization []string `yaml:"organization,omitempty"`
}
