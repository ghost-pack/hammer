package gcp

import (
	"context"
	"errors"
	"fmt"
	"log/slog"

	"cloud.google.com/go/storage"
)

type CloudStorageClient interface {
	EnsureBucketExists(ctx context.Context, projectId, location, bucketName string) error
	Close() error
}

type CloudStorageClientImpl struct {
	client *storage.Client
}

func NewCloudStorageClient(ctx context.Context) (*CloudStorageClientImpl, error) {
	client, err := storage.NewClient(ctx)
	if err != nil {
		return nil, fmt.Errorf("creating cloud build client: %w", err)
	}
	return &CloudStorageClientImpl{client: client}, nil
}

func (c *CloudStorageClientImpl) Close() error {
	return c.client.Close()
}

func (c *CloudStorageClientImpl) EnsureBucketExists(ctx context.Context, projectId, location, bucketName string) error {
	_, err := c.client.Bucket(bucketName).Attrs(ctx)
	if err == nil {
		slog.InfoContext(ctx, fmt.Sprintf("Bucket gs://%s already exists.", bucketName))
		return nil
	}
	if !errors.Is(err, storage.ErrBucketNotExist) {
		return fmt.Errorf("checking bucket gs://%s: %w", bucketName, err)
	}
	slog.InfoContext(ctx, fmt.Sprintf("Bucket gs://%s does not exist - creating bucket.", bucketName))
	bucketAttrs := &storage.BucketAttrs{
		Location: location,
		UniformBucketLevelAccess: storage.UniformBucketLevelAccess{
			Enabled: true,
		},
		VersioningEnabled: true,
		Lifecycle: storage.Lifecycle{
			Rules: []storage.LifecycleRule{
				{
					Action: storage.LifecycleAction{Type: "Delete"},
					Condition: storage.LifecycleCondition{
						DaysSinceNoncurrentTime: 90,
						NumNewerVersions:        3,
					},
				},
			},
		},
		PublicAccessPrevention: storage.PublicAccessPreventionEnforced,
	}
	if err := c.client.Bucket(bucketName).Create(ctx, projectId, bucketAttrs); err != nil {
		return fmt.Errorf("creating bucket gs://%s: %w", bucketName, err)
	}

	slog.InfoContext(ctx, fmt.Sprintf("Bucket gs://%s created successfully.", bucketName))
	return nil
}
