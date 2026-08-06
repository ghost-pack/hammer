package project

import (
	"context"
	"fmt"
	"log/slog"
	"strings"

	serviceusage "cloud.google.com/go/serviceusage/apiv1"
	serviceusagepb "cloud.google.com/go/serviceusage/apiv1/serviceusagepb"
	"github.com/ghost-pack/hammer/internal/observability/tracing"
	gax "github.com/googleapis/gax-go/v2"
	"go.opentelemetry.io/otel/attribute"
	otelCodes "go.opentelemetry.io/otel/codes"
	"go.opentelemetry.io/otel/trace"
	"google.golang.org/api/option"
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

func NewServiceUsageClient(ctx context.Context, opts ...option.ClientOption) (*ServiceUsageClientImpl, error) {
	client, err := serviceusage.NewClient(ctx, opts...)
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
	ctx, span := tracing.Tracer("enable apis").Start(ctx, "enable apis",
		trace.WithAttributes(
			attribute.String("project ID", projectID),
			attribute.String("apis", strings.Join(apis, ","))))
	defer span.End()

	op, err := c.client.BatchEnableServices(ctx, &serviceusagepb.BatchEnableServicesRequest{
		Parent:     "projects/" + projectID,
		ServiceIds: apis,
	})
	if err != nil {
		span.RecordError(err)
		span.SetStatus(otelCodes.Error, err.Error())
		return fmt.Errorf("enabling APIs on %s: %w", projectID, err)
	}
	if _, err := op.Wait(ctx); err != nil {
		span.RecordError(err)
		span.SetStatus(otelCodes.Error, err.Error())
		return fmt.Errorf("waiting for API enablement on %s: %w", projectID, err)
	}
	slog.InfoContext(ctx, "APIs enabled", "projectID", projectID, "apis", apis)
	span.SetStatus(otelCodes.Ok, "APIs enabled")
	return nil
}
