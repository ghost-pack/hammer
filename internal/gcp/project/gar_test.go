package project

import (
	"context"
	"fmt"
	"testing"

	artifactregistry "cloud.google.com/go/artifactregistry/apiv1"
	"cloud.google.com/go/artifactregistry/apiv1/artifactregistrypb"
	"cloud.google.com/go/iam/apiv1/iampb"
	"github.com/googleapis/gax-go/v2"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
	"google.golang.org/api/option"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

type MockGarAPI struct {
	mock.Mock
}

func (m *MockGarAPI) GetRepository(ctx context.Context, req *artifactregistrypb.GetRepositoryRequest, opts ...gax.CallOption) (*artifactregistrypb.Repository, error) {
	args := m.Called(ctx, req)
	op, _ := args.Get(0).(*artifactregistrypb.Repository)
	return op, args.Error(1)
}

func (m *MockGarAPI) CreateRepository(ctx context.Context, req *artifactregistrypb.CreateRepositoryRequest, opts ...gax.CallOption) (*artifactregistry.CreateRepositoryOperation, error) {
	args := m.Called(ctx, req)
	op, _ := args.Get(0).(*artifactregistry.CreateRepositoryOperation)
	return op, args.Error(1)
}

func (m *MockGarAPI) GetIamPolicy(ctx context.Context, req *iampb.GetIamPolicyRequest, opts ...gax.CallOption) (*iampb.Policy, error) {
	args := m.Called(ctx, req)
	op, _ := args.Get(0).(*iampb.Policy)
	return op, args.Error(1)
}

func (m *MockGarAPI) SetIamPolicy(ctx context.Context, req *iampb.SetIamPolicyRequest, opts ...gax.CallOption) (*iampb.Policy, error) {
	args := m.Called(ctx, req)
	op, _ := args.Get(0).(*iampb.Policy)
	return op, args.Error(1)
}

func (m *MockGarAPI) Close() error {
	args := m.Called()
	return args.Error(0)
}

func TestEnsureRepository(t *testing.T) {
	t.Run("successfully get existing repository", func(t *testing.T) {
		mockAPI := &MockGarAPI{}
		mockAPI.On("GetRepository", mock.Anything, mock.MatchedBy(func(req *artifactregistrypb.GetRepositoryRequest) bool {
			return req.Name == "projects/my-project/locations/us-central1/repositories/my-repo"
		})).Return(nil, nil)

		client := newGarClientWithAPI(mockAPI)
		err := client.EnsureRepository(context.Background(), "my-project", "us-central1", "my-repo")
		require.NoError(t, err)

		mockAPI.AssertExpectations(t)
	})

	t.Run("successfully create repository", func(t *testing.T) {
		mockAPI := &MockGarAPI{}
		mockAPI.On("GetRepository", mock.Anything, mock.MatchedBy(func(req *artifactregistrypb.GetRepositoryRequest) bool {
			return req.Name == "projects/my-project/locations/us-central1/repositories/my-repo"
		})).Return(nil, status.Error(codes.NotFound, "repo not found"))

		mockAPI.On("CreateRepository", mock.Anything, mock.MatchedBy(func(req *artifactregistrypb.CreateRepositoryRequest) bool {
			return req.Parent == "projects/my-project/locations/us-central1" &&
				req.RepositoryId == "my-repo" &&
				req.Repository.Format == artifactregistrypb.Repository_DOCKER
		})).Return(nil, nil)

		client := newGarClientWithAPI(mockAPI)
		err := client.EnsureRepository(context.Background(), "my-project", "us-central1", "my-repo")
		require.NoError(t, err)

		mockAPI.AssertExpectations(t)
	})

	t.Run("fail get existing repository", func(t *testing.T) {
		mockAPI := &MockGarAPI{}
		mockAPI.On("GetRepository", mock.Anything, mock.MatchedBy(func(req *artifactregistrypb.GetRepositoryRequest) bool {
			return req.Name == "projects/my-project/locations/us-central1/repositories/my-repo"
		})).Return(nil, fmt.Errorf("some error"))

		client := newGarClientWithAPI(mockAPI)
		err := client.EnsureRepository(context.Background(), "my-project", "us-central1", "my-repo")
		require.Error(t, err)

		mockAPI.AssertExpectations(t)
	})

	t.Run("successfully do not create repository if already exists", func(t *testing.T) {
		mockAPI := &MockGarAPI{}
		mockAPI.On("GetRepository", mock.Anything, mock.MatchedBy(func(req *artifactregistrypb.GetRepositoryRequest) bool {
			return req.Name == "projects/my-project/locations/us-central1/repositories/my-repo"
		})).Return(nil, status.Error(codes.NotFound, "repo not found"))

		mockAPI.On("CreateRepository", mock.Anything, mock.MatchedBy(func(req *artifactregistrypb.CreateRepositoryRequest) bool {
			return req.Parent == "projects/my-project/locations/us-central1" &&
				req.RepositoryId == "my-repo" &&
				req.Repository.Format == artifactregistrypb.Repository_DOCKER
		})).Return(nil, status.Error(codes.AlreadyExists, "already exists"))

		client := newGarClientWithAPI(mockAPI)
		err := client.EnsureRepository(context.Background(), "my-project", "us-central1", "my-repo")
		require.NoError(t, err)

		mockAPI.AssertExpectations(t)
	})

	t.Run("fail to create repository", func(t *testing.T) {
		mockAPI := &MockGarAPI{}
		mockAPI.On("GetRepository", mock.Anything, mock.MatchedBy(func(req *artifactregistrypb.GetRepositoryRequest) bool {
			return req.Name == "projects/my-project/locations/us-central1/repositories/my-repo"
		})).Return(nil, status.Error(codes.NotFound, "repo not found"))

		mockAPI.On("CreateRepository", mock.Anything, mock.MatchedBy(func(req *artifactregistrypb.CreateRepositoryRequest) bool {
			return req.Parent == "projects/my-project/locations/us-central1" &&
				req.RepositoryId == "my-repo" &&
				req.Repository.Format == artifactregistrypb.Repository_DOCKER
		})).Return(nil, status.Error(codes.FailedPrecondition, "precondition failed"))

		client := newGarClientWithAPI(mockAPI)
		err := client.EnsureRepository(context.Background(), "my-project", "us-central1", "my-repo")
		require.Error(t, err)

		mockAPI.AssertExpectations(t)
	})

}

func TestGrantRepositoryReader(t *testing.T) {
	t.Run("successfully grant repository reader", func(t *testing.T) {
		mockAPI := &MockGarAPI{}
		mockAPI.On("GetIamPolicy", mock.Anything, mock.MatchedBy(func(req *iampb.GetIamPolicyRequest) bool {
			return req.Resource == fmt.Sprintf("projects/%s/locations/%s/repositories/%s", "my-project", "us-central1", "my-repo")
		})).Return(&iampb.Policy{}, nil)

		mockAPI.On("SetIamPolicy", mock.Anything, mock.Anything).Return(nil, nil)

		client := newGarClientWithAPI(mockAPI)
		err := client.GrantRepositoryReader(context.Background(), "my-project", "us-central1", "my-repo", "sa-pipeline@hammer-bootstrap.iam.gserviceaccount.com")
		require.NoError(t, err)

		mockAPI.AssertExpectations(t)
	})

	t.Run("fail to get policy", func(t *testing.T) {
		mockAPI := &MockGarAPI{}
		mockAPI.On("GetIamPolicy", mock.Anything, mock.MatchedBy(func(req *iampb.GetIamPolicyRequest) bool {
			return req.Resource == fmt.Sprintf("projects/%s/locations/%s/repositories/%s", "my-project", "us-central1", "my-repo")
		})).Return(nil, fmt.Errorf("some error"))

		client := newGarClientWithAPI(mockAPI)
		err := client.GrantRepositoryReader(context.Background(), "my-project", "us-central1", "my-repo", "sa-pipeline@hammer-bootstrap.iam.gserviceaccount.com")
		require.Error(t, err)

		mockAPI.AssertExpectations(t)
	})

	t.Run("fail to set policy", func(t *testing.T) {
		mockAPI := &MockGarAPI{}
		mockAPI.On("GetIamPolicy", mock.Anything, mock.MatchedBy(func(req *iampb.GetIamPolicyRequest) bool {
			return req.Resource == fmt.Sprintf("projects/%s/locations/%s/repositories/%s", "my-project", "us-central1", "my-repo")
		})).Return(&iampb.Policy{}, nil)

		mockAPI.On("SetIamPolicy", mock.Anything, mock.Anything).Return(nil, fmt.Errorf("some error"))

		client := newGarClientWithAPI(mockAPI)
		err := client.GrantRepositoryReader(context.Background(), "my-project", "us-central1", "my-repo", "sa-pipeline@hammer-bootstrap.iam.gserviceaccount.com")
		require.Error(t, err)

		mockAPI.AssertExpectations(t)
	})
}

func TestNewGarClient(t *testing.T) {
	tests := []struct {
		name           string
		setupGarClient func(ctx context.Context, opts ...option.ClientOption) (GarClient, error)
		wantErr        bool
	}{
		{
			name: "failed client creation",
			setupGarClient: func(ctx context.Context, opts ...option.ClientOption) (GarClient, error) {
				client, err := NewGarClient(ctx, opts...)
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
			_, creationErr := tt.setupGarClient(context.Background(), option.WithCredentialsFile("/nonexistent/credentials.json"))
			if creationErr != nil && tt.wantErr {
				require.Error(t, creationErr)
			} else {
				require.NoError(t, creationErr)
			}
		})
	}
}

func TestGarClientClose(t *testing.T) {
	t.Run("successfully close", func(t *testing.T) {
		mockAPI := &MockGarAPI{}
		mockAPI.On("Close").Return(nil)

		client := newGarClientWithAPI(mockAPI)
		err := client.Close()
		require.NoError(t, err)

		mockAPI.AssertExpectations(t)
	})

}

func TestGarClientImpl_Close(t *testing.T) {
	tests := []struct {
		name           string
		setupGarClient func(ctx context.Context, opts ...option.ClientOption) (GarClient, error)
		wantErr        bool
	}{
		{
			name: "successful close",
			setupGarClient: func(ctx context.Context, opts ...option.ClientOption) (GarClient, error) {
				client, err := NewGarClient(ctx, opts...)
				if err != nil {
					return nil, err
				}
				return client, nil
			},
			wantErr: false,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			g, creationErr := tt.setupGarClient(context.Background())
			defer g.Close()
			require.NoError(t, creationErr)
			if err := g.Close(); (err != nil) != tt.wantErr {
				require.Error(t, err)
			} else {
				require.NoError(t, err)
			}
		})
	}
}
