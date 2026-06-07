package gcp

import (
	"context"
	"fmt"

	artifactregistry "cloud.google.com/go/artifactregistry/apiv1"
	"cloud.google.com/go/artifactregistry/apiv1/artifactregistrypb"
	"github.com/ghost-pack/hammer/internal/observability/tracing"
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

type GarClientImpl struct {
	client *artifactregistry.Client
}

func NewGarClient(ctx context.Context, opts ...option.ClientOption) (GarClient, error) {
	client, err := artifactregistry.NewClient(ctx, opts...)
	if err != nil {
		return nil, fmt.Errorf("creating artifact registry client: %w", err)
	}
	return &GarClientImpl{client: client}, nil
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
		// Defensive coding in case another build is creating the repository
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
