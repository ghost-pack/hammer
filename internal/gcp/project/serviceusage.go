package project

import (
	"context"
	"fmt"
	"log/slog"

	serviceusage "cloud.google.com/go/serviceusage/apiv1"
	serviceusagepb "cloud.google.com/go/serviceusage/apiv1/serviceusagepb"
	gax "github.com/googleapis/gax-go/v2"
)

// batchEnableOperation wraps the concrete operation so it can be mocked
type batchEnableOperation interface {
	Wait(ctx context.Context, opts ...gax.CallOption) (*serviceusagepb.BatchEnableServicesResponse, error)
}

// serviceUsageAPI is the minimal interface we need from the GCP client
type serviceUsageAPI interface {
	BatchEnableServices(ctx context.Context, req *serviceusagepb.BatchEnableServicesRequest, opts ...gax.CallOption) (batchEnableOperation, error)
	Close() error
}

type serviceUsageAdapter struct {
	client *serviceusage.Client
}

func (a *serviceUsageAdapter) BatchEnableServices(ctx context.Context, req *serviceusagepb.BatchEnableServicesRequest, opts ...gax.CallOption) (batchEnableOperation, error) {
	return a.client.BatchEnableServices(ctx, req, opts...)
}

func (a *serviceUsageAdapter) Close() error {
	return a.client.Close()
}

type ServiceUsageClient interface {
	EnableAPIs(ctx context.Context, projectID string, apis []string) error
	Close() error
}

type ServiceUsageClientImpl struct {
	client serviceUsageAPI
}

func NewServiceUsageClient(ctx context.Context) (*ServiceUsageClientImpl, error) {
	client, err := serviceusage.NewClient(ctx)
	if err != nil {
		return nil, fmt.Errorf("creating service usage client: %w", err)
	}
	return &ServiceUsageClientImpl{
		client: &serviceUsageAdapter{client: client},
	}, nil
}

func newServiceUsageClientWithAPI(api serviceUsageAPI) *ServiceUsageClientImpl {
	return &ServiceUsageClientImpl{client: api}
}

func (c *ServiceUsageClientImpl) Close() error {
	return c.client.Close()
}

func (c *ServiceUsageClientImpl) EnableAPIs(ctx context.Context, projectID string, apis []string) error {
	serviceNames := make([]string, len(apis))
	for i, api := range apis {
		serviceNames[i] = fmt.Sprintf("projects/%s/services/%s", projectID, api)
	}

	op, err := c.client.BatchEnableServices(ctx, &serviceusagepb.BatchEnableServicesRequest{
		Parent:     "projects/" + projectID,
		ServiceIds: serviceNames,
	})
	if err != nil {
		return fmt.Errorf("enabling APIs on %s: %w", projectID, err)
	}
	if _, err := op.Wait(ctx); err != nil {
		return fmt.Errorf("waiting for API enablement on %s: %w", projectID, err)
	}
	slog.InfoContext(ctx, "APIs enabled", "projectID", projectID, "apis", apis)
	return nil
}
