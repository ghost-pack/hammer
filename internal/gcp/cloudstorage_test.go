package gcp

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"io"
	"testing"

	"cloud.google.com/go/storage"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
	"google.golang.org/api/iterator"
	"google.golang.org/api/option"
)

type MockStorageClientAPI struct{ mock.Mock }

func (m *MockStorageClientAPI) Bucket(name string) bucketHandleAPI {
	return m.Called(name).Get(0).(bucketHandleAPI)
}
func (m *MockStorageClientAPI) Close() error { return m.Called().Error(0) }

type MockBucketHandleAPI struct{ mock.Mock }

func (m *MockBucketHandleAPI) Attrs(ctx context.Context) (*storage.BucketAttrs, error) {
	args := m.Called(ctx)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*storage.BucketAttrs), args.Error(1)
}
func (m *MockBucketHandleAPI) Create(ctx context.Context, projectID string, attrs *storage.BucketAttrs) error {
	return m.Called(ctx, projectID, attrs).Error(0)
}
func (m *MockBucketHandleAPI) Object(name string) objectHandleAPI {
	return m.Called(name).Get(0).(objectHandleAPI)
}

func (m *MockBucketHandleAPI) Objects(ctx context.Context, q *storage.Query) objectIteratorAPI {
	// ctx and q can be ignored or asserted with matchers
	return m.Called(ctx, q).Get(0).(objectIteratorAPI)
}

type MockObjectHandleAPI struct{ mock.Mock }

func (m *MockObjectHandleAPI) NewReader(ctx context.Context) (io.ReadCloser, error) {
	args := m.Called(ctx)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(io.ReadCloser), args.Error(1)
}
func (m *MockObjectHandleAPI) NewWriter(ctx context.Context) storageWriterAPI {
	return m.Called(ctx).Get(0).(storageWriterAPI)
}

type MockStorageWriterAPI struct{ mock.Mock }

func (m *MockStorageWriterAPI) Write(p []byte) (int, error) {
	args := m.Called(p)
	return args.Int(0), args.Error(1)
}
func (m *MockStorageWriterAPI) Close() error                           { return m.Called().Error(0) }
func (m *MockStorageWriterAPI) SetMetadata(metadata map[string]string) { m.Called(metadata) }

type MockObjectIteratorAPI struct {
	mock.Mock
}

func (m *MockObjectIteratorAPI) Next() (*storage.ObjectAttrs, error) {
	args := m.Called()
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*storage.ObjectAttrs), args.Error(1)
}

func TestWriteObject(t *testing.T) {
	t.Run("writes object successfully", func(t *testing.T) {
		mockWriter := &MockStorageWriterAPI{}
		mockWriter.On("SetMetadata", mock.Anything).Return()
		mockWriter.On("Write", mock.Anything).Return(5, nil)
		mockWriter.On("Close").Return(nil)

		mockObject := &MockObjectHandleAPI{}
		mockObject.On("NewWriter", mock.Anything).Return(mockWriter)

		mockBucket := &MockBucketHandleAPI{}
		mockBucket.On("Object", "tenants/acme-corp.json").Return(mockObject)

		mockClient := &MockStorageClientAPI{}
		mockClient.On("Bucket", "my-registry-bucket").Return(mockBucket)

		client := newCloudStorageClientWithAPI(mockClient)
		err := client.WriteObject(context.Background(), "my-registry-bucket", "tenants/acme-corp.json", []byte("data"), nil)

		require.NoError(t, err)
		mockClient.AssertExpectations(t)
		mockBucket.AssertExpectations(t)
		mockObject.AssertExpectations(t)
		mockWriter.AssertExpectations(t)
	})

	t.Run("writes object fails", func(t *testing.T) {
		mockWriter := &MockStorageWriterAPI{}
		mockWriter.On("SetMetadata", mock.Anything).Return()
		mockWriter.On("Write", mock.Anything).Return(0, fmt.Errorf("error"))
		mockWriter.On("Close").Return(nil)

		mockObject := &MockObjectHandleAPI{}
		mockObject.On("NewWriter", mock.Anything).Return(mockWriter)

		mockBucket := &MockBucketHandleAPI{}
		mockBucket.On("Object", "tenants/acme-corp.json").Return(mockObject)

		mockClient := &MockStorageClientAPI{}
		mockClient.On("Bucket", "my-registry-bucket").Return(mockBucket)

		client := newCloudStorageClientWithAPI(mockClient)
		err := client.WriteObject(context.Background(), "my-registry-bucket", "tenants/acme-corp.json", []byte("data"), nil)

		require.Error(t, err)
		mockClient.AssertExpectations(t)
		mockBucket.AssertExpectations(t)
		mockObject.AssertExpectations(t)
		mockWriter.AssertExpectations(t)
	})

	t.Run("writes object fails close", func(t *testing.T) {
		mockWriter := &MockStorageWriterAPI{}
		mockWriter.On("SetMetadata", mock.Anything).Return()
		mockWriter.On("Write", mock.Anything).Return(5, nil)
		mockWriter.On("Close").Return(fmt.Errorf("error"))

		mockObject := &MockObjectHandleAPI{}
		mockObject.On("NewWriter", mock.Anything).Return(mockWriter)

		mockBucket := &MockBucketHandleAPI{}
		mockBucket.On("Object", "tenants/acme-corp.json").Return(mockObject)

		mockClient := &MockStorageClientAPI{}
		mockClient.On("Bucket", "my-registry-bucket").Return(mockBucket)

		client := newCloudStorageClientWithAPI(mockClient)
		err := client.WriteObject(context.Background(), "my-registry-bucket", "tenants/acme-corp.json", []byte("data"), nil)

		require.Error(t, err)
		mockClient.AssertExpectations(t)
		mockBucket.AssertExpectations(t)
		mockObject.AssertExpectations(t)
		mockWriter.AssertExpectations(t)
	})
}

func TestGetObject(t *testing.T) {
	t.Run("returns object data successfully", func(t *testing.T) {
		expectedData := []byte("hello from gcs")

		mockObject := &MockObjectHandleAPI{}
		mockObject.On("NewReader", mock.Anything).
			Return(io.NopCloser(bytes.NewReader(expectedData)), nil)

		mockBucket := &MockBucketHandleAPI{}
		mockBucket.On("Object", "tenants/acme-corp.json").Return(mockObject)

		mockClient := &MockStorageClientAPI{}
		mockClient.On("Bucket", "my-registry-bucket").Return(mockBucket)

		client := newCloudStorageClientWithAPI(mockClient)
		data, err := client.GetObject(context.Background(), "my-registry-bucket", "tenants/acme-corp.json")

		require.NoError(t, err)
		require.Equal(t, expectedData, data)
		mockClient.AssertExpectations(t)
		mockBucket.AssertExpectations(t)
		mockObject.AssertExpectations(t)
	})

	t.Run("returns error when object does not exist", func(t *testing.T) {
		mockObject := &MockObjectHandleAPI{}
		mockObject.On("NewReader", mock.Anything).
			Return(nil, storage.ErrObjectNotExist)

		mockBucket := &MockBucketHandleAPI{}
		mockBucket.On("Object", "tenants/acme-corp.json").Return(mockObject)

		mockClient := &MockStorageClientAPI{}
		mockClient.On("Bucket", "my-registry-bucket").Return(mockBucket)

		client := newCloudStorageClientWithAPI(mockClient)
		_, err := client.GetObject(context.Background(), "my-registry-bucket", "tenants/acme-corp.json")

		require.Error(t, err)
		require.ErrorContains(t, err, "does not exist")
		require.ErrorIs(t, err, storage.ErrObjectNotExist)
	})

	t.Run("returns error on unexpected reader failure", func(t *testing.T) {
		mockObject := &MockObjectHandleAPI{}
		mockObject.On("NewReader", mock.Anything).
			Return(nil, errors.New("transport error"))

		mockBucket := &MockBucketHandleAPI{}
		mockBucket.On("Object", "tenants/acme-corp.json").Return(mockObject)

		mockClient := &MockStorageClientAPI{}
		mockClient.On("Bucket", "my-registry-bucket").Return(mockBucket)

		client := newCloudStorageClientWithAPI(mockClient)
		_, err := client.GetObject(context.Background(), "my-registry-bucket", "tenants/acme-corp.json")

		require.Error(t, err)
		require.ErrorContains(t, err, "opening gs://")
		require.ErrorContains(t, err, "transport error")
	})
}

func TestEnsureBucketExists(t *testing.T) {
	t.Run("does nothing when bucket already exists", func(t *testing.T) {
		mockBucket := &MockBucketHandleAPI{}
		mockBucket.On("Attrs", mock.Anything).
			Return(&storage.BucketAttrs{Name: "my-bucket"}, nil)

		mockClient := &MockStorageClientAPI{}
		mockClient.On("Bucket", "my-bucket").Return(mockBucket)

		client := newCloudStorageClientWithAPI(mockClient)
		err := client.EnsureBucketExists(context.Background(), "my-project", "us-central1", "my-bucket")

		require.NoError(t, err)
		mockBucket.AssertNotCalled(t, "Create", mock.Anything, mock.Anything, mock.Anything)
		mockClient.AssertExpectations(t)
		mockBucket.AssertExpectations(t)
	})

	t.Run("creates bucket when it does not exist", func(t *testing.T) {
		mockBucket := &MockBucketHandleAPI{}
		mockBucket.On("Attrs", mock.Anything).
			Return(nil, storage.ErrBucketNotExist)
		mockBucket.On("Create", mock.Anything, "my-project", mock.MatchedBy(func(attrs *storage.BucketAttrs) bool {
			return attrs.Location == "us-central1" &&
				attrs.VersioningEnabled &&
				attrs.UniformBucketLevelAccess.Enabled &&
				attrs.PublicAccessPrevention == storage.PublicAccessPreventionEnforced
		})).Return(nil)

		mockClient := &MockStorageClientAPI{}
		mockClient.On("Bucket", "my-bucket").Return(mockBucket)

		client := newCloudStorageClientWithAPI(mockClient)
		err := client.EnsureBucketExists(context.Background(), "my-project", "us-central1", "my-bucket")

		require.NoError(t, err)
		mockClient.AssertExpectations(t)
		mockBucket.AssertExpectations(t)
	})

	t.Run("returns error on unexpected Attrs failure", func(t *testing.T) {
		mockBucket := &MockBucketHandleAPI{}
		mockBucket.On("Attrs", mock.Anything).
			Return(nil, errors.New("permission denied"))

		mockClient := &MockStorageClientAPI{}
		mockClient.On("Bucket", "my-bucket").Return(mockBucket)

		client := newCloudStorageClientWithAPI(mockClient)
		err := client.EnsureBucketExists(context.Background(), "my-project", "us-central1", "my-bucket")

		require.Error(t, err)
		require.ErrorContains(t, err, "checking bucket")
		require.ErrorContains(t, err, "permission denied")
		mockBucket.AssertNotCalled(t, "Create", mock.Anything, mock.Anything, mock.Anything)
	})

	t.Run("returns error when bucket creation fails", func(t *testing.T) {
		mockBucket := &MockBucketHandleAPI{}
		mockBucket.On("Attrs", mock.Anything).
			Return(nil, storage.ErrBucketNotExist)
		mockBucket.On("Create", mock.Anything, mock.Anything, mock.Anything).
			Return(errors.New("quota exceeded"))

		mockClient := &MockStorageClientAPI{}
		mockClient.On("Bucket", "my-bucket").Return(mockBucket)

		client := newCloudStorageClientWithAPI(mockClient)
		err := client.EnsureBucketExists(context.Background(), "my-project", "us-central1", "my-bucket")

		require.Error(t, err)
		require.ErrorContains(t, err, "creating bucket")
		require.ErrorContains(t, err, "quota exceeded")
	})
}

func TestNewCloudStorageClient(t *testing.T) {
	tests := []struct {
		name                    string
		setupCloudStorageClient func(ctx context.Context, opts ...option.ClientOption) (CloudStorageClient, error)
		wantErr                 bool
	}{
		{
			name: "failed client creation",
			setupCloudStorageClient: func(ctx context.Context, opts ...option.ClientOption) (CloudStorageClient, error) {
				client, err := NewCloudStorageClient(ctx, opts...)
				if err != nil {
					return nil, err
				}
				return client, nil
			},
			wantErr: true,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, creationErr := tt.setupCloudStorageClient(context.Background(), option.WithCredentialsFile("/nonexistent/credentials.json"))
			if creationErr != nil && tt.wantErr {
				require.Error(t, creationErr)
			} else {
				require.NoError(t, creationErr)
			}
		})
	}
}

func TestListPrefixes(t *testing.T) {

	t.Run("successfully list prefixes", func(t *testing.T) {
		mockBucket := new(MockBucketHandleAPI)
		mockClient := new(MockStorageClientAPI)

		// Setup: client.Bucket("my-bucket") returns mockBucket
		mockClient.On("Bucket", "my-bucket").Return(mockBucket)

		// The iterator mock – we’ll make it return two prefixes then finish
		mockIter := new(MockObjectIteratorAPI)
		// First call returns a Prefix
		mockIter.On("Next").Return(&storage.ObjectAttrs{Prefix: "tenants/tenant-a/"}, nil).Once()
		// Second call returns another Prefix
		mockIter.On("Next").Return(&storage.ObjectAttrs{Prefix: "tenants/tenant-b/"}, nil).Once()
		// Third call signals done
		mockIter.On("Next").Return(nil, iterator.Done).Once()

		mockBucket.On("Objects", mock.Anything, mock.Anything).Return(mockIter)

		client := newCloudStorageClientWithAPI(mockClient)

		prefixes, err := client.ListPrefixes(context.Background(), "my-bucket", "tenants/", "/")
		require.NoError(t, err)
		require.Equal(t, []string{"tenants/tenant-a/", "tenants/tenant-b/"}, prefixes)

		mockClient.AssertExpectations(t)
		mockBucket.AssertExpectations(t)
		mockIter.AssertExpectations(t)
	})

	t.Run("Error list prefixes captured correctly", func(t *testing.T) {
		mockBucket := new(MockBucketHandleAPI)
		mockClient := new(MockStorageClientAPI)

		// Setup: client.Bucket("my-bucket") returns mockBucket
		mockClient.On("Bucket", "my-bucket").Return(mockBucket)

		// The iterator mock – we’ll make it return two prefixes then finish
		mockIter := new(MockObjectIteratorAPI)
		// First call returns a Prefix
		mockIter.On("Next").Return(&storage.ObjectAttrs{Prefix: "tenants/tenant-a/"}, nil).Once()
		// Second call returns another Prefix
		mockIter.On("Next").Return(nil, fmt.Errorf("some error")).Once()

		mockBucket.On("Objects", mock.Anything, mock.Anything).Return(mockIter)

		client := newCloudStorageClientWithAPI(mockClient)

		prefixes, err := client.ListPrefixes(context.Background(), "my-bucket", "tenants/", "/")
		require.Error(t, err)
		require.Empty(t, prefixes)

		mockClient.AssertExpectations(t)
		mockBucket.AssertExpectations(t)
		mockIter.AssertExpectations(t)
	})
}

func TestBucketExists(t *testing.T) {
	t.Run("returns true when bucket exists", func(t *testing.T) {
		mockBucket := &MockBucketHandleAPI{}
		mockBucket.On("Attrs", mock.Anything).
			Return(&storage.BucketAttrs{Name: "my-bucket"}, nil)

		mockClient := &MockStorageClientAPI{}
		mockClient.On("Bucket", "my-bucket").Return(mockBucket)

		client := newCloudStorageClientWithAPI(mockClient)
		exists, err := client.BucketExists(context.Background(), "my-bucket")

		require.NoError(t, err)
		require.True(t, exists)

		mockClient.AssertExpectations(t)
		mockBucket.AssertExpectations(t)
	})

	t.Run("returns false when bucket does not exist", func(t *testing.T) {
		mockBucket := &MockBucketHandleAPI{}
		mockBucket.On("Attrs", mock.Anything).
			Return(nil, storage.ErrBucketNotExist)

		mockClient := &MockStorageClientAPI{}
		mockClient.On("Bucket", "my-bucket").Return(mockBucket)

		client := newCloudStorageClientWithAPI(mockClient)
		exists, err := client.BucketExists(context.Background(), "my-bucket")

		require.NoError(t, err)
		require.False(t, exists)

		mockClient.AssertExpectations(t)
		mockBucket.AssertExpectations(t)
	})

	t.Run("returns error on unexpected Attrs failure", func(t *testing.T) {
		mockBucket := &MockBucketHandleAPI{}
		mockBucket.On("Attrs", mock.Anything).
			Return(nil, errors.New("permission denied"))

		mockClient := &MockStorageClientAPI{}
		mockClient.On("Bucket", "my-bucket").Return(mockBucket)

		client := newCloudStorageClientWithAPI(mockClient)
		exists, err := client.BucketExists(context.Background(), "my-bucket")

		require.Error(t, err)
		require.False(t, exists)
		require.ErrorContains(t, err, "checking bucket gs://my-bucket")
		require.ErrorContains(t, err, "permission denied")

		mockClient.AssertExpectations(t)
		mockBucket.AssertExpectations(t)
	})
}
