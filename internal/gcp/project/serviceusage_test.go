package project

import (
	"context"
	"errors"
	"testing"

	serviceusagepb "cloud.google.com/go/serviceusage/apiv1/serviceusagepb"
	gax "github.com/googleapis/gax-go/v2"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
	"google.golang.org/api/option"
)

// MockBatchEnableOperation mocks the operation returned by BatchEnableServices
type MockBatchEnableOperation struct {
	mock.Mock
}

func (m *MockBatchEnableOperation) Wait(ctx context.Context, opts ...gax.CallOption) (*serviceusagepb.BatchEnableServicesResponse, error) {
	args := m.Called(ctx)
	return args.Get(0).(*serviceusagepb.BatchEnableServicesResponse), args.Error(1)
}

// MockServiceUsageAPI mocks the internal GCP client interface
type MockServiceUsageAPI struct {
	mock.Mock
}

func (m *MockServiceUsageAPI) BatchEnableServices(ctx context.Context, req *serviceusagepb.BatchEnableServicesRequest, opts ...gax.CallOption) (batchEnableOperation, error) {
	args := m.Called(ctx, req)
	// handle nil op case (when we want BatchEnableServices itself to error)
	op, _ := args.Get(0).(batchEnableOperation)
	return op, args.Error(1)
}

func (m *MockServiceUsageAPI) Close() error {
	args := m.Called()
	return args.Error(0)
}

func TestEnableAPIs(t *testing.T) {
	t.Run("successfully enables APIs and builds correct request", func(t *testing.T) {
		mockOp := &MockBatchEnableOperation{}
		mockOp.On("Wait", mock.Anything).Return(&serviceusagepb.BatchEnableServicesResponse{}, nil)

		mockAPI := &MockServiceUsageAPI{}
		mockAPI.On("BatchEnableServices", mock.Anything, mock.MatchedBy(func(req *serviceusagepb.BatchEnableServicesRequest) bool {
			return req.Parent == "projects/my-project" &&
				len(req.ServiceIds) == 2 &&
				req.ServiceIds[0] == "projects/my-project/services/run.googleapis.com" &&
				req.ServiceIds[1] == "projects/my-project/services/artifactregistry.googleapis.com"
		})).Return(mockOp, nil)

		client := newServiceUsageClientWithAPI(mockAPI)
		err := client.EnableAPIs(context.Background(), "my-project", []string{
			"run.googleapis.com",
			"artifactregistry.googleapis.com",
		})
		require.NoError(t, err)

		mockAPI.AssertExpectations(t)
		mockOp.AssertExpectations(t)
	})

	t.Run("returns error when BatchEnableServices fails", func(t *testing.T) {
		mockAPI := &MockServiceUsageAPI{}
		mockAPI.On("BatchEnableServices", mock.Anything, mock.Anything).
			Return(nil, errors.New("permission denied"))

		client := newServiceUsageClientWithAPI(mockAPI)
		err := client.EnableAPIs(context.Background(), "my-project", []string{"run.googleapis.com"})

		require.Error(t, err)
		assert.ErrorContains(t, err, "permission denied")
		mockAPI.AssertExpectations(t)
	})

	t.Run("returns error when Wait fails", func(t *testing.T) {
		mockOp := &MockBatchEnableOperation{}
		mockOp.On("Wait", mock.Anything).Return((*serviceusagepb.BatchEnableServicesResponse)(nil), errors.New("operation failed"))

		mockAPI := &MockServiceUsageAPI{}
		mockAPI.On("BatchEnableServices", mock.Anything, mock.Anything).
			Return(mockOp, nil)

		client := newServiceUsageClientWithAPI(mockAPI)
		err := client.EnableAPIs(context.Background(), "my-project", []string{"run.googleapis.com"})

		require.Error(t, err)
		assert.ErrorContains(t, err, "operation failed")
		mockAPI.AssertExpectations(t)
		mockOp.AssertExpectations(t)
	})

	t.Run("handles empty API list", func(t *testing.T) {
		mockOp := &MockBatchEnableOperation{}
		mockOp.On("Wait", mock.Anything).Return(&serviceusagepb.BatchEnableServicesResponse{}, nil)

		mockAPI := &MockServiceUsageAPI{}
		mockAPI.On("BatchEnableServices", mock.Anything, mock.Anything).
			Return(mockOp, nil)

		client := newServiceUsageClientWithAPI(mockAPI)
		err := client.EnableAPIs(context.Background(), "my-project", []string{})

		require.NoError(t, err)
		mockAPI.AssertExpectations(t)
	})
}

func TestClose(t *testing.T) {
	t.Run("successfully close", func(t *testing.T) {
		mockOp := &MockBatchEnableOperation{}
		mockAPI := &MockServiceUsageAPI{}
		mockAPI.On("Close").Return(nil)

		client := newServiceUsageClientWithAPI(mockAPI)
		err := client.Close()
		require.NoError(t, err)

		mockAPI.AssertExpectations(t)
		mockOp.AssertExpectations(t)
	})

}

func TestNewServiceUsageClient(t *testing.T) {
	tests := []struct {
		name                    string
		setupServiceUsageClient func(ctx context.Context, opts ...option.ClientOption) (ServiceUsageClient, error)
		wantErr                 bool
	}{
		{
			name: "failed client creation",
			setupServiceUsageClient: func(ctx context.Context, opts ...option.ClientOption) (ServiceUsageClient, error) {
				client, err := NewServiceUsageClient(ctx, opts...)
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
			_, creationErr := tt.setupServiceUsageClient(context.Background(), option.WithCredentialsFile("/nonexistent/credentials.json"))
			if creationErr != nil && tt.wantErr {
				require.Error(t, creationErr)
			} else {
				require.NoError(t, creationErr)
			}
		})
	}
}
