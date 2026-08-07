package gcp

import (
	"bytes"
	"context"
	"fmt"
	"reflect"
	"testing"

	"cloud.google.com/go/iam/apiv1/iampb"
	"cloud.google.com/go/pubsub/v2/apiv1/pubsubpb"
	gax "github.com/googleapis/gax-go/v2"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
	"google.golang.org/api/option"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

// MockServiceUsageAPI mocks the internal GCP client interface
type MockPubsubAPI struct {
	mock.Mock
}

func (m *MockPubsubAPI) GetTopic(ctx context.Context, req *pubsubpb.GetTopicRequest, opts ...gax.CallOption) (*pubsubpb.Topic, error) {
	args := m.Called(ctx, req)
	op, _ := args.Get(0).(*pubsubpb.Topic)
	return op, args.Error(1)
}

func (m *MockPubsubAPI) CreateTopic(ctx context.Context, req *pubsubpb.Topic, opts ...gax.CallOption) (*pubsubpb.Topic, error) {
	args := m.Called(ctx, req)
	op, _ := args.Get(0).(*pubsubpb.Topic)
	return op, args.Error(1)
}

func (m *MockPubsubAPI) DeleteTopic(ctx context.Context, req *pubsubpb.DeleteTopicRequest, opts ...gax.CallOption) error {
	args := m.Called(ctx, req)
	return args.Error(0)
}

func (m *MockPubsubAPI) Publish(ctx context.Context, req *pubsubpb.PublishRequest, opts ...gax.CallOption) (*pubsubpb.PublishResponse, error) {
	args := m.Called(ctx, req)
	op, _ := args.Get(0).(*pubsubpb.PublishResponse)
	return op, args.Error(1)
}

func (m *MockPubsubAPI) GetIamPolicy(ctx context.Context, req *iampb.GetIamPolicyRequest, opts ...gax.CallOption) (*iampb.Policy, error) {
	args := m.Called(ctx, req)
	op, _ := args.Get(0).(*iampb.Policy)
	return op, args.Error(1)
}

func (m *MockPubsubAPI) SetIamPolicy(ctx context.Context, req *iampb.SetIamPolicyRequest, opts ...gax.CallOption) (*iampb.Policy, error) {
	args := m.Called(ctx, req)
	op, _ := args.Get(0).(*iampb.Policy)
	return op, args.Error(1)
}

func (m *MockPubsubAPI) Close() error {
	args := m.Called()
	return args.Error(0)
}

func TestEnsureTopic(t *testing.T) {
	t.Run("successfully create topic", func(t *testing.T) {
		mockAPI := &MockPubsubAPI{}
		mockAPI.On("GetTopic", mock.Anything, mock.MatchedBy(func(req *pubsubpb.GetTopicRequest) bool {
			return req.Topic == "projects/my-project/topics/my-topic"
		})).Return(nil, status.Error(codes.NotFound, "repo not found"))

		mockAPI.On("CreateTopic", mock.Anything, mock.MatchedBy(func(req *pubsubpb.Topic) bool {
			return req.Name == "projects/my-project/topics/my-topic"
		})).Return(nil, nil)

		mockAPI.On("GetIamPolicy", mock.Anything, mock.MatchedBy(func(req *iampb.GetIamPolicyRequest) bool {
			return req.Resource == "projects/my-project/topics/my-topic"
		})).Return(&iampb.Policy{}, nil)

		mockAPI.On("SetIamPolicy", mock.Anything, mock.MatchedBy(func(req *iampb.SetIamPolicyRequest) bool {
			return req.Resource == "projects/my-project/topics/my-topic" &&
				req.Policy.Bindings[0].Role == "roles/pubsub.publisher" &&
				req.Policy.Bindings[0].Members[0] == "serviceAccount:sa-pipeline@my-project.iam.gserviceaccount.com"
		})).Return(&iampb.Policy{}, nil)

		client := newPubsubClientWithAPI(mockAPI)
		err := client.EnsureTopic(context.Background(), "my-project", "my-topic", "sa-pipeline@my-project.iam.gserviceaccount.com")
		require.NoError(t, err)

		mockAPI.AssertExpectations(t)
	})

	t.Run("error when setting iam policy", func(t *testing.T) {
		mockAPI := &MockPubsubAPI{}
		mockAPI.On("GetTopic", mock.Anything, mock.MatchedBy(func(req *pubsubpb.GetTopicRequest) bool {
			return req.Topic == "projects/my-project/topics/my-topic"
		})).Return(nil, status.Error(codes.NotFound, "repo not found"))

		mockAPI.On("CreateTopic", mock.Anything, mock.MatchedBy(func(req *pubsubpb.Topic) bool {
			return req.Name == "projects/my-project/topics/my-topic"
		})).Return(nil, nil)

		mockAPI.On("GetIamPolicy", mock.Anything, mock.MatchedBy(func(req *iampb.GetIamPolicyRequest) bool {
			return req.Resource == "projects/my-project/topics/my-topic"
		})).Return(&iampb.Policy{}, nil)

		mockAPI.On("SetIamPolicy", mock.Anything, mock.MatchedBy(func(req *iampb.SetIamPolicyRequest) bool {
			return req.Resource == "projects/my-project/topics/my-topic" &&
				req.Policy.Bindings[0].Role == "roles/pubsub.publisher" &&
				req.Policy.Bindings[0].Members[0] == "serviceAccount:sa-pipeline@my-project.iam.gserviceaccount.com"
		})).Return(nil, fmt.Errorf("iam policy error"))

		client := newPubsubClientWithAPI(mockAPI)
		err := client.EnsureTopic(context.Background(), "my-project", "my-topic", "sa-pipeline@my-project.iam.gserviceaccount.com")
		require.Error(t, err)

		mockAPI.AssertExpectations(t)
	})

	t.Run("error when getting iam policy", func(t *testing.T) {
		mockAPI := &MockPubsubAPI{}
		mockAPI.On("GetTopic", mock.Anything, mock.MatchedBy(func(req *pubsubpb.GetTopicRequest) bool {
			return req.Topic == "projects/my-project/topics/my-topic"
		})).Return(nil, status.Error(codes.NotFound, "repo not found"))

		mockAPI.On("CreateTopic", mock.Anything, mock.MatchedBy(func(req *pubsubpb.Topic) bool {
			return req.Name == "projects/my-project/topics/my-topic"
		})).Return(nil, nil)

		mockAPI.On("GetIamPolicy", mock.Anything, mock.MatchedBy(func(req *iampb.GetIamPolicyRequest) bool {
			return req.Resource == "projects/my-project/topics/my-topic"
		})).Return(nil, fmt.Errorf("iam policy error"))

		client := newPubsubClientWithAPI(mockAPI)
		err := client.EnsureTopic(context.Background(), "my-project", "my-topic", "sa-pipeline@my-project.iam.gserviceaccount.com")
		require.Error(t, err)

		mockAPI.AssertExpectations(t)
	})

	t.Run("error when creating topic", func(t *testing.T) {
		mockAPI := &MockPubsubAPI{}
		mockAPI.On("GetTopic", mock.Anything, mock.MatchedBy(func(req *pubsubpb.GetTopicRequest) bool {
			return req.Topic == "projects/my-project/topics/my-topic"
		})).Return(nil, status.Error(codes.NotFound, "repo not found"))

		mockAPI.On("CreateTopic", mock.Anything, mock.MatchedBy(func(req *pubsubpb.Topic) bool {
			return req.Name == "projects/my-project/topics/my-topic"
		})).Return(nil, fmt.Errorf("iam policy error"))

		client := newPubsubClientWithAPI(mockAPI)
		err := client.EnsureTopic(context.Background(), "my-project", "my-topic", "sa-pipeline@my-project.iam.gserviceaccount.com")
		require.Error(t, err)

		mockAPI.AssertExpectations(t)
	})

	t.Run("error when getting topic", func(t *testing.T) {
		mockAPI := &MockPubsubAPI{}
		mockAPI.On("GetTopic", mock.Anything, mock.MatchedBy(func(req *pubsubpb.GetTopicRequest) bool {
			return req.Topic == "projects/my-project/topics/my-topic"
		})).Return(nil, fmt.Errorf("iam policy error"))

		client := newPubsubClientWithAPI(mockAPI)
		err := client.EnsureTopic(context.Background(), "my-project", "my-topic", "sa-pipeline@my-project.iam.gserviceaccount.com")
		require.Error(t, err)

		mockAPI.AssertExpectations(t)
	})

	t.Run("success topic already exists", func(t *testing.T) {
		mockAPI := &MockPubsubAPI{}
		mockAPI.On("GetTopic", mock.Anything, mock.MatchedBy(func(req *pubsubpb.GetTopicRequest) bool {
			return req.Topic == "projects/my-project/topics/my-topic"
		})).Return(nil, status.Error(codes.AlreadyExists, "already exists"))

		mockAPI.On("GetIamPolicy", mock.Anything, mock.MatchedBy(func(req *iampb.GetIamPolicyRequest) bool {
			return req.Resource == "projects/my-project/topics/my-topic"
		})).Return(&iampb.Policy{}, nil)

		mockAPI.On("SetIamPolicy", mock.Anything, mock.MatchedBy(func(req *iampb.SetIamPolicyRequest) bool {
			return req.Resource == "projects/my-project/topics/my-topic" &&
				req.Policy.Bindings[0].Role == "roles/pubsub.publisher" &&
				req.Policy.Bindings[0].Members[0] == "serviceAccount:sa-pipeline@my-project.iam.gserviceaccount.com"
		})).Return(&iampb.Policy{}, nil)

		client := newPubsubClientWithAPI(mockAPI)
		err := client.EnsureTopic(context.Background(), "my-project", "my-topic", "sa-pipeline@my-project.iam.gserviceaccount.com")
		require.NoError(t, err)

		mockAPI.AssertExpectations(t)
	})
}

func TestDeleteTopic(t *testing.T) {
	t.Run("successfully delete topic", func(t *testing.T) {
		mockAPI := &MockPubsubAPI{}
		mockAPI.On("DeleteTopic", mock.Anything, mock.MatchedBy(func(req *pubsubpb.DeleteTopicRequest) bool {
			return req.Topic == "projects/my-project/topics/my-topic"
		})).Return(nil)

		client := newPubsubClientWithAPI(mockAPI)
		err := client.DeleteTopic(context.Background(), "my-project", "my-topic")
		require.NoError(t, err)

		mockAPI.AssertExpectations(t)
	})

	t.Run("successfully fail to delete topic when topic not found", func(t *testing.T) {
		mockAPI := &MockPubsubAPI{}
		mockAPI.On("DeleteTopic", mock.Anything, mock.MatchedBy(func(req *pubsubpb.DeleteTopicRequest) bool {
			return req.Topic == "projects/my-project/topics/my-topic"
		})).Return(status.Error(codes.NotFound, "topic not found"))

		client := newPubsubClientWithAPI(mockAPI)
		err := client.DeleteTopic(context.Background(), "my-project", "my-topic")
		require.NoError(t, err)

		mockAPI.AssertExpectations(t)
	})

	t.Run("error when strange error happens on delete", func(t *testing.T) {
		mockAPI := &MockPubsubAPI{}
		mockAPI.On("DeleteTopic", mock.Anything, mock.MatchedBy(func(req *pubsubpb.DeleteTopicRequest) bool {
			return req.Topic == "projects/my-project/topics/my-topic"
		})).Return(fmt.Errorf("some error"))

		client := newPubsubClientWithAPI(mockAPI)
		err := client.DeleteTopic(context.Background(), "my-project", "my-topic")
		require.Error(t, err)

		mockAPI.AssertExpectations(t)
	})
}

func TestPublishMessage(t *testing.T) {
	t.Run("successfully publish message to topic", func(t *testing.T) {
		mockAPI := &MockPubsubAPI{}
		mockAPI.On("Publish", mock.Anything, mock.MatchedBy(func(req *pubsubpb.PublishRequest) bool {
			return req.Topic == "projects/my-project/topics/my-topic" &&
				bytes.Equal(req.Messages[0].Data, []byte("my-data")) &&
				reflect.DeepEqual(req.Messages[0].Attributes, map[string]string{"my-key": "my-value"})
		})).Return(&pubsubpb.PublishResponse{MessageIds: []string{"my-id"}}, nil)

		client := newPubsubClientWithAPI(mockAPI)
		messageId, err := client.PublishMessage(context.Background(), "my-project", "my-topic", []byte("my-data"), map[string]string{"my-key": "my-value"})
		require.NoError(t, err)
		require.Equal(t, "my-id", messageId)
		mockAPI.AssertExpectations(t)
	})

	t.Run("error when publish message to topic", func(t *testing.T) {
		mockAPI := &MockPubsubAPI{}
		mockAPI.On("Publish", mock.Anything, mock.MatchedBy(func(req *pubsubpb.PublishRequest) bool {
			return req.Topic == "projects/my-project/topics/my-topic" &&
				bytes.Equal(req.Messages[0].Data, []byte("my-data")) &&
				reflect.DeepEqual(req.Messages[0].Attributes, map[string]string{"my-key": "my-value"})
		})).Return(nil, fmt.Errorf("some error"))

		client := newPubsubClientWithAPI(mockAPI)
		messageId, err := client.PublishMessage(context.Background(), "my-project", "my-topic", []byte("my-data"), map[string]string{"my-key": "my-value"})
		require.Error(t, err)
		require.Empty(t, messageId)
		mockAPI.AssertExpectations(t)
	})
}

func TestPubSubClientClose(t *testing.T) {
	t.Run("successfully close", func(t *testing.T) {
		mockAPI := &MockPubsubAPI{}
		mockAPI.On("Close").Return(nil)

		client := newPubsubClientWithAPI(mockAPI)
		err := client.Close()
		require.NoError(t, err)

		mockAPI.AssertExpectations(t)
	})
}

func TestNewPubSubClient(t *testing.T) {
	tests := []struct {
		name              string
		setupPubSubClient func(ctx context.Context, opts ...option.ClientOption) (PubsubClient, error)
		wantErr           bool
	}{
		{
			name: "failed client creation",
			setupPubSubClient: func(ctx context.Context, opts ...option.ClientOption) (PubsubClient, error) {
				client, err := NewPubsubClient(ctx, opts...)
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
			_, creationErr := tt.setupPubSubClient(context.Background(), option.WithCredentialsFile("/nonexistent/credentials.json"))
			if creationErr != nil && tt.wantErr {
				require.Error(t, creationErr)
			} else {
				require.NoError(t, creationErr)
			}
		})
	}
}
