package project

import (
	"context"
	"fmt"
	"testing"

	"cloud.google.com/go/resourcemanager/apiv3/resourcemanagerpb"
	"github.com/googleapis/gax-go/v2"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
	"google.golang.org/api/iterator"
	"google.golang.org/api/option"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
	"google.golang.org/protobuf/proto"
)

type MockCreateFolderOperation struct {
	mock.Mock
}

func (m *MockCreateFolderOperation) Wait(ctx context.Context, opts ...gax.CallOption) (*resourcemanagerpb.Folder, error) {
	args := m.Called(ctx)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*resourcemanagerpb.Folder), args.Error(1)
}

type MockFolderIterator struct {
	mock.Mock
}

func (m *MockFolderIterator) Next() (*resourcemanagerpb.Folder, error) {
	args := m.Called()
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*resourcemanagerpb.Folder), args.Error(1)
}

type MockFoldersAPI struct {
	mock.Mock
}

func (m *MockFoldersAPI) CreateFolder(ctx context.Context, req *resourcemanagerpb.CreateFolderRequest, opts ...gax.CallOption) (createFolderOperation, error) {
	args := m.Called(ctx, req)
	op, _ := args.Get(0).(createFolderOperation)
	return op, args.Error(1)
}

func (m *MockFoldersAPI) ListFolders(ctx context.Context, req *resourcemanagerpb.ListFoldersRequest, opts ...gax.CallOption) folderIterator {
	args := m.Called(ctx, req)
	return args.Get(0).(folderIterator)
}

func (m *MockFoldersAPI) Close() error {
	args := m.Called()
	return args.Error(0)
}

func TestEnsureFolderExists(t *testing.T) {
	t.Run("successfully create folder", func(t *testing.T) {
		mockOp := &MockCreateFolderOperation{}
		mockOp.On("Wait", mock.Anything).Return(&resourcemanagerpb.Folder{
			Name:        "folders/123456789",
			DisplayName: "my-display-name",
		}, nil)

		mockFoldersApi := &MockFoldersAPI{}
		mockFoldersApi.On("CreateFolder", mock.Anything, mock.MatchedBy(func(req *resourcemanagerpb.CreateFolderRequest) bool {
			expected := &resourcemanagerpb.CreateFolderRequest{
				Folder: &resourcemanagerpb.Folder{
					Parent:      "my-parent",
					DisplayName: "my-display-name",
				},
			}
			return proto.Equal(expected, req)
		})).Return(mockOp, nil)

		client := newResourceManagerWithAPI(nil, mockFoldersApi)
		name, err := client.EnsureFolderExists(context.Background(), "my-display-name", "my-parent")

		require.NoError(t, err)
		require.Equal(t, "folders/123456789", name)
		mockFoldersApi.AssertExpectations(t)
		mockOp.AssertExpectations(t)
	})

	t.Run("successfully find folder", func(t *testing.T) {
		mockIter := &MockFolderIterator{}
		// first call returns a folder that doesn't match
		mockIter.On("Next").Return(&resourcemanagerpb.Folder{
			Name:        "folders/111",
			DisplayName: "other-folder",
		}, nil).Once()
		// second call returns the one we're looking for
		mockIter.On("Next").Return(&resourcemanagerpb.Folder{
			Name:        "folders/123456789",
			DisplayName: "my-display-name",
		}, nil).Once()

		mockFoldersApi := &MockFoldersAPI{}
		mockFoldersApi.On("CreateFolder", mock.Anything, mock.MatchedBy(func(req *resourcemanagerpb.CreateFolderRequest) bool {
			expected := &resourcemanagerpb.CreateFolderRequest{
				Folder: &resourcemanagerpb.Folder{
					Parent:      "my-parent",
					DisplayName: "my-display-name",
				},
			}
			return proto.Equal(expected, req)
		})).Return(nil, status.Error(codes.AlreadyExists, "already exists"))

		mockFoldersApi.On("ListFolders", mock.Anything, mock.Anything).
			Return(mockIter)

		client := newResourceManagerWithAPI(nil, mockFoldersApi)
		name, err := client.EnsureFolderExists(context.Background(), "my-display-name", "my-parent")

		require.NoError(t, err)
		require.Equal(t, "folders/123456789", name)
		mockFoldersApi.AssertExpectations(t)
		mockIter.AssertExpectations(t)
	})

	t.Run("returns error when folder not found in list", func(t *testing.T) {
		mockIter := &MockFolderIterator{}
		mockIter.On("Next").Return(nil, iterator.Done)

		mockFolders := &MockFoldersAPI{}
		mockFolders.On("CreateFolder", mock.Anything, mock.Anything).
			Return(nil, status.Error(codes.AlreadyExists, "already exists"))
		mockFolders.On("ListFolders", mock.Anything, mock.Anything).
			Return(mockIter)

		client := newResourceManagerWithAPI(nil, mockFolders)
		_, err := client.EnsureFolderExists(context.Background(), "my-display-name", "my-parent")

		require.Error(t, err)
		require.ErrorContains(t, err, "not found")
	})

	t.Run("returns weird error during iterate", func(t *testing.T) {
		mockIter := &MockFolderIterator{}
		mockIter.On("Next").Return(nil, fmt.Errorf("some error"))

		mockFolders := &MockFoldersAPI{}
		mockFolders.On("CreateFolder", mock.Anything, mock.Anything).
			Return(nil, status.Error(codes.AlreadyExists, "already exists"))
		mockFolders.On("ListFolders", mock.Anything, mock.Anything).
			Return(mockIter)

		client := newResourceManagerWithAPI(nil, mockFolders)
		_, err := client.EnsureFolderExists(context.Background(), "my-display-name", "my-parent")

		require.Error(t, err)
		require.ErrorContains(t, err, "some error")
	})

	t.Run("fail to create folder", func(t *testing.T) {
		mockFoldersApi := &MockFoldersAPI{}
		mockFoldersApi.On("CreateFolder", mock.Anything, mock.MatchedBy(func(req *resourcemanagerpb.CreateFolderRequest) bool {
			expected := &resourcemanagerpb.CreateFolderRequest{
				Folder: &resourcemanagerpb.Folder{
					Parent:      "my-parent",
					DisplayName: "my-display-name",
				},
			}
			return proto.Equal(expected, req)
		})).Return(nil, fmt.Errorf("some error"))

		client := newResourceManagerWithAPI(nil, mockFoldersApi)
		name, err := client.EnsureFolderExists(context.Background(), "my-display-name", "my-parent")

		require.Error(t, err)
		require.Empty(t, name)
		mockFoldersApi.AssertExpectations(t)
	})

	t.Run("failed while creating folder", func(t *testing.T) {
		mockOp := &MockCreateFolderOperation{}
		mockOp.On("Wait", mock.Anything).Return(nil, fmt.Errorf("some error"))

		mockFoldersApi := &MockFoldersAPI{}
		mockFoldersApi.On("CreateFolder", mock.Anything, mock.MatchedBy(func(req *resourcemanagerpb.CreateFolderRequest) bool {
			expected := &resourcemanagerpb.CreateFolderRequest{
				Folder: &resourcemanagerpb.Folder{
					Parent:      "my-parent",
					DisplayName: "my-display-name",
				},
			}
			return proto.Equal(expected, req)
		})).Return(mockOp, nil)

		client := newResourceManagerWithAPI(nil, mockFoldersApi)
		name, err := client.EnsureFolderExists(context.Background(), "my-display-name", "my-parent")

		require.Error(t, err)
		require.Empty(t, name)
		mockFoldersApi.AssertExpectations(t)
		mockOp.AssertExpectations(t)
	})
}

func TestEnsureProjectExists(t *testing.T) {
	t.Run("successfully get project", func(t *testing.T) {
		mockProjectsApi := &MockProjectsAPI{}
		mockProjectsApi.On("GetProject", mock.Anything, mock.MatchedBy(func(req *resourcemanagerpb.GetProjectRequest) bool {
			return req.Name == "projects/my-project"
		})).Return(&resourcemanagerpb.Project{Name: "projects/12345"}, nil)

		client := newResourceManagerWithAPI(mockProjectsApi, nil)
		projNumber, err := client.EnsureProjectExists(context.Background(), "my-project", "my-parent", "123456789")

		require.NoError(t, err)
		require.Equal(t, "12345", projNumber)
		mockProjectsApi.AssertExpectations(t)
	})

	t.Run("fail to get project with non-notfound error", func(t *testing.T) {
		mockProjectsApi := &MockProjectsAPI{}
		mockProjectsApi.On("GetProject", mock.Anything, mock.MatchedBy(func(req *resourcemanagerpb.GetProjectRequest) bool {
			return req.Name == "projects/my-project"
		})).Return(nil, fmt.Errorf("some error"))

		client := newResourceManagerWithAPI(mockProjectsApi, nil)
		projectNumber, err := client.EnsureProjectExists(context.Background(), "my-project", "my-parent", "123456789")

		require.Error(t, err)
		require.Empty(t, projectNumber)
		mockProjectsApi.AssertExpectations(t)
	})

	t.Run("successfully create project", func(t *testing.T) {
		mockOp := &MockCreateProjectOperation{}
		mockOp.On("Wait", mock.Anything).Return(&resourcemanagerpb.Project{
			ProjectId: "my-project",
			Name:      "projects/12345",
		}, nil)

		mockProjectsApi := &MockProjectsAPI{}
		mockProjectsApi.On("GetProject", mock.Anything, mock.MatchedBy(func(req *resourcemanagerpb.GetProjectRequest) bool {
			return req.Name == "projects/my-project"
		})).Return(nil, status.Error(codes.NotFound, "not found"))
		mockProjectsApi.On("CreateProject", mock.Anything, mock.MatchedBy(func(req *resourcemanagerpb.CreateProjectRequest) bool {
			return proto.Equal(&resourcemanagerpb.CreateProjectRequest{
				Project: &resourcemanagerpb.Project{
					ProjectId:   "my-project",
					DisplayName: "My Project",
					Parent:      "folders/123",
				},
			}, req)
		})).Return(mockOp, nil)

		client := newResourceManagerWithAPI(mockProjectsApi, nil)
		projectNumber, err := client.EnsureProjectExists(context.Background(), "my-project", "My Project", "folders/123")

		require.NoError(t, err)
		require.Equal(t, "12345", projectNumber)
		mockProjectsApi.AssertExpectations(t)
		mockOp.AssertExpectations(t)
	})

	t.Run("failed create project", func(t *testing.T) {
		mockProjectsApi := &MockProjectsAPI{}
		mockProjectsApi.On("GetProject", mock.Anything, mock.MatchedBy(func(req *resourcemanagerpb.GetProjectRequest) bool {
			return req.Name == "projects/my-project"
		})).Return(nil, status.Error(codes.NotFound, "not found"))
		mockProjectsApi.On("CreateProject", mock.Anything, mock.MatchedBy(func(req *resourcemanagerpb.CreateProjectRequest) bool {
			return proto.Equal(&resourcemanagerpb.CreateProjectRequest{
				Project: &resourcemanagerpb.Project{
					ProjectId:   "my-project",
					DisplayName: "My Project",
					Parent:      "folders/123",
				},
			}, req)
		})).Return(nil, fmt.Errorf("some error"))

		client := newResourceManagerWithAPI(mockProjectsApi, nil)
		projectNumber, err := client.EnsureProjectExists(context.Background(), "my-project", "My Project", "folders/123")

		require.Error(t, err)
		require.Empty(t, projectNumber)
		mockProjectsApi.AssertExpectations(t)
	})

	t.Run("failed create project wait", func(t *testing.T) {
		mockOp := &MockCreateProjectOperation{}
		mockOp.On("Wait", mock.Anything).Return(nil, fmt.Errorf("some error"))

		mockProjectsApi := &MockProjectsAPI{}
		mockProjectsApi.On("GetProject", mock.Anything, mock.MatchedBy(func(req *resourcemanagerpb.GetProjectRequest) bool {
			return req.Name == "projects/my-project"
		})).Return(nil, status.Error(codes.NotFound, "not found"))
		mockProjectsApi.On("CreateProject", mock.Anything, mock.MatchedBy(func(req *resourcemanagerpb.CreateProjectRequest) bool {
			return proto.Equal(&resourcemanagerpb.CreateProjectRequest{
				Project: &resourcemanagerpb.Project{
					ProjectId:   "my-project",
					DisplayName: "My Project",
					Parent:      "folders/123",
				},
			}, req)
		})).Return(mockOp, nil)

		client := newResourceManagerWithAPI(mockProjectsApi, nil)
		projectNumber, err := client.EnsureProjectExists(context.Background(), "my-project", "My Project", "folders/123")

		require.Error(t, err)
		require.Empty(t, projectNumber)
		mockProjectsApi.AssertExpectations(t)
		mockOp.AssertExpectations(t)
	})
}

func TestResourceManagerClose(t *testing.T) {
	t.Run("successfully close", func(t *testing.T) {
		mockFoldersApi := &MockFoldersAPI{}
		mockProjectsAPI := &MockProjectsAPI{}
		mockFoldersApi.On("Close").Return(nil)
		mockProjectsAPI.On("Close").Return(nil)

		client := newResourceManagerWithAPI(mockProjectsAPI, mockFoldersApi)
		err := client.Close()
		require.NoError(t, err)

		mockFoldersApi.AssertExpectations(t)
		mockProjectsAPI.AssertExpectations(t)
	})

	t.Run("fail close", func(t *testing.T) {
		mockFoldersApi := &MockFoldersAPI{}
		mockProjectsAPI := &MockProjectsAPI{}
		mockFoldersApi.On("Close").Return(fmt.Errorf("some error"))
		mockProjectsAPI.On("Close").Return(nil)

		client := newResourceManagerWithAPI(mockProjectsAPI, mockFoldersApi)
		err := client.Close()
		require.Error(t, err)

		mockFoldersApi.AssertExpectations(t)
		mockProjectsAPI.AssertExpectations(t)
	})
}

func TestNewResourceManangerClient(t *testing.T) {
	tests := []struct {
		name                       string
		setupResourceManagerClient func(ctx context.Context, opts ...option.ClientOption) (ResourceManagerClient, error)
		wantErr                    bool
	}{
		{
			name: "failed client creation",
			setupResourceManagerClient: func(ctx context.Context, opts ...option.ClientOption) (ResourceManagerClient, error) {
				client, err := NewResourceManagerClient(ctx, opts...)
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
			_, creationErr := tt.setupResourceManagerClient(context.Background(), option.WithCredentialsFile("/nonexistent/credentials.json"))
			if creationErr != nil && tt.wantErr {
				require.Error(t, creationErr)
			} else {
				require.NoError(t, creationErr)
			}
		})
	}
}
