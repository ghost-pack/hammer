package gcp

import (
	"context"
	"errors"
	"fmt"
	"io"
	"log/slog"

	"cloud.google.com/go/storage"
)

type CloudStorageClient interface {
	EnsureBucketExists(ctx context.Context, projectId, location, bucketName string) error
	GetObject(ctx context.Context, bucket, object string) ([]byte, error)
	WriteObject(ctx context.Context, bucket, object string, data []byte, metadata map[string]string) error
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

func (c *CloudStorageClientImpl) GetObject(ctx context.Context, bucket, object string) ([]byte, error) {
	rc, err := c.client.Bucket(bucket).Object(object).NewReader(ctx)
	if err != nil {
		if errors.Is(err, storage.ErrObjectNotExist) {
			return nil, fmt.Errorf("object gs://%s/%s does not exist: %w", bucket, object, err)
		}
		return nil, fmt.Errorf("opening gs://%s/%s: %w", bucket, object, err)
	}
	defer rc.Close()

	data, err := io.ReadAll(rc)
	if err != nil {
		return nil, fmt.Errorf("reading gs://%s/%s: %w", bucket, object, err)
	}

	slog.InfoContext(ctx, fmt.Sprintf("read gs://%s/%s", bucket, object))
	return data, nil
}

func (c *CloudStorageClientImpl) WriteObject(ctx context.Context, bucket, object string, data []byte, metadata map[string]string) error {
	w := c.client.Bucket(bucket).Object(object).NewWriter(ctx)
	w.Metadata = metadata

	if _, err := w.Write(data); err != nil {
		_ = w.Close()
		return fmt.Errorf("writing gs://%s/%s: %w", bucket, object, err)
	}
	if err := w.Close(); err != nil {
		return fmt.Errorf("finalising gs://%s/%s: %w", bucket, object, err)
	}

	slog.InfoContext(ctx, fmt.Sprintf("wrote gs://%s/%s", bucket, object))
	return nil
}
