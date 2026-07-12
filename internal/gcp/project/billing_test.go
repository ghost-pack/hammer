package project

import (
	"context"
	"fmt"
	"testing"

	"cloud.google.com/go/billing/apiv1/billingpb"
	gax "github.com/googleapis/gax-go/v2"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
)

// MockServiceUsageAPI mocks the internal GCP client interface
type MockBillingAPI struct {
	mock.Mock
}

func (m *MockBillingAPI) UpdateProjectBillingInfo(ctx context.Context, req *billingpb.UpdateProjectBillingInfoRequest, opts ...gax.CallOption) (*billingpb.ProjectBillingInfo, error) {
	args := m.Called(ctx, req)
	op, _ := args.Get(0).(*billingpb.ProjectBillingInfo)
	return op, args.Error(1)
}

func (m *MockBillingAPI) Close() error {
	args := m.Called()
	return args.Error(0)
}

func TestLinkBillingAccount(t *testing.T) {
	t.Run("successfully link billing account", func(t *testing.T) {
		mockAPI := &MockBillingAPI{}
		mockAPI.On("UpdateProjectBillingInfo", mock.Anything, mock.MatchedBy(func(req *billingpb.UpdateProjectBillingInfoRequest) bool {
			return req.Name == "projects/my-project" &&
				req.ProjectBillingInfo.BillingAccountName == "billingAccounts/my-billing-account"
		})).Return(nil, nil)

		client := newBillingClientWithAPI(mockAPI)
		err := client.LinkBillingAccount(context.Background(), "my-project", "my-billing-account")
		require.NoError(t, err)

		mockAPI.AssertExpectations(t)
	})

	t.Run("fail to link billing account", func(t *testing.T) {
		mockAPI := &MockBillingAPI{}
		mockAPI.On("UpdateProjectBillingInfo", mock.Anything, mock.MatchedBy(func(req *billingpb.UpdateProjectBillingInfoRequest) bool {
			return req.Name == "projects/my-project" &&
				req.ProjectBillingInfo.BillingAccountName == "billingAccounts/my-billing-account"
		})).Return(nil, fmt.Errorf("error"))

		client := newBillingClientWithAPI(mockAPI)
		err := client.LinkBillingAccount(context.Background(), "my-project", "my-billing-account")
		require.Error(t, err)

		mockAPI.AssertExpectations(t)
	})
}

func TestBillingClientClose(t *testing.T) {
	t.Run("successfully close", func(t *testing.T) {
		mockAPI := &MockBillingAPI{}
		mockAPI.On("Close").Return(nil)

		client := newBillingClientWithAPI(mockAPI)
		err := client.Close()
		require.NoError(t, err)

		mockAPI.AssertExpectations(t)
	})

}
