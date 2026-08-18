package project

import (
	"context"
	"fmt"

	artifactregistry "cloud.google.com/go/artifactregistry/apiv1"
	"cloud.google.com/go/artifactregistry/apiv1/artifactregistrypb"
	"cloud.google.com/go/iam/apiv1/iampb"
	"github.com/ghost-pack/hammer/internal/observability/tracing"
	"github.com/googleapis/gax-go/v2"
	"go.opentelemetry.io/otel/attribute"
	otelCodes "go.opentelemetry.io/otel/codes"
	"go.opentelemetry.io/otel/trace"
	"google.golang.org/api/option"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

type GarClient interface {
	EnsureRepository(ctx context.Context, projectID, location, repoID string) error
	Close() error
}

type garAPI interface {
	GetRepository(ctx context.Context, req *artifactregistrypb.GetRepositoryRequest, opts ...gax.CallOption) (*artifactregistrypb.Repository, error)
	CreateRepository(ctx context.Context, req *artifactregistrypb.CreateRepositoryRequest, opts ...gax.CallOption) (*artifactregistry.CreateRepositoryOperation, error)
	Close() error
}

type garAdapter struct {
	client *artifactregistry.Client
}

func (a *garAdapter) GetRepository(ctx context.Context, req *artifactregistrypb.GetRepositoryRequest, opts ...gax.CallOption) (*artifactregistrypb.Repository, error) {
	return a.client.GetRepository(ctx, req, opts...)
}

func (a *garAdapter) CreateRepository(ctx context.Context, req *artifactregistrypb.CreateRepositoryRequest, opts ...gax.CallOption) (*artifactregistry.CreateRepositoryOperation, error) {
	return a.client.CreateRepository(ctx, req, opts...)
}

func (a *garAdapter) GetIamPolicy(ctx context.Context, req *iampb.GetIamPolicyRequest, opts ...gax.CallOption) (*iampb.Policy, error) {
	return a.client.GetIamPolicy(ctx, req, opts...)
}

func (a *garAdapter) SetIamPolicy(ctx context.Context, req *iampb.SetIamPolicyRequest, opts ...gax.CallOption) (*iampb.Policy, error) {
	return a.client.SetIamPolicy(ctx, req, opts...)
}

func (a *garAdapter) Close() error {
	return a.client.Close()
}

type GarClientImpl struct {
	client garAPI
}

func NewGarClient(ctx context.Context, opts ...option.ClientOption) (*GarClientImpl, error) {
	client, err := artifactregistry.NewClient(ctx, opts...)
	if err != nil {
		return nil, fmt.Errorf("creating artifact registry client: %w", err)
	}
	return &GarClientImpl{client: &garAdapter{client}}, nil
}

func newGarClientWithAPI(api garAPI) *GarClientImpl {
	return &GarClientImpl{client: api}
}

func (g *GarClientImpl) Close() error {
	return g.client.Close()
}

func (g *GarClientImpl) EnsureRepository(ctx context.Context, projectID, location, repoID string) error {
	ctx, span := tracing.Tracer("gcloud artifacts repositories").Start(ctx, "gcloud artifacts repositories",
		trace.WithAttributes(
			attribute.String("cmd", "gcloud"),
			attribute.StringSlice("args", []string{"artifacts", "repositories", "create"})))
	defer span.End()
	parent := fmt.Sprintf("projects/%s/locations/%s", projectID, location)
	name := fmt.Sprintf("%s/repositories/%s", parent, repoID)

	req := &artifactregistrypb.GetRepositoryRequest{Name: name}
	_, err := g.client.GetRepository(ctx, req)
	if err == nil {
		// Already exists.
		span.SetStatus(otelCodes.Ok, "")
		return nil
	}

	if status.Code(err) != codes.NotFound {
		repositoryNotFoundError := fmt.Errorf("checking repository %q: %w", name, err)
		span.RecordError(repositoryNotFoundError)
		span.SetStatus(otelCodes.Error, repositoryNotFoundError.Error())
		return repositoryNotFoundError
	}

	createReq := &artifactregistrypb.CreateRepositoryRequest{
		Parent:       parent,
		RepositoryId: repoID,
		Repository: &artifactregistrypb.Repository{
			Format: artifactregistrypb.Repository_DOCKER,
		},
	}

	_, err = g.client.CreateRepository(ctx, createReq)
	if err != nil {
		if status.Code(err) == codes.AlreadyExists {
			span.SetStatus(otelCodes.Ok, "")
			return nil
		}
		repositoryCreationFailedError := fmt.Errorf("creating repository %q: %w", name, err)
		span.RecordError(repositoryCreationFailedError)
		span.SetStatus(otelCodes.Error, repositoryCreationFailedError.Error())
		return repositoryCreationFailedError
	}
	span.SetStatus(otelCodes.Ok, "")
	return nil
}
