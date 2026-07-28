package tenantpipeline

import (
	"context"
	"fmt"

	"github.com/ghost-pack/hammer/internal/gcp"
	"github.com/ghost-pack/hammer/internal/gcp/project"
	"github.com/ghost-pack/hammer/internal/tenant"
)

type Provisioner interface {
	Apply(ctx context.Context) error
}

type DependencyClients struct {
	ResourceManager project.ResourceManagerClient
	ServiceUsage    project.ServiceUsageClient
	OrgPolicy       project.OrgPolicyClient
	IAM             project.IAMClient
	CloudStorage    gcp.CloudStorageClient
	Billing         project.BillingClient
}

type Factory func(*tenant.Tenant, *DependencyClients) (Provisioner, error)

var registry = map[string]Factory{}

func Register(kind string, f Factory) {
	if kind == "" {
		panic("tenantpipeline.Register: kind is empty")
	}
	if f == nil {
		panic("tenantpipeline.Register: factory is nil")
	}
	if _, exists := registry[kind]; exists {
		panic("tenantpipeline.Register: kind already registered: " + kind)
	}
	registry[kind] = f
}

func For(t *tenant.Tenant, deps *DependencyClients) (Provisioner, error) {
	f, ok := registry[t.Kind]
	if !ok {
		return nil, fmt.Errorf("no provisioner registered for kind %s", t.Kind)
	}
	return f(t, deps)
}
