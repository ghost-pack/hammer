package opentofu

import (
	"context"
	"fmt"
	"log/slog"
	"os"
	"path/filepath"

	"github.com/ghost-pack/hammer/internal/observability/tracing"
	"github.com/ghost-pack/hammer/internal/tenant"
	"go.opentelemetry.io/otel/codes"
	"gopkg.in/yaml.v3"
)

func (p *Pipeline) getSupportedEnvironments(ctx context.Context) ([]string, error) {
	ctx, span := tracing.Tracer("opentofu").Start(ctx, "get supported opentofu environments")
	defer span.End()
	// Step 1: Get tenant environments from registry
	tenantBytes, err := p.cloudStorageClient.GetObject(
		ctx,
		"hammer-registry",
		fmt.Sprintf("tenants/%s/spec.yaml", p.app.Metadata.Name),
	)
	if err != nil {
		span.RecordError(err)
		span.SetStatus(codes.Error, err.Error())
		return nil, fmt.Errorf("getting tenant spec: %w", err)
	}

	var currentTenant tenant.Tenant // No need for a pointer here
	if err := yaml.Unmarshal(tenantBytes, &currentTenant); err != nil {
		span.RecordError(err)
		span.SetStatus(codes.Error, err.Error())
		return nil, fmt.Errorf("unmarshalling tenant state: %w", err)
	}

	tenantEnvs := currentTenant.Spec.Environments

	// Step 2: Get terraform environments by scanning the filesystem
	var props properties
	if props, err = parseOpenTofuPath(p); err != nil {
		span.RecordError(err)
		span.SetStatus(codes.Error, err.Error())
		return nil, fmt.Errorf("parsing opentofu path: %w", err)
	}

	terraformEnvs, err := scanTerraformEnvironments(props.Path)
	if err != nil {
		span.RecordError(err)
		span.SetStatus(codes.Error, err.Error())
		return nil, fmt.Errorf("scanning terraform environments: %w", err)
	}

	// Step 3: Find intersection
	supportedEnvs := intersectEnvironments(terraformEnvs, tenantEnvs)

	slog.InfoContext(ctx, "calculated supported environments",
		"tenant_envs", tenantEnvs,
		"terraform_envs", terraformEnvs,
		"supported_envs", supportedEnvs)

	if len(supportedEnvs) == 0 {
		span.SetStatus(codes.Ok, "no supported environments found")
		return []string{}, nil
	}

	span.SetStatus(codes.Ok, fmt.Sprintf("found %d supported environments", len(supportedEnvs)))
	return supportedEnvs, nil
}

func scanTerraformEnvironments(opentofuPath string) ([]string, error) {
	environmentsDir := filepath.Join(opentofuPath, "environments")

	entries, err := os.ReadDir(environmentsDir)
	if err != nil {
		// If the directory doesn't exist, there are no terraform environments.
		if os.IsNotExist(err) {
			return []string{}, nil
		}
		return nil, fmt.Errorf("reading environments directory: %w", err)
	}

	var envs []string
	for _, entry := range entries {
		if entry.IsDir() {
			envs = append(envs, entry.Name())
		}
	}

	return envs, nil
}

// intersectEnvironments returns environments that exist in both lists.
func intersectEnvironments(terraformEnvs, tenantEnvs []string) []string {
	tenantSet := make(map[string]struct{}, len(tenantEnvs))
	for _, env := range tenantEnvs {
		tenantSet[env] = struct{}{}
	}

	var supported []string
	for _, env := range terraformEnvs {
		if _, exists := tenantSet[env]; exists {
			supported = append(supported, env)
		}
	}

	return supported
}
