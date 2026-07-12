package project

import (
	"context"
	"fmt"
	"testing"

	"cloud.google.com/go/iam/admin/apiv1/adminpb"
	"cloud.google.com/go/iam/apiv1/iampb"
	"cloud.google.com/go/resourcemanager/apiv3/resourcemanagerpb"
	"github.com/googleapis/gax-go/v2"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
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

	t.Run("fail to create service accounts", func(t *testing.T) {
		mockIamApi := &MockIamAPI{}
		mockIamApi.On("GetServiceAccount", mock.Anything, mock.MatchedBy(func(req *adminpb.GetServiceAccountRequest) bool {
			return req.Name == "projects/my-project/serviceAccounts/sa-cldrun@my-project.iam.gserviceaccount.com"
		})).Return(nil, status.Error(codes.NotFound, "policy not found"))

		mockIamApi.On("CreateServiceAccount", mock.Anything, mock.MatchedBy(func(req *adminpb.CreateServiceAccountRequest) bool {
			return req.Name == "projects/my-project" &&
				req.AccountId == "sa-cldrun" &&
				req.ServiceAccount.DisplayName == "sa-cldrun"
		})).Return(nil, fmt.Errorf("some error"))

		client := newIamClientWithAPI(mockIamApi, &MockProjectsAPI{})
		email, err := client.EnsureServiceAccountExists(context.Background(), "my-project", "sa-cldrun", "sa-cldrun")
		require.Error(t, err)
		require.Empty(t, email)

		mockIamApi.AssertExpectations(t)
	})

	t.Run("successfully find service accounts", func(t *testing.T) {
		mockIamApi := &MockIamAPI{}
		mockIamApi.On("GetServiceAccount", mock.Anything, mock.MatchedBy(func(req *adminpb.GetServiceAccountRequest) bool {
			return req.Name == "projects/my-project/serviceAccounts/sa-cldrun@my-project.iam.gserviceaccount.com"
		})).Return(&adminpb.ServiceAccount{Email: "sa-cldrun@my-project.iam.gserviceaccount.com"}, nil)

		client := newIamClientWithAPI(mockIamApi, &MockProjectsAPI{})
		email, err := client.EnsureServiceAccountExists(context.Background(), "my-project", "sa-cldrun", "sa-cldrun")
		require.NoError(t, err)
		require.Equal(t, "sa-cldrun@my-project.iam.gserviceaccount.com", email)

		mockIamApi.AssertExpectations(t)
	})

	t.Run("fail on find service accounts", func(t *testing.T) {
		mockIamApi := &MockIamAPI{}
		mockIamApi.On("GetServiceAccount", mock.Anything, mock.MatchedBy(func(req *adminpb.GetServiceAccountRequest) bool {
			return req.Name == "projects/my-project/serviceAccounts/sa-cldrun@my-project.iam.gserviceaccount.com"
		})).Return(nil, fmt.Errorf("some error"))

		client := newIamClientWithAPI(mockIamApi, &MockProjectsAPI{})
		email, err := client.EnsureServiceAccountExists(context.Background(), "my-project", "sa-cldrun", "sa-cldrun")
		require.Error(t, err)
		require.Empty(t, email)

		mockIamApi.AssertExpectations(t)
	})

}

func TestEnsureProjectExists(t *testing.T) {
	t.Run("fail to get iam policy", func(t *testing.T) {
		mockIamApi := &MockIamAPI{}
		mockProjectsAPI := &MockProjectsAPI{}
		mockProjectsAPI.On("GetIamPolicy", mock.Anything, mock.MatchedBy(func(req *iampb.GetIamPolicyRequest) bool {
			return req.Resource == "projects/my-project"
		})).Return(nil, fmt.Errorf("some error"))

		client := newIamClientWithAPI(mockIamApi, mockProjectsAPI)
		err := client.BindProjectRoles(context.Background(), "my-project", "sa-cldrun@my-project.iam.gserviceaccount.com", []string{"my-role"})
		require.Error(t, err)

		mockIamApi.AssertExpectations(t)
	})

}

func TestResourceManagerClose(t *testing.T) {
	t.Run("successfully close", func(t *testing.T) {
		mockIamApi := &MockIamAPI{}
		mockProjectsAPI := &MockProjectsAPI{}
		mockIamApi.On("Close").Return(nil)
		mockProjectsAPI.On("Close").Return(nil)

		client := newIamClientWithAPI(mockIamApi, mockProjectsAPI)
		err := client.Close()
		require.NoError(t, err)

		mockIamApi.AssertExpectations(t)
		mockProjectsAPI.AssertExpectations(t)
	})

	t.Run("fail close", func(t *testing.T) {
		mockIamApi := &MockIamAPI{}
		mockProjectsAPI := &MockProjectsAPI{}
		mockIamApi.On("Close").Return(fmt.Errorf("error"))

		client := newIamClientWithAPI(mockIamApi, mockProjectsAPI)
		err := client.Close()
		require.Error(t, err)

		mockIamApi.AssertExpectations(t)
		mockProjectsAPI.AssertExpectations(t)
	})
}
