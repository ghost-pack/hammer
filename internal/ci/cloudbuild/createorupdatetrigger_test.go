package cloudbuild

import (
	"context"
	"errors"
	"fmt"
	"testing"

	"github.com/ghost-pack/hammer/internal/oam"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"gopkg.in/yaml.v3"
)

func TestPipeline_createorupdatetrigger(t *testing.T) {
	tests := []struct {
		name      string
		component *oam.Component
		setupMock func(*MockCloudBuildClient, *MockPubSubClient)
		wantErr   bool
	}{
		{
			name: "SuccessfulCreateOrUpdateTriggerTestPhase",
			component: &oam.Component{Name: "testComponent", Type: "cloudbuild", Properties: yaml.Node{
				Kind: yaml.MappingNode,
				Content: []*yaml.Node{
					{Kind: yaml.ScalarNode, Value: "path"},
					{Kind: yaml.ScalarNode, Value: "./testdata/cloudbuild.yaml"},
					{Kind: yaml.ScalarNode, Value: "trigger_type"},
					{Kind: yaml.ScalarNode, Value: "pubsub"},
					{Kind: yaml.ScalarNode, Value: "pubsub_topic"},
					{Kind: yaml.ScalarNode, Value: "whatever"},
					{Kind: yaml.ScalarNode, Value: "service_account"},
					{Kind: yaml.ScalarNode, Value: "sa-oam@hammer-central-prod.iam.gserviceaccount.com"},
				},
			}},
			setupMock: func(mockCloudBuildClient *MockCloudBuildClient, mockPubSubClient *MockPubSubClient) {
				mockCloudBuildClient.On("CreateOrUpdateCloudBuildTrigger", mock.Anything, mock.Anything, mock.Anything, mock.Anything, mock.Anything, mock.Anything, mock.Anything, mock.Anything, mock.Anything, mock.Anything).
					Return(nil)
				mockPubSubClient.On("EnsureTopic", mock.Anything, mock.Anything, mock.Anything, mock.Anything).Return(nil).
					Return(nil)
			},
			wantErr: false,
		},
		{
			name: "fail to create pubsub topic",
			component: &oam.Component{Name: "testComponent", Type: "cloudbuild", Properties: yaml.Node{
				Kind: yaml.MappingNode,
				Content: []*yaml.Node{
					{Kind: yaml.ScalarNode, Value: "path"},
					{Kind: yaml.ScalarNode, Value: "./testdata/cloudbuild.yaml"},
					{Kind: yaml.ScalarNode, Value: "trigger_type"},
					{Kind: yaml.ScalarNode, Value: "pubsub"},
					{Kind: yaml.ScalarNode, Value: "pubsub_topic"},
					{Kind: yaml.ScalarNode, Value: "whatever"},
					{Kind: yaml.ScalarNode, Value: "service_account"},
					{Kind: yaml.ScalarNode, Value: "sa-oam@hammer-central-prod.iam.gserviceaccount.com"},
				},
			}},
			setupMock: func(mockCloudBuildClient *MockCloudBuildClient, mockPubSubClient *MockPubSubClient) {
				mockPubSubClient.On("EnsureTopic", mock.Anything, mock.Anything, mock.Anything, mock.Anything).Return(nil).
					Return(fmt.Errorf("error"))
			},
			wantErr: true,
		},
		{
			name: "FailedCreateOrUpdateTriggerTestPipeline_bad_component",
			component: &oam.Component{Name: "testComponent", Type: "cloudbuild",
				Properties: yaml.Node{
					Kind: yaml.MappingNode,
					Tag:  "!!map",
					Content: []*yaml.Node{
						{Kind: yaml.ScalarNode, Tag: "!!str", Value: "path"},
						{
							Kind: yaml.SequenceNode,
							Tag:  "!!seq",
							Content: []*yaml.Node{
								{Kind: yaml.ScalarNode, Tag: "!!str", Value: "item"},
							},
						},
					},
				},
			},
			setupMock: func(mockCloudBuildClient *MockCloudBuildClient, mockPubSubClient *MockPubSubClient) {
			},
			wantErr: true,
		},
		{
			name: "FailedCreateOrUpdateTriggerTestPipeline_failed_cloudbuild_test_read",
			component: &oam.Component{Name: "testComponent", Type: "cloudbuild",
				Properties: yaml.Node{
					Kind:    yaml.MappingNode,
					Content: []*yaml.Node{},
				},
			},
			setupMock: func(mockCloudBuildClient *MockCloudBuildClient, mockPubSubClient *MockPubSubClient) {
				mockCloudBuildClient.On("CreateOrUpdateCloudBuildTrigger", mock.Anything, mock.Anything, mock.Anything, mock.Anything, mock.Anything, mock.Anything, mock.Anything, mock.Anything, mock.Anything, mock.Anything).
					Return(errors.New("failed"))
			},
			wantErr: true,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mockCloudBuildClient := new(MockCloudBuildClient)
			mockPubSubClient := new(MockPubSubClient)
			tt.setupMock(mockCloudBuildClient, mockPubSubClient)

			p := &Pipeline{
				component:        tt.component,
				cloudBuildClient: mockCloudBuildClient,
				pubsubClient:     mockPubSubClient,
			}

			err := p.createOrUpdateTrigger(context.Background())

			if tt.wantErr {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
			}

			mockCloudBuildClient.AssertExpectations(t)
			mockPubSubClient.AssertExpectations(t)
		})
	}
}
