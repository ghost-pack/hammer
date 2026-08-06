package gcp

import (
	"context"
	"fmt"
	"log/slog"

	"github.com/ghost-pack/hammer/internal/observability/tracing"
	"github.com/ghost-pack/hammer/internal/provisioner"
	"github.com/ghost-pack/hammer/internal/tenant"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/trace"
)

func init() {
	provisioner.Register("Tenant", New)
}

func New(t *tenant.Tenant, clients *provisioner.DependencyClients) (provisioner.Provisioner, error) {
	return &Provisioner{
		tenant:           t,
		clients:          clients,
		registryBucket:   "hammer-registry",
		platformProject:  "hammer-central-prod",
		defaultRegion:    "us-central1",
		newState:         &TenantState{},
		lastAppliedState: &TenantState{},
	}, nil
}

type Provisioner struct {
	// config — set at construction, never changes
	tenant          *tenant.Tenant
	clients         *provisioner.DependencyClients
	registryBucket  string
	platformProject string
	defaultRegion   string

	// loaded from GCS at start of run — nil if first run
	lastAppliedState *TenantState

	// computed during reconcile phase, consumed by later phases
	apisToAdd    []string
	apisToRemove []string

	// built up during reconcile, applied as we go along
	newState *TenantState
}

func (p *Provisioner) Apply(ctx context.Context) error {
	ctx, span := tracing.Tracer(fmt.Sprintf("%s tenant", p.tenant.Metadata.Name)).Start(ctx, fmt.Sprintf("%s tenant", p.tenant.Metadata.Name),
		trace.WithAttributes(
			attribute.String("cmd", fmt.Sprintf("%s tenant", p.tenant.Metadata.Name))))
	defer span.End()

	phases := []phase{
		{"Ensure GCS Bucket Exists", p.ensureBucketExists},
		{"Load Last Applied State", p.loadLastAppliedState},
		{"Reconcile", p.reconcile},
		{"Ensure Folder Exists", p.ensureFolderExists},
		{"Ensure Projects Exist", p.ensureProjectsExists},
		{"Link Billing Accounts", p.linkBillingAccounts},
		{"Ensure Service Accounts", p.ensureServiceAccounts},
		{"Apply Role Changes", p.applyRoleChanges},
		{"Enable APIs", p.enableApis},
		{"Apply Org Policies", p.applyOrgPolicies},
		{"Write State to Registry", p.writeStateToRegistry},
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
