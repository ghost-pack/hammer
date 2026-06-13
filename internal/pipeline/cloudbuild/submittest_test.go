package cloudbuild

import (
	"context"
	"errors"
	"testing"

	"github.com/ghost-pack/hammer/internal/oam"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"gopkg.in/yaml.v3"
)

func TestPipeline_submittest(t *testing.T) {
	tests := []struct {
		name      string
		component *oam.Component
		setupMock func(*MockCloudBuildClient)
		wantErr   bool
	}{
		{
			name: "SuccessfulSubmitTestPhase",
			component: &oam.Component{Name: "testComponent", Type: "cloudbuild", Properties: yaml.Node{
				Kind: yaml.MappingNode,
				Content: []*yaml.Node{
					{Kind: yaml.ScalarNode, Value: "path"},
					{Kind: yaml.ScalarNode, Value: "./testdata/cloudbuild.yaml"},
					{Kind: yaml.ScalarNode, Value: "testPath"},
					{Kind: yaml.ScalarNode, Value: "./testdata/cloudbuild_test.yaml"},
				},
			}},
			setupMock: func(mockCloudBuildClient *MockCloudBuildClient) {
				mockCloudBuildClient.On("TestCloudBuild", mock.Anything, mock.Anything, mock.Anything, mock.Anything, mock.Anything).
					Return(nil)
			},
			wantErr: false,
		},
		{
			name: "FailedSubmitTestPipeline_bad_component",
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
			setupMock: func(mockCloudBuildClient *MockCloudBuildClient) {
			},
			wantErr: true,
		},
		{
			name: "SuccessSubmitTestPipeline_no_cloudbuild",
			component: &oam.Component{Name: "testComponent", Type: "cloudbuild",
				Properties: yaml.Node{
					Kind: yaml.MappingNode,
					Content: []*yaml.Node{
						{Kind: yaml.ScalarNode, Value: "testPath"},
						{Kind: yaml.ScalarNode, Value: "./testdata/cloudbuild_test.yaml"},
					},
				},
			},
			setupMock: func(mockCloudBuildClient *MockCloudBuildClient) {
				mockCloudBuildClient.On("TestCloudBuild", mock.Anything, mock.Anything, mock.Anything, mock.Anything, mock.Anything).
					Return(nil)
			},
			wantErr: false,
		},
		{
			name: "FailedSubmitTestPipeline_failed_cloudbuild_test_read",
			component: &oam.Component{Name: "testComponent", Type: "cloudbuild",
				Properties: yaml.Node{
					Kind: yaml.MappingNode,
					Content: []*yaml.Node{
						{Kind: yaml.ScalarNode, Value: "testPath"},
						{Kind: yaml.ScalarNode, Value: ""},
					},
				},
			},
			setupMock: func(mockCloudBuildClient *MockCloudBuildClient) {
			},
			wantErr: true,
		},
		{
			name: "FailedSubmitTestPipeline_cloud_build_failure",
			component: &oam.Component{Name: "testComponent", Type: "cloudbuild", Properties: yaml.Node{
				Kind: yaml.MappingNode,
				Content: []*yaml.Node{
					{Kind: yaml.ScalarNode, Value: "path"},
					{Kind: yaml.ScalarNode, Value: "./testdata/cloudbuild.yaml"},
					{Kind: yaml.ScalarNode, Value: "testPath"},
					{Kind: yaml.ScalarNode, Value: "./testdata/cloudbuild_test.yaml"},
				},
			}},
			setupMock: func(mockCloudBuildClient *MockCloudBuildClient) {
				mockCloudBuildClient.On("TestCloudBuild", mock.Anything, mock.Anything, mock.Anything, mock.Anything, mock.Anything).
					Return(errors.New("test error"))
			},
			wantErr: true,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mockCloudBuildClient := new(MockCloudBuildClient)
			tt.setupMock(mockCloudBuildClient)

			p := &Pipeline{
				component:        tt.component,
				cloudBuildClient: mockCloudBuildClient,
			}

			err := p.submitTest(context.Background())

			if tt.wantErr {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
			}

			mockCloudBuildClient.AssertExpectations(t)
		})
	}
}
