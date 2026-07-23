package project

import (
	"context"
	"fmt"
	"testing"

	"cloud.google.com/go/orgpolicy/apiv2/orgpolicypb"
	gax "github.com/googleapis/gax-go/v2"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
	"google.golang.org/api/option"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

// MockServiceUsageAPI mocks the internal GCP client interface
type MockOrgPoliciesAPI struct {
	mock.Mock
}

func (m *MockOrgPoliciesAPI) GetPolicy(ctx context.Context, req *orgpolicypb.GetPolicyRequest, opts ...gax.CallOption) (*orgpolicypb.Policy, error) {
	args := m.Called(ctx, req)
	op, _ := args.Get(0).(*orgpolicypb.Policy)
	return op, args.Error(1)
}

func (m *MockOrgPoliciesAPI) CreatePolicy(ctx context.Context, req *orgpolicypb.CreatePolicyRequest, opts ...gax.CallOption) (*orgpolicypb.Policy, error) {
	args := m.Called(ctx, req)
	op, _ := args.Get(0).(*orgpolicypb.Policy)
	return op, args.Error(1)
}

func (m *MockOrgPoliciesAPI) UpdatePolicy(ctx context.Context, req *orgpolicypb.UpdatePolicyRequest, opts ...gax.CallOption) (*orgpolicypb.Policy, error) {
	args := m.Called(ctx, req)
	op, _ := args.Get(0).(*orgpolicypb.Policy)
	return op, args.Error(1)
}

func (m *MockOrgPoliciesAPI) Close() error {
	args := m.Called()
	return args.Error(0)
}

func TestEnforcePolicy(t *testing.T) {
	t.Run("successfully create enforced policy", func(t *testing.T) {
		mockAPI := &MockOrgPoliciesAPI{}
		mockAPI.On("GetPolicy", mock.Anything, mock.MatchedBy(func(req *orgpolicypb.GetPolicyRequest) bool {
			return req.Name == "projects/my-project/policies/constraints/iam.disableServiceAccountKeyCreation"
		})).Return(nil, status.Error(codes.NotFound, "policy not found"))

		mockAPI.On("CreatePolicy", mock.Anything, mock.MatchedBy(func(req *orgpolicypb.CreatePolicyRequest) bool {
			return req.Parent == "projects/my-project" &&
				req.Policy != nil &&
				req.Policy.Name == "projects/my-project/policies/constraints/iam.disableServiceAccountKeyCreation" &&
				len(req.Policy.Spec.GetRules()) == 1 &&
				req.Policy.Spec.GetRules()[0].GetEnforce() == true
		})).Return(nil, nil)

		client := newOrgPolicyClientWithAPI(mockAPI)
		err := client.EnforcePolicy(context.Background(), "projects/my-project", "constraints/iam.disableServiceAccountKeyCreation")
		require.NoError(t, err)

		mockAPI.AssertExpectations(t)
	})

	t.Run("successfully update enforced policy", func(t *testing.T) {
		mockAPI := &MockOrgPoliciesAPI{}
		mockAPI.On("GetPolicy", mock.Anything, mock.MatchedBy(func(req *orgpolicypb.GetPolicyRequest) bool {
			return req.Name == "projects/my-project/policies/constraints/iam.disableServiceAccountKeyCreation"
		})).Return(&orgpolicypb.Policy{Etag: "1"}, nil)

		mockAPI.On("UpdatePolicy", mock.Anything, mock.MatchedBy(func(req *orgpolicypb.UpdatePolicyRequest) bool {
			return req.Policy != nil &&
				req.Policy.Name == "projects/my-project/policies/constraints/iam.disableServiceAccountKeyCreation" &&
				len(req.Policy.Spec.GetRules()) == 1 &&
				req.Policy.Spec.GetRules()[0].GetEnforce() == true
		})).Return(nil, nil)

		client := newOrgPolicyClientWithAPI(mockAPI)
		err := client.EnforcePolicy(context.Background(), "projects/my-project", "constraints/iam.disableServiceAccountKeyCreation")
		require.NoError(t, err)

		mockAPI.AssertExpectations(t)
	})

	t.Run("Fail to create enforced policy", func(t *testing.T) {
		mockAPI := &MockOrgPoliciesAPI{}
		mockAPI.On("GetPolicy", mock.Anything, mock.MatchedBy(func(req *orgpolicypb.GetPolicyRequest) bool {
			return req.Name == "projects/my-project/policies/constraints/iam.disableServiceAccountKeyCreation"
		})).Return(nil, status.Error(codes.NotFound, "policy not found"))

		mockAPI.On("CreatePolicy", mock.Anything, mock.MatchedBy(func(req *orgpolicypb.CreatePolicyRequest) bool {
			return req.Parent == "projects/my-project" &&
				req.Policy != nil &&
				req.Policy.Name == "projects/my-project/policies/constraints/iam.disableServiceAccountKeyCreation" &&
				len(req.Policy.Spec.GetRules()) == 1 &&
				req.Policy.Spec.GetRules()[0].GetEnforce() == true
		})).Return(nil, fmt.Errorf("error"))

		client := newOrgPolicyClientWithAPI(mockAPI)
		err := client.EnforcePolicy(context.Background(), "projects/my-project", "constraints/iam.disableServiceAccountKeyCreation")
		require.Error(t, err)

		mockAPI.AssertExpectations(t)
	})

	t.Run("failed get enforced policy", func(t *testing.T) {
		mockAPI := &MockOrgPoliciesAPI{}
		mockAPI.On("GetPolicy", mock.Anything, mock.MatchedBy(func(req *orgpolicypb.GetPolicyRequest) bool {
			return req.Name == "projects/my-project/policies/constraints/iam.disableServiceAccountKeyCreation"
		})).Return(nil, status.Error(codes.FailedPrecondition, "precondition failed"))

		client := newOrgPolicyClientWithAPI(mockAPI)
		err := client.EnforcePolicy(context.Background(), "projects/my-project", "constraints/iam.disableServiceAccountKeyCreation")
		require.Error(t, err)

		mockAPI.AssertExpectations(t)
	})

	t.Run("failed create enforced policy", func(t *testing.T) {
		mockAPI := &MockOrgPoliciesAPI{}
		mockAPI.On("GetPolicy", mock.Anything, mock.MatchedBy(func(req *orgpolicypb.GetPolicyRequest) bool {
			return req.Name == "projects/my-project/policies/constraints/iam.disableServiceAccountKeyCreation"
		})).Return(nil, status.Error(codes.NotFound, "policy not found"))

		mockAPI.On("CreatePolicy", mock.Anything, mock.MatchedBy(func(req *orgpolicypb.CreatePolicyRequest) bool {
			return req.Parent == "projects/my-project" &&
				req.Policy != nil &&
				req.Policy.Name == "projects/my-project/policies/constraints/iam.disableServiceAccountKeyCreation" &&
				len(req.Policy.Spec.GetRules()) == 1 &&
				req.Policy.Spec.GetRules()[0].GetEnforce() == true
		})).Return(nil, status.Error(codes.FailedPrecondition, "precondition failed"))

		client := newOrgPolicyClientWithAPI(mockAPI)
		err := client.EnforcePolicy(context.Background(), "projects/my-project", "constraints/iam.disableServiceAccountKeyCreation")
		require.Error(t, err)

		mockAPI.AssertExpectations(t)
	})
}

func TestOrgPolicyClientClose(t *testing.T) {
	t.Run("successfully close", func(t *testing.T) {
		mockAPI := &MockOrgPoliciesAPI{}
		mockAPI.On("Close").Return(nil)

		client := newOrgPolicyClientWithAPI(mockAPI)
		err := client.Close()
		require.NoError(t, err)

		mockAPI.AssertExpectations(t)
	})

}

func TestNewOrgPoliciesClient(t *testing.T) {
	tests := []struct {
		name                   string
		setupOrgPoliciesClient func(ctx context.Context, opts ...option.ClientOption) (OrgPolicyClient, error)
		wantErr                bool
	}{
		{
			name: "failed client creation",
			setupOrgPoliciesClient: func(ctx context.Context, opts ...option.ClientOption) (OrgPolicyClient, error) {
				client, err := NewOrgPolicyClient(ctx, opts...)
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
			_, creationErr := tt.setupOrgPoliciesClient(context.Background(), option.WithCredentialsFile("/nonexistent/credentials.json"))
			if creationErr != nil && tt.wantErr {
				require.Error(t, creationErr)
			} else {
				require.NoError(t, creationErr)
			}
		})
	}
}
