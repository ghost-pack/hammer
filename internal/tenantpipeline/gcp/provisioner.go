package gcp

import (
	"context"
	"fmt"
	"log/slog"

	"github.com/ghost-pack/hammer/internal/observability/tracing"
	"github.com/ghost-pack/hammer/internal/tenant"
	"github.com/ghost-pack/hammer/internal/tenantpipeline"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/trace"
)

func init() {
	tenantpipeline.Register("Tenant", New)
}

func New(t *tenant.Tenant, clients tenantpipeline.DependencyClients) (tenantpipeline.Provisioner, error) {
	return &Provisioner{
		tenant:  t,
		clients: clients,
	}, nil
}

type Provisioner struct {
	tenant         *tenant.Tenant
	clients        tenantpipeline.DependencyClients
	registryBucket string
}

func (p *Provisioner) Apply(ctx context.Context) error {
	// create folder, projects, enable APIs, org policies, SAs...
	// Phase 1: Create GCP Bucket if it doesn't exist.
	// Phase 2: Grab latest state from bucket (if it exists) and reconcile it.
	// Phase 3: Create project, I guess.
	// Phase 4: Link billing account and parent folder.
	// Phase 5: Apply org policies to project.
	// Phase 6: Create service accounts.
	// Phase 7: Write OAM file to bucket.
	// If any steps fail during this,
	ctx, span := tracing.Tracer(fmt.Sprintf("%s tenant", p.tenant.Metadata.Name)).Start(ctx, fmt.Sprintf("%s tenant", p.tenant.Metadata.Name),
		trace.WithAttributes(
			attribute.String("cmd", fmt.Sprintf("%s tenant", p.tenant.Metadata.Name))))
	defer span.End()

	phases := []phase{
		{"Ensure GCP Bucket Exists", p.ensureBucketExists},
		//{"Ensure Project Exists", p.build},
		//{"Link billing account", p.build},
		//{"Ensure Org Policies Exists", p.build},
		//{"Ensure Service Accounts Exist", p.build},
		//{"Write OAM file", p.build},
	}

	for _, ph := range phases {
		slog.InfoContext(ctx, "phase start", "phase", ph.name)
		if err := ph.run(ctx); err != nil {
			slog.ErrorContext(ctx, "phase error", "phase", ph.name, "error", err)
			return fmt.Errorf("phase %s error: %w", ph.name, err)
		}
	}
	return nil
}

type phase struct {
	name string
	run  func(context.Context) error
}
