package project

import (
	"context"
	"errors"
	"fmt"
	"log/slog"

	resourcemanager "cloud.google.com/go/resourcemanager/apiv3"
	resourcemanagerpb "cloud.google.com/go/resourcemanager/apiv3/resourcemanagerpb"
	"github.com/ghost-pack/hammer/internal/observability/tracing"
	"github.com/googleapis/gax-go/v2"
	"go.opentelemetry.io/otel/attribute"
	otelCodes "go.opentelemetry.io/otel/codes"
	"go.opentelemetry.io/otel/trace"
	"google.golang.org/api/iterator"
	"google.golang.org/api/option"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

// interfaces for the operation and iterator so they can be mocked
type createFolderOperation interface {
	Wait(ctx context.Context, opts ...gax.CallOption) (*resourcemanagerpb.Folder, error)
}

type folderIterator interface {
	Next() (*resourcemanagerpb.Folder, error)
}

type foldersAPI interface {
	CreateFolder(ctx context.Context, req *resourcemanagerpb.CreateFolderRequest, opts ...gax.CallOption) (createFolderOperation, error)
	ListFolders(ctx context.Context, req *resourcemanagerpb.ListFoldersRequest, opts ...gax.CallOption) folderIterator
	Close() error
}

// adapter wraps the real GCP client — real types satisfy the interfaces implicitly
type foldersAdapter struct {
	client *resourcemanager.FoldersClient
}

func (a *foldersAdapter) CreateFolder(ctx context.Context, req *resourcemanagerpb.CreateFolderRequest, opts ...gax.CallOption) (createFolderOperation, error) {
	return a.client.CreateFolder(ctx, req, opts...)
}

func (a *foldersAdapter) ListFolders(ctx context.Context, req *resourcemanagerpb.ListFoldersRequest, opts ...gax.CallOption) folderIterator {
	return a.client.ListFolders(ctx, req, opts...)
}

func (a *foldersAdapter) Close() error {
	return a.client.Close()
}

type ResourceManagerClient interface {
	EnsureFolderExists(ctx context.Context, displayName, parent string) (string, error)
	EnsureProjectExists(ctx context.Context, projectID, displayName, parent string) error
	Close() error
}

type ResourceManagerClientImpl struct {
	folders  foldersAPI
	projects projectsAPI
}

func NewResourceManagerClient(ctx context.Context, opts ...option.ClientOption) (*ResourceManagerClientImpl, error) {
	folders, err := resourcemanager.NewFoldersClient(ctx, opts...)
	if err != nil {
		return nil, fmt.Errorf("creating folders client: %w", err)
	}
	projects, err := resourcemanager.NewProjectsClient(ctx, opts...)
	if err != nil {
		folders.Close()
		return nil, fmt.Errorf("creating projects client: %w", err)
	}
	return &ResourceManagerClientImpl{
		folders:  &foldersAdapter{client: folders},
		projects: &projectsAdapter{client: projects},
	}, nil
}

func newResourceManagerWithAPI(api projectsAPI, api2 foldersAPI) *ResourceManagerClientImpl {
	return &ResourceManagerClientImpl{projects: api, folders: api2}
}

func (c *ResourceManagerClientImpl) Close() error {
	if err := c.folders.Close(); err != nil {
		c.projects.Close()
		return err
	}
	return c.projects.Close()
}
func (c *ResourceManagerClientImpl) EnsureFolderExists(ctx context.Context, displayName, parent string) (string, error) {
	ctx, span := tracing.Tracer("ensure folder exists").Start(ctx, "ensure folder exists",
		trace.WithAttributes(
			attribute.String("folder name", displayName)))
	defer span.End()

	op, err := c.folders.CreateFolder(ctx, &resourcemanagerpb.CreateFolderRequest{
		Folder: &resourcemanagerpb.Folder{
			Parent:      parent,
			DisplayName: displayName,
		},
	})
	if err != nil {
		if status.Code(err) != codes.AlreadyExists {
			span.RecordError(err)
			span.SetStatus(otelCodes.Error, err.Error())
			return "", fmt.Errorf("creating folder %s: %w", displayName, err)
		}
		span.SetStatus(otelCodes.Ok, "folder already exists")
		return c.findFolder(ctx, displayName, parent)
	}
	folder, err := op.Wait(ctx)
	if err != nil {
		span.RecordError(err)
		span.SetStatus(otelCodes.Error, err.Error())
		return "", fmt.Errorf("waiting for folder %s: %w", displayName, err)
	}
	slog.InfoContext(ctx, "folder created", "name", folder.Name)
	span.SetStatus(otelCodes.Ok, "folder created")
	return folder.Name, nil
}

func (c *ResourceManagerClientImpl) findFolder(ctx context.Context, displayName, parent string) (string, error) {
	iter := c.folders.ListFolders(ctx, &resourcemanagerpb.ListFoldersRequest{
		Parent: parent,
	})
	for {
		folder, err := iter.Next()
		if errors.Is(err, iterator.Done) {
			break
		}
		if err != nil {
			return "", fmt.Errorf("listing folders under %s: %w", parent, err)
		}
		if folder.DisplayName == displayName {
			slog.InfoContext(ctx, "folder already exists", "name", folder.Name)
			return folder.Name, nil
		}
	}
	return "", fmt.Errorf("folder %q not found under %s", displayName, parent)
}

// EnsureProjectExists creates a project if it doesn't exist
func (c *ResourceManagerClientImpl) EnsureProjectExists(ctx context.Context, projectID, displayName, parent string) error {
	ctx, span := tracing.Tracer("ensure project exists").Start(ctx, "ensure project exists",
		trace.WithAttributes(
			attribute.String("project ID", projectID)))
	defer span.End()
	_, err := c.projects.GetProject(ctx, &resourcemanagerpb.GetProjectRequest{
		Name: "projects/" + projectID,
	})
	if err == nil {
		slog.InfoContext(ctx, "project already exists", "projectID", projectID)
		span.SetStatus(otelCodes.Ok, "project already exists")
		return nil
	}
	if status.Code(err) != codes.NotFound {
		span.RecordError(err)
		span.SetStatus(otelCodes.Error, err.Error())
		return fmt.Errorf("checking project %s: %w", projectID, err)
	}

	op, err := c.projects.CreateProject(ctx, &resourcemanagerpb.CreateProjectRequest{
		Project: &resourcemanagerpb.Project{
			ProjectId:   projectID,
			DisplayName: displayName,
			Parent:      parent,
		},
	})
	if err != nil {
		span.RecordError(err)
		span.SetStatus(otelCodes.Error, err.Error())
		return fmt.Errorf("creating project %s: %w", projectID, err)
	}
	if _, err := op.Wait(ctx); err != nil {
		span.RecordError(err)
		span.SetStatus(otelCodes.Error, err.Error())
		return fmt.Errorf("waiting for project %s: %w", projectID, err)
	}
	slog.InfoContext(ctx, "project created", "projectID", projectID)
	span.SetStatus(otelCodes.Ok, "project created")
	return nil
}
