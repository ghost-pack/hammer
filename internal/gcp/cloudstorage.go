package gcp

import (
	"context"
	"errors"
	"fmt"
	"io"
	"log/slog"

	"cloud.google.com/go/storage"
	"github.com/ghost-pack/hammer/internal/observability/tracing"
	"go.opentelemetry.io/otel/attribute"
	otelCodes "go.opentelemetry.io/otel/codes"
	"go.opentelemetry.io/otel/trace"
	"google.golang.org/api/iterator"
	"google.golang.org/api/option"
)

// -- interfaces for each level of the chain --

type CloudStorageClient interface {
	EnsureBucketExists(ctx context.Context, projectId, location, bucketName string) error
	GetObject(ctx context.Context, bucket, object string) ([]byte, error)
	WriteObject(ctx context.Context, bucket, object string, data []byte, metadata map[string]string) error
	ListPrefixes(ctx context.Context, bucket, prefix, delimiter string) ([]string, error)
	Close() error
}

type storageClientAPI interface {
	Bucket(name string) bucketHandleAPI
	Close() error
}

type objectIteratorAPI interface {
	Next() (*storage.ObjectAttrs, error)
}

type bucketHandleAPI interface {
	Attrs(ctx context.Context) (*storage.BucketAttrs, error)
	Create(ctx context.Context, projectID string, attrs *storage.BucketAttrs) error
	Object(name string) objectHandleAPI
	Objects(ctx context.Context, q *storage.Query) objectIteratorAPI
}

type objectHandleAPI interface {
	NewReader(ctx context.Context) (io.ReadCloser, error)
	NewWriter(ctx context.Context) storageWriterAPI
}

// -- adapters --

type storageWriterAPI interface {
	io.WriteCloser
	SetMetadata(metadata map[string]string)
}

type storageClientAdapter struct {
	client *storage.Client
}

func (a *storageClientAdapter) Bucket(name string) bucketHandleAPI {
	return &bucketHandleAdapter{handle: a.client.Bucket(name)}
}

func (a *storageClientAdapter) Close() error {
	return a.client.Close()
}

type bucketHandleAdapter struct {
	handle *storage.BucketHandle
}

func (a *bucketHandleAdapter) Attrs(ctx context.Context) (*storage.BucketAttrs, error) {
	return a.handle.Attrs(ctx)
}

func (a *bucketHandleAdapter) Create(ctx context.Context, projectID string, attrs *storage.BucketAttrs) error {
	return a.handle.Create(ctx, projectID, attrs)
}

func (a *bucketHandleAdapter) Object(name string) objectHandleAPI {
	return &objectHandleAdapter{handle: a.handle.Object(name)}
}

func (a *bucketHandleAdapter) Objects(ctx context.Context, q *storage.Query) objectIteratorAPI {
	return &bucketIteratorAdapter{it: a.handle.Objects(ctx, q)}
}

type bucketIteratorAdapter struct {
	it *storage.ObjectIterator
}

func (a *bucketIteratorAdapter) Next() (*storage.ObjectAttrs, error) {
	return a.it.Next()
}

type objectHandleAdapter struct {
	handle *storage.ObjectHandle
}

func (a *objectHandleAdapter) NewReader(ctx context.Context) (io.ReadCloser, error) {
	return a.handle.NewReader(ctx)
}

func (a *objectHandleAdapter) NewWriter(ctx context.Context) storageWriterAPI {
	return &storageWriterAdapter{writer: a.handle.NewWriter(ctx)}
}

type storageWriterAdapter struct {
	writer *storage.Writer
}

func (a *storageWriterAdapter) Write(p []byte) (int, error) {
	return a.writer.Write(p)
}

func (a *storageWriterAdapter) Close() error {
	return a.writer.Close()
}

func (a *storageWriterAdapter) SetMetadata(metadata map[string]string) {
	a.writer.Metadata = metadata
}

type CloudStorageClientImpl struct {
	client storageClientAPI // ← interface now, not *storage.Client
}

func NewCloudStorageClient(ctx context.Context, opts ...option.ClientOption) (*CloudStorageClientImpl, error) {
	client, err := storage.NewClient(ctx, opts...)
	if err != nil {
		return nil, fmt.Errorf("creating cloud storage client: %w", err)
	}
	return &CloudStorageClientImpl{client: &storageClientAdapter{client: client}}, nil
}

func newCloudStorageClientWithAPI(api storageClientAPI) *CloudStorageClientImpl {
	return &CloudStorageClientImpl{client: api}
}

func (c *CloudStorageClientImpl) Close() error {
	return c.client.Close()
}

func (c *CloudStorageClientImpl) EnsureBucketExists(ctx context.Context, projectId, location, bucketName string) error {
	ctx, span := tracing.Tracer("ensure GCS bucket exists").Start(ctx, "ensure GCS bucket exists",
		trace.WithAttributes(
			attribute.String("bucket", bucketName),
			attribute.String("location", location),
			attribute.String("projectId", projectId)))
	defer span.End()
	_, err := c.client.Bucket(bucketName).Attrs(ctx)
	if err == nil {
		slog.InfoContext(ctx, fmt.Sprintf("Bucket gs://%s already exists.", bucketName))
		span.SetStatus(otelCodes.Ok, "Bucket already exists")
		return nil
	}
	if !errors.Is(err, storage.ErrBucketNotExist) {
		span.RecordError(err)
		span.SetStatus(otelCodes.Error, err.Error())
		return fmt.Errorf("checking bucket gs://%s: %w", bucketName, err)
	}
	slog.InfoContext(ctx, fmt.Sprintf("Bucket gs://%s does not exist - creating bucket.", bucketName))
	bucketAttrs := &storage.BucketAttrs{
		Location:                 location,
		UniformBucketLevelAccess: storage.UniformBucketLevelAccess{Enabled: true},
		VersioningEnabled:        true,
		Lifecycle: storage.Lifecycle{
			Rules: []storage.LifecycleRule{
				{
					Action:    storage.LifecycleAction{Type: "Delete"},
					Condition: storage.LifecycleCondition{DaysSinceNoncurrentTime: 90, NumNewerVersions: 3},
				},
				{
					Action: storage.LifecycleAction{Type: "Delete"},
					Condition: storage.LifecycleCondition{
						AgeInDays:     30,
						MatchesPrefix: []string{"deployments/oam/", "deployments/opentofu/", "deployments/cloudbuild/"},
					},
				},
			},
		},
		PublicAccessPrevention: storage.PublicAccessPreventionEnforced,
	}
	if err := c.client.Bucket(bucketName).Create(ctx, projectId, bucketAttrs); err != nil {
		span.RecordError(err)
		span.SetStatus(otelCodes.Error, err.Error())
		return fmt.Errorf("creating bucket gs://%s: %w", bucketName, err)
	}
	slog.InfoContext(ctx, fmt.Sprintf("Bucket gs://%s created successfully.", bucketName))
	span.SetStatus(otelCodes.Ok, "Bucket created")
	return nil
}

func (c *CloudStorageClientImpl) GetObject(ctx context.Context, bucket, object string) ([]byte, error) {
	ctx, span := tracing.Tracer("get object from GCS bucket").Start(ctx, "get object from GCS bucket",
		trace.WithAttributes(
			attribute.String("bucket", bucket)))
	defer span.End()

	rc, err := c.client.Bucket(bucket).Object(object).NewReader(ctx)
	if err != nil {
		span.RecordError(err)
		span.SetStatus(otelCodes.Error, err.Error())
		if errors.Is(err, storage.ErrObjectNotExist) {
			return nil, fmt.Errorf("object gs://%s/%s does not exist: %w", bucket, object, err)
		}
		return nil, fmt.Errorf("opening gs://%s/%s: %w", bucket, object, err)
	}
	defer rc.Close()

	data, err := io.ReadAll(rc)
	if err != nil {
		span.RecordError(err)
		span.SetStatus(otelCodes.Error, err.Error())
		return nil, fmt.Errorf("reading gs://%s/%s: %w", bucket, object, err)
	}
	slog.InfoContext(ctx, fmt.Sprintf("read gs://%s/%s", bucket, object))
	span.SetStatus(otelCodes.Ok, "Object read")
	return data, nil
}

func (c *CloudStorageClientImpl) WriteObject(ctx context.Context, bucket, object string, data []byte, metadata map[string]string) error {
	ctx, span := tracing.Tracer("write object from GCS bucket").Start(ctx, "write object from GCS bucket",
		trace.WithAttributes(
			attribute.String("bucket", bucket)))
	defer span.End()

	w := c.client.Bucket(bucket).Object(object).NewWriter(ctx)
	w.SetMetadata(metadata)

	if _, err := w.Write(data); err != nil {
		span.RecordError(err)
		span.SetStatus(otelCodes.Error, err.Error())
		_ = w.Close()
		return fmt.Errorf("writing gs://%s/%s: %w", bucket, object, err)
	}
	if err := w.Close(); err != nil {
		span.RecordError(err)
		span.SetStatus(otelCodes.Error, err.Error())
		return fmt.Errorf("finalising gs://%s/%s: %w", bucket, object, err)
	}
	slog.InfoContext(ctx, fmt.Sprintf("wrote gs://%s/%s", bucket, object))
	span.SetStatus(otelCodes.Ok, "Object wrote")
	return nil
}

func (c *CloudStorageClientImpl) ListPrefixes(ctx context.Context, bucket, prefix, delimiter string) ([]string, error) {
	q := &storage.Query{
		Prefix:    prefix,
		Delimiter: delimiter,
	}
	it := c.client.Bucket(bucket).Objects(ctx, q)
	var prefixes []string
	for {
		attrs, err := it.Next()
		if errors.Is(err, iterator.Done) {
			break
		}
		if err != nil {
			return nil, fmt.Errorf("listing prefixes: %w", err)
		}
		if attrs.Prefix != "" {
			prefixes = append(prefixes, attrs.Prefix)
		}
	}
	return prefixes, nil
}
