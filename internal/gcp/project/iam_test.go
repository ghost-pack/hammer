package project

import (
	"context"
	"fmt"
	"testing"

	"cloud.google.com/go/iam/admin/apiv1/adminpb"
	"cloud.google.com/go/iam/apiv1/iampb"
	"cloud.google.com/go/resourcemanager/apiv3/resourcemanagerpb"
	gax "github.com/googleapis/gax-go/v2"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
	"google.golang.org/api/option"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
	"google.golang.org/protobuf/proto"
)

// MockServiceUsageAPI mocks the internal GCP client interface
type MockIamAPI struct {
	mock.Mock
}

func (m *MockIamAPI) GetServiceAccount(ctx context.Context, req *adminpb.GetServiceAccountRequest, opts ...gax.CallOption) (*adminpb.ServiceAccount, error) {
	args := m.Called(ctx, req)
	op, _ := args.Get(0).(*adminpb.ServiceAccount)
	return op, args.Error(1)
}

func (m *MockIamAPI) CreateServiceAccount(ctx context.Context, req *adminpb.CreateServiceAccountRequest, opts ...gax.CallOption) (*adminpb.ServiceAccount, error) {
	args := m.Called(ctx, req)
	op, _ := args.Get(0).(*adminpb.ServiceAccount)
	return op, args.Error(1)
}

func (m *MockIamAPI) Close() error {
	args := m.Called()
	return args.Error(0)
}

type MockCreateProjectOperation struct {
	mock.Mock
}

func (m *MockCreateProjectOperation) Wait(ctx context.Context, opts ...gax.CallOption) (*resourcemanagerpb.Project, error) {
	args := m.Called(ctx)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*resourcemanagerpb.Project), args.Error(1)
}

type MockProjectsAPI struct {
	mock.Mock
}

func (m *MockProjectsAPI) GetIamPolicy(ctx context.Context, req *iampb.GetIamPolicyRequest, opts ...gax.CallOption) (*iampb.Policy, error) {
	args := m.Called(ctx, req)
	op, _ := args.Get(0).(*iampb.Policy)
	return op, args.Error(1)
}

func (m *MockProjectsAPI) SetIamPolicy(ctx context.Context, req *iampb.SetIamPolicyRequest, opts ...gax.CallOption) (*iampb.Policy, error) {
	args := m.Called(ctx, req)
	op, _ := args.Get(0).(*iampb.Policy)
	return op, args.Error(1)
}

func (m *MockProjectsAPI) GetProject(ctx context.Context, req *resourcemanagerpb.GetProjectRequest, opts ...gax.CallOption) (*resourcemanagerpb.Project, error) {
	args := m.Called(ctx, req)
	op, _ := args.Get(0).(*resourcemanagerpb.Project)
	return op, args.Error(1)
}

func (m *MockProjectsAPI) CreateProject(ctx context.Context, req *resourcemanagerpb.CreateProjectRequest, opts ...gax.CallOption) (createProjectOperation, error) {
	args := m.Called(ctx, req)
	op, _ := args.Get(0).(createProjectOperation)
	return op, args.Error(1)
}

func (m *MockProjectsAPI) Close() error {
	args := m.Called()
	return args.Error(0)
}

func TestEnsureServiceAccountExists(t *testing.T) {
	t.Run("successfully create service accounts", func(t *testing.T) {
		mockIamApi := &MockIamAPI{}
		mockIamApi.On("GetServiceAccount", mock.Anything, mock.MatchedBy(func(req *adminpb.GetServiceAccountRequest) bool {
			return req.Name == "projects/my-project/serviceAccounts/sa-cldrun@my-project.iam.gserviceaccount.com"
		})).Return(nil, status.Error(codes.NotFound, "policy not found"))

		mockIamApi.On("CreateServiceAccount", mock.Anything, mock.MatchedBy(func(req *adminpb.CreateServiceAccountRequest) bool {
			return req.Name == "projects/my-project" &&
				req.AccountId == "sa-cldrun" &&
				req.ServiceAccount.DisplayName == "sa-cldrun"
		})).Return(&adminpb.ServiceAccount{Email: "sa-cldrun@my-project.iam.gserviceaccount.com"}, nil)

		client := newIamClientWithAPI(mockIamApi, &MockProjectsAPI{})
		email, err := client.EnsureServiceAccountExists(context.Background(), "my-project", "sa-cldrun", "sa-cldrun")
		require.NoError(t, err)
		require.Equal(t, "sa-cldrun@my-project.iam.gserviceaccount.com", email)

		mockIamApi.AssertExpectations(t)
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

func TestBindProjectRoles(t *testing.T) {
	t.Run("successfully bind new role", func(t *testing.T) {
		mockIamApi := &MockIamAPI{}
		mockProjectsAPI := &MockProjectsAPI{}
		mockProjectsAPI.On("GetIamPolicy", mock.Anything, mock.MatchedBy(func(req *iampb.GetIamPolicyRequest) bool {
			return req.Resource == "projects/my-project"
		})).Return(&iampb.Policy{}, nil)

		mockProjectsAPI.On("SetIamPolicy", mock.Anything, mock.MatchedBy(func(req *iampb.SetIamPolicyRequest) bool {
			expected := &iampb.Policy{Bindings: []*iampb.Binding{
				{Role: "my-role", Members: []string{"serviceAccount:sa-cldrun@my-project.iam.gserviceaccount.com"}},
			}}
			return req.Resource == "projects/my-project" && proto.Equal(req.Policy, expected)
		})).Return(&iampb.Policy{Bindings: []*iampb.Binding{
			{Role: "my-role", Members: []string{"serviceAccount:sa-cldrun@my-project.iam.gserviceaccount.com"}},
		}}, nil)

		client := newIamClientWithAPI(mockIamApi, mockProjectsAPI)
		err := client.BindProjectRoles(context.Background(), "my-project", "sa-cldrun@my-project.iam.gserviceaccount.com", []string{"my-role"})
		require.NoError(t, err)

		mockIamApi.AssertExpectations(t)
	})

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

	t.Run("fail to set IAM policy", func(t *testing.T) {
		mockIamApi := &MockIamAPI{}
		mockProjectsAPI := &MockProjectsAPI{}
		mockProjectsAPI.On("GetIamPolicy", mock.Anything, mock.MatchedBy(func(req *iampb.GetIamPolicyRequest) bool {
			return req.Resource == "projects/my-project"
		})).Return(&iampb.Policy{}, nil)

		mockProjectsAPI.On("SetIamPolicy", mock.Anything, mock.MatchedBy(func(req *iampb.SetIamPolicyRequest) bool {
			expected := &iampb.Policy{Bindings: []*iampb.Binding{
				{Role: "my-role", Members: []string{"serviceAccount:sa-cldrun@my-project.iam.gserviceaccount.com"}},
			}}
			return req.Resource == "projects/my-project" && proto.Equal(req.Policy, expected)
		})).Return(nil, fmt.Errorf("some error"))

		client := newIamClientWithAPI(mockIamApi, mockProjectsAPI)
		err := client.BindProjectRoles(context.Background(), "my-project", "sa-cldrun@my-project.iam.gserviceaccount.com", []string{"my-role"})
		require.Error(t, err)

		mockIamApi.AssertExpectations(t)
	})

	t.Run("successfully bind new member on existing role", func(t *testing.T) {
		mockIamApi := &MockIamAPI{}
		mockProjectsAPI := &MockProjectsAPI{}
		mockProjectsAPI.On("GetIamPolicy", mock.Anything, mock.MatchedBy(func(req *iampb.GetIamPolicyRequest) bool {
			return req.Resource == "projects/my-project"
		})).Return(&iampb.Policy{Bindings: []*iampb.Binding{
			{Role: "my-role", Members: []string{"serviceAccount:sa-cldrun-2@my-project.iam.gserviceaccount.com"}},
		}}, nil)

		mockProjectsAPI.On("SetIamPolicy", mock.Anything, mock.MatchedBy(func(req *iampb.SetIamPolicyRequest) bool {
			expected := &iampb.Policy{Bindings: []*iampb.Binding{
				{Role: "my-role", Members: []string{"serviceAccount:sa-cldrun-2@my-project.iam.gserviceaccount.com", "serviceAccount:sa-cldrun@my-project.iam.gserviceaccount.com"}},
			}}
			return req.Resource == "projects/my-project" && proto.Equal(req.Policy, expected)
		})).Return(&iampb.Policy{Bindings: []*iampb.Binding{
			{Role: "my-role", Members: []string{"serviceAccount:sa-cldrun-2@my-project.iam.gserviceaccount.com", "serviceAccount:sa-cldrun@my-project.iam.gserviceaccount.com"}},
		}}, nil)

		client := newIamClientWithAPI(mockIamApi, mockProjectsAPI)
		err := client.BindProjectRoles(context.Background(), "my-project", "sa-cldrun@my-project.iam.gserviceaccount.com", []string{"my-role"})
		require.NoError(t, err)

		mockIamApi.AssertExpectations(t)
	})

	t.Run("successfully no-op on existing member of existing role", func(t *testing.T) {
		mockIamApi := &MockIamAPI{}
		mockProjectsAPI := &MockProjectsAPI{}
		mockProjectsAPI.On("GetIamPolicy", mock.Anything, mock.MatchedBy(func(req *iampb.GetIamPolicyRequest) bool {
			return req.Resource == "projects/my-project"
		})).Return(&iampb.Policy{Bindings: []*iampb.Binding{
			{Role: "my-role", Members: []string{"serviceAccount:sa-cldrun@my-project.iam.gserviceaccount.com"}},
		}}, nil)

		mockProjectsAPI.On("SetIamPolicy", mock.Anything, mock.MatchedBy(func(req *iampb.SetIamPolicyRequest) bool {
			expected := &iampb.Policy{Bindings: []*iampb.Binding{
				{Role: "my-role", Members: []string{"serviceAccount:sa-cldrun@my-project.iam.gserviceaccount.com"}},
			}}
			return req.Resource == "projects/my-project" && proto.Equal(req.Policy, expected)
		})).Return(&iampb.Policy{Bindings: []*iampb.Binding{
			{Role: "my-role", Members: []string{"serviceAccount:sa-cldrun@my-project.iam.gserviceaccount.com"}},
		}}, nil)

		client := newIamClientWithAPI(mockIamApi, mockProjectsAPI)
		err := client.BindProjectRoles(context.Background(), "my-project", "sa-cldrun@my-project.iam.gserviceaccount.com", []string{"my-role"})
		require.NoError(t, err)

		mockIamApi.AssertExpectations(t)
	})
}

func TestUnbindProjectRoles(t *testing.T) {
	t.Run("removes sole member and removes binding entirely", func(t *testing.T) {
		mockIamApi := &MockIamAPI{}
		mockProjectsAPI := &MockProjectsAPI{}
		mockProjectsAPI.On("GetIamPolicy", mock.Anything, mock.MatchedBy(func(req *iampb.GetIamPolicyRequest) bool {
			return req.Resource == "projects/my-project"
		})).Return(&iampb.Policy{Bindings: []*iampb.Binding{
			{Role: "my-role", Members: []string{"serviceAccount:sa-cldrun@my-project.iam.gserviceaccount.com"}},
		}}, nil)

		mockProjectsAPI.On("SetIamPolicy", mock.Anything, mock.MatchedBy(func(req *iampb.SetIamPolicyRequest) bool {
			// binding removed entirely since it had only one member
			expected := &iampb.Policy{Bindings: []*iampb.Binding{}}
			return req.Resource == "projects/my-project" && proto.Equal(req.Policy, expected)
		})).Return(&iampb.Policy{}, nil)

		client := newIamClientWithAPI(mockIamApi, mockProjectsAPI)
		err := client.UnbindProjectRoles(context.Background(), "my-project", "sa-cldrun@my-project.iam.gserviceaccount.com", []string{"my-role"})
		require.NoError(t, err)
		mockProjectsAPI.AssertExpectations(t)
		mockIamApi.AssertExpectations(t)
	})

	t.Run("removes member but keeps other members in binding", func(t *testing.T) {
		mockIamApi := &MockIamAPI{}
		mockProjectsAPI := &MockProjectsAPI{}
		mockProjectsAPI.On("GetIamPolicy", mock.Anything, mock.MatchedBy(func(req *iampb.GetIamPolicyRequest) bool {
			return req.Resource == "projects/my-project"
		})).Return(&iampb.Policy{Bindings: []*iampb.Binding{
			{Role: "my-role", Members: []string{
				"serviceAccount:sa-cldrun@my-project.iam.gserviceaccount.com",
				"serviceAccount:sa-other@my-project.iam.gserviceaccount.com",
			}},
		}}, nil)

		mockProjectsAPI.On("SetIamPolicy", mock.Anything, mock.MatchedBy(func(req *iampb.SetIamPolicyRequest) bool {
			// only the other member should remain
			expected := &iampb.Policy{Bindings: []*iampb.Binding{
				{Role: "my-role", Members: []string{"serviceAccount:sa-other@my-project.iam.gserviceaccount.com"}},
			}}
			return req.Resource == "projects/my-project" && proto.Equal(req.Policy, expected)
		})).Return(&iampb.Policy{}, nil)

		client := newIamClientWithAPI(mockIamApi, mockProjectsAPI)
		err := client.UnbindProjectRoles(context.Background(), "my-project", "sa-cldrun@my-project.iam.gserviceaccount.com", []string{"my-role"})
		require.NoError(t, err)
		mockProjectsAPI.AssertExpectations(t)
		mockIamApi.AssertExpectations(t)
	})

	t.Run("no-op when role does not exist in policy", func(t *testing.T) {
		mockIamApi := &MockIamAPI{}
		mockProjectsAPI := &MockProjectsAPI{}
		mockProjectsAPI.On("GetIamPolicy", mock.Anything, mock.MatchedBy(func(req *iampb.GetIamPolicyRequest) bool {
			return req.Resource == "projects/my-project"
		})).Return(&iampb.Policy{}, nil)

		mockProjectsAPI.On("SetIamPolicy", mock.Anything, mock.MatchedBy(func(req *iampb.SetIamPolicyRequest) bool {
			return req.Resource == "projects/my-project" && proto.Equal(req.Policy, &iampb.Policy{})
		})).Return(&iampb.Policy{}, nil)

		client := newIamClientWithAPI(mockIamApi, mockProjectsAPI)
		err := client.UnbindProjectRoles(context.Background(), "my-project", "sa-cldrun@my-project.iam.gserviceaccount.com", []string{"my-role"})
		require.NoError(t, err)
		mockProjectsAPI.AssertExpectations(t)
		mockIamApi.AssertExpectations(t)
	})

	t.Run("no-op when member not in role", func(t *testing.T) {
		mockIamApi := &MockIamAPI{}
		mockProjectsAPI := &MockProjectsAPI{}
		existingPolicy := &iampb.Policy{Bindings: []*iampb.Binding{
			{Role: "my-role", Members: []string{"serviceAccount:sa-other@my-project.iam.gserviceaccount.com"}},
		}}
		mockProjectsAPI.On("GetIamPolicy", mock.Anything, mock.MatchedBy(func(req *iampb.GetIamPolicyRequest) bool {
			return req.Resource == "projects/my-project"
		})).Return(existingPolicy, nil)

		mockProjectsAPI.On("SetIamPolicy", mock.Anything, mock.MatchedBy(func(req *iampb.SetIamPolicyRequest) bool {
			// policy unchanged — sa-other is untouched
			expected := &iampb.Policy{Bindings: []*iampb.Binding{
				{Role: "my-role", Members: []string{"serviceAccount:sa-other@my-project.iam.gserviceaccount.com"}},
			}}
			return req.Resource == "projects/my-project" && proto.Equal(req.Policy, expected)
		})).Return(&iampb.Policy{}, nil)

		client := newIamClientWithAPI(mockIamApi, mockProjectsAPI)
		err := client.UnbindProjectRoles(context.Background(), "my-project", "sa-cldrun@my-project.iam.gserviceaccount.com", []string{"my-role"})
		require.NoError(t, err)
		mockProjectsAPI.AssertExpectations(t)
		mockIamApi.AssertExpectations(t)
	})

	t.Run("fail to get iam policy", func(t *testing.T) {
		mockIamApi := &MockIamAPI{}
		mockProjectsAPI := &MockProjectsAPI{}
		mockProjectsAPI.On("GetIamPolicy", mock.Anything, mock.MatchedBy(func(req *iampb.GetIamPolicyRequest) bool {
			return req.Resource == "projects/my-project"
		})).Return(nil, fmt.Errorf("some error"))

		client := newIamClientWithAPI(mockIamApi, mockProjectsAPI)
		err := client.UnbindProjectRoles(context.Background(), "my-project", "sa-cldrun@my-project.iam.gserviceaccount.com", []string{"my-role"})
		require.Error(t, err)
		require.ErrorContains(t, err, "getting IAM policy")
		mockProjectsAPI.AssertNotCalled(t, "SetIamPolicy", mock.Anything, mock.Anything)
		mockIamApi.AssertExpectations(t)
	})

	t.Run("fail to set IAM policy", func(t *testing.T) {
		mockIamApi := &MockIamAPI{}
		mockProjectsAPI := &MockProjectsAPI{}
		mockProjectsAPI.On("GetIamPolicy", mock.Anything, mock.MatchedBy(func(req *iampb.GetIamPolicyRequest) bool {
			return req.Resource == "projects/my-project"
		})).Return(&iampb.Policy{Bindings: []*iampb.Binding{
			{Role: "my-role", Members: []string{"serviceAccount:sa-cldrun@my-project.iam.gserviceaccount.com"}},
		}}, nil)

		mockProjectsAPI.On("SetIamPolicy", mock.Anything, mock.Anything).
			Return(nil, fmt.Errorf("some error"))

		client := newIamClientWithAPI(mockIamApi, mockProjectsAPI)
		err := client.UnbindProjectRoles(context.Background(), "my-project", "sa-cldrun@my-project.iam.gserviceaccount.com", []string{"my-role"})
		require.Error(t, err)
		require.ErrorContains(t, err, "removing IAM roles")
		mockProjectsAPI.AssertExpectations(t)
		mockIamApi.AssertExpectations(t)
	})
}

func TestIAMClientClose(t *testing.T) {
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

func TestNewIamClient(t *testing.T) {
	tests := []struct {
		name           string
		setupIamClient func(ctx context.Context, opts ...option.ClientOption) (IAMClient, error)
		wantErr        bool
	}{
		{
			name: "failed client creation",
			setupIamClient: func(ctx context.Context, opts ...option.ClientOption) (IAMClient, error) {
				client, err := NewIAMClient(ctx, opts...)
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
			_, creationErr := tt.setupIamClient(context.Background(), option.WithCredentialsFile("/nonexistent/credentials.json"))
			if creationErr != nil && tt.wantErr {
				require.Error(t, creationErr)
			} else {
				require.NoError(t, creationErr)
			}
		})
	}
}
