package cloudbuild

import (
	"context"
	"errors"
	"testing"

	"github.com/ghost-pack/hammer/internal/gcp"
	"github.com/ghost-pack/hammer/internal/oam"
	"github.com/ghost-pack/hammer/internal/pipeline"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
	"gopkg.in/yaml.v3"
)

type MockCloudBuildClient struct {
	mock.Mock
}

func (m *MockCloudBuildClient) TestCloudBuild(ctx context.Context, projectID, location, cloudbuildPath, cloudBuildTestPath string) error {
	callArgs := m.Called(ctx, projectID, location, cloudbuildPath, cloudBuildTestPath)
	return callArgs.Error(0)
}

func (m *MockCloudBuildClient) CreateOrUpdateCloudBuildTrigger(ctx context.Context, projectID, projectNumber, location, cloudBuildPath, triggerName string) error {
	callArgs := m.Called(ctx, projectID, projectNumber, location, cloudBuildPath, triggerName)
	return callArgs.Error(0)
}

func (m *MockCloudBuildClient) Close() error {
	callArgs := m.Called()
	return callArgs.Error(0)
}

func TestNewPipeline(t *testing.T) {
	type args struct {
		component        oam.Component
		cloudBuildClient gcp.CloudBuildClient
	}
	tests := []struct {
		name    string
		args    args
		want    pipeline.Pipeline
		wantErr bool
	}{
		{
			name:    "SuccessfulNewPipeline",
			args:    args{component: oam.Component{Name: "testComponent", Type: "cloudbuild"}, cloudBuildClient: &MockCloudBuildClient{}},
			want:    &Pipeline{component: &oam.Component{Name: "testComponent", Type: "cloudbuild"}, cloudBuildClient: &MockCloudBuildClient{}},
			wantErr: false,
		},
		{
			name:    "FailedNewPipeline",
			args:    args{component: oam.Component{Name: "testComponent", Type: "notcloudbuild"}, cloudBuildClient: &MockCloudBuildClient{}},
			want:    nil,
			wantErr: true,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := New(tt.args.component, pipeline.DependencyClients{CloudBuild: tt.args.cloudBuildClient})
			if err != nil {
				if tt.wantErr {
					require.Error(t, err)
				} else {
					t.Errorf("New() error = %v, wantErr %v", err, tt.wantErr)
				}
			}
			require.Equal(t, tt.want, got)
		})
	}
}

func TestPipeline_CI(t *testing.T) {
	tests := []struct {
		name      string
		component *oam.Component
		setupMock func(*MockCloudBuildClient)
		wantErr   bool
	}{
		{
			name: "SuccessfulCIPipeline",
			component: &oam.Component{Name: "testComponent", Type: "cloudbuild", Properties: yaml.Node{
				Kind: yaml.MappingNode,
				Content: []*yaml.Node{
					{Kind: yaml.ScalarNode, Value: "path"},
					{Kind: yaml.ScalarNode, Value: "./testdata/cloudbuild.yaml"},
					{Kind: yaml.ScalarNode, Value: "tests"},
					{
						Kind: yaml.SequenceNode,
						Content: []*yaml.Node{
							{
								Kind: yaml.MappingNode,
								Content: []*yaml.Node{
									{Kind: yaml.ScalarNode, Value: "path"},
									{Kind: yaml.ScalarNode, Value: "./testdata/cloudbuild_test.yaml"},
									{Kind: yaml.ScalarNode, Value: "required"},
									{Kind: yaml.ScalarNode, Value: "true"},
								},
							},
						},
					},
				},
			}},
			setupMock: func(mockCloudBuildClient *MockCloudBuildClient) {
				mockCloudBuildClient.On("TestCloudBuild", mock.Anything, mock.Anything, mock.Anything, mock.Anything, mock.Anything).
					Return(nil)
			},
			wantErr: false,
		},
		{
			name: "FailedCIPipeline_bad_component",
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
			name: "FailedCIPipeline_failed_cloudbuild_read",
			component: &oam.Component{Name: "testComponent", Type: "cloudbuild",
				Properties: yaml.Node{
					Kind: yaml.MappingNode,
					Content: []*yaml.Node{
						{Kind: yaml.ScalarNode, Value: "testPath"},
						{Kind: yaml.ScalarNode, Value: "tests"},
						{
							Kind: yaml.SequenceNode,
							Content: []*yaml.Node{
								{
									Kind: yaml.MappingNode,
									Content: []*yaml.Node{
										{Kind: yaml.ScalarNode, Value: "path"},
										{Kind: yaml.ScalarNode, Value: "./testdata/cloudbuild_test.yaml"},
										{Kind: yaml.ScalarNode, Value: "required"},
										{Kind: yaml.ScalarNode, Value: "true"},
									},
								},
							},
						},
					},
				}},
			setupMock: func(mockCloudBuildClient *MockCloudBuildClient) {
			},
			wantErr: true,
		},
		{
			name: "FailedCIPipeline_bad_cloudbuild_yaml",
			component: &oam.Component{Name: "testComponent", Type: "cloudbuild", Properties: yaml.Node{
				Kind: yaml.MappingNode,
				Content: []*yaml.Node{
					{Kind: yaml.ScalarNode, Value: "path"},
					{Kind: yaml.ScalarNode, Value: "./testdata/cloudbuild_bad_yaml.yaml"},
					{Kind: yaml.ScalarNode, Value: "tests"},
					{
						Kind: yaml.SequenceNode,
						Content: []*yaml.Node{
							{
								Kind: yaml.MappingNode,
								Content: []*yaml.Node{
									{Kind: yaml.ScalarNode, Value: "path"},
									{Kind: yaml.ScalarNode, Value: "./testdata/cloudbuild_test.yaml"},
									{Kind: yaml.ScalarNode, Value: "required"},
									{Kind: yaml.ScalarNode, Value: "true"},
								},
							},
						},
					},
				},
			}},
			setupMock: func(mockCloudBuildClient *MockCloudBuildClient) {
			},
			wantErr: true,
		},
		{
			name: "FailedCIPipeline_bad_schema",
			component: &oam.Component{Name: "testComponent", Type: "cloudbuild", Properties: yaml.Node{
				Kind: yaml.MappingNode,
				Content: []*yaml.Node{
					{Kind: yaml.ScalarNode, Value: "path"},
					{Kind: yaml.ScalarNode, Value: "./testdata/cloudbuild_not_matching_schema.yaml"},
					{Kind: yaml.ScalarNode, Value: "tests"},
					{
						Kind: yaml.SequenceNode,
						Content: []*yaml.Node{
							{
								Kind: yaml.MappingNode,
								Content: []*yaml.Node{
									{Kind: yaml.ScalarNode, Value: "path"},
									{Kind: yaml.ScalarNode, Value: "./testdata/cloudbuild_test.yaml"},
									{Kind: yaml.ScalarNode, Value: "required"},
									{Kind: yaml.ScalarNode, Value: "true"},
								},
							},
						},
					},
				},
			}},
			setupMock: func(mockCloudBuildClient *MockCloudBuildClient) {
			},
			wantErr: true,
		},
		{
			name: "FailedCIPipeline_cloud_build_failure",
			component: &oam.Component{Name: "testComponent", Type: "cloudbuild", Properties: yaml.Node{
				Kind: yaml.MappingNode,
				Content: []*yaml.Node{
					{Kind: yaml.ScalarNode, Value: "path"},
					{Kind: yaml.ScalarNode, Value: "./testdata/cloudbuild.yaml"},
					{Kind: yaml.ScalarNode, Value: "tests"},
					{
						Kind: yaml.SequenceNode,
						Content: []*yaml.Node{
							{
								Kind: yaml.MappingNode,
								Content: []*yaml.Node{
									{Kind: yaml.ScalarNode, Value: "path"},
									{Kind: yaml.ScalarNode, Value: "./testdata/cloudbuild_test.yaml"},
									{Kind: yaml.ScalarNode, Value: "required"},
									{Kind: yaml.ScalarNode, Value: "true"},
								},
							},
						},
					},
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

			err := p.CI(context.Background())

			if tt.wantErr {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
			}

			mockCloudBuildClient.AssertExpectations(t)
		})
	}
}

func TestPipeline_Analyze(t *testing.T) {
	tests := []struct {
		name      string
		component *oam.Component
		setupMock func(*MockCloudBuildClient)
		wantErr   bool
	}{
		{
			name: "SuccessfulAnalyzePipeline",
			component: &oam.Component{Name: "testComponent", Type: "cloudbuild", Properties: yaml.Node{
				Kind: yaml.MappingNode,
				Content: []*yaml.Node{
					{Kind: yaml.ScalarNode, Value: "path"},
					{Kind: yaml.ScalarNode, Value: "./testdata/cloudbuild.yaml"},
					{Kind: yaml.ScalarNode, Value: "tests"},
					{
						Kind: yaml.SequenceNode,
						Content: []*yaml.Node{
							{
								Kind: yaml.MappingNode,
								Content: []*yaml.Node{
									{Kind: yaml.ScalarNode, Value: "path"},
									{Kind: yaml.ScalarNode, Value: "./testdata/cloudbuild_test.yaml"},
									{Kind: yaml.ScalarNode, Value: "required"},
									{Kind: yaml.ScalarNode, Value: "true"},
								},
							},
						},
					},
				},
			}},
			setupMock: func(mockCloudBuildClient *MockCloudBuildClient) {
				mockCloudBuildClient.On("TestCloudBuild", mock.Anything, mock.Anything, mock.Anything, mock.Anything, mock.Anything).
					Return(nil)
			},
			wantErr: false,
		},
		{
			name: "FailedAnalyzePipeline_bad_component",
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
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mockCloudBuildClient := new(MockCloudBuildClient)
			tt.setupMock(mockCloudBuildClient)

			p := &Pipeline{
				component:        tt.component,
				cloudBuildClient: mockCloudBuildClient,
			}

			err := p.Analyze(context.Background())

			if tt.wantErr {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
			}

			// Ensure all expected mock calls were made
			mockCloudBuildClient.AssertExpectations(t)
		})
	}
}

func TestPipeline_Deploy(t *testing.T) {
	tests := []struct {
		name      string
		component *oam.Component
		setupMock func(*MockCloudBuildClient)
		noOutput  bool
		wantErr   bool
	}{
		{
			name: "SuccessfulDeployPipeline",
			component: &oam.Component{Name: "testComponent", Type: "cloudbuild", Properties: yaml.Node{
				Kind: yaml.MappingNode,
				Content: []*yaml.Node{
					{Kind: yaml.ScalarNode, Value: "path"},
					{Kind: yaml.ScalarNode, Value: "./testdata/cloudbuild.yaml"},
					{Kind: yaml.ScalarNode, Value: "tests"},
					{
						Kind: yaml.SequenceNode,
						Content: []*yaml.Node{
							{
								Kind: yaml.MappingNode,
								Content: []*yaml.Node{
									{Kind: yaml.ScalarNode, Value: "path"},
									{Kind: yaml.ScalarNode, Value: "./testdata/cloudbuild_test.yaml"},
									{Kind: yaml.ScalarNode, Value: "required"},
									{Kind: yaml.ScalarNode, Value: "true"},
								},
							},
						},
					},
				},
			}},
			setupMock: func(mockCloudBuildClient *MockCloudBuildClient) {
				mockCloudBuildClient.On("CreateOrUpdateCloudBuildTrigger", mock.Anything, mock.Anything, mock.Anything, mock.Anything, mock.Anything, mock.Anything).
					Return(nil)
			},
			noOutput: false,
			wantErr:  false,
		},
		{
			name: "FailedDeployPipeline",
			component: &oam.Component{Name: "testComponent", Type: "cloudbuild", Properties: yaml.Node{
				Kind: yaml.MappingNode,
				Content: []*yaml.Node{
					{Kind: yaml.ScalarNode, Value: "path"},
					{Kind: yaml.ScalarNode, Value: "./testdata/cloudbuild.yaml"},
					{Kind: yaml.ScalarNode, Value: "tests"},
					{
						Kind: yaml.SequenceNode,
						Content: []*yaml.Node{
							{
								Kind: yaml.MappingNode,
								Content: []*yaml.Node{
									{Kind: yaml.ScalarNode, Value: "path"},
									{Kind: yaml.ScalarNode, Value: "./testdata/cloudbuild_test.yaml"},
									{Kind: yaml.ScalarNode, Value: "required"},
									{Kind: yaml.ScalarNode, Value: "true"},
								},
							},
						},
					},
				},
			}},
			setupMock: func(mockCloudBuildClient *MockCloudBuildClient) {
				mockCloudBuildClient.On("CreateOrUpdateCloudBuildTrigger", mock.Anything, mock.Anything, mock.Anything, mock.Anything, mock.Anything, mock.Anything).
					Return(errors.New("error"))
			},
			noOutput: false,
			wantErr:  true,
		},
		{
			name: "FailedDeployPipeline",
			component: &oam.Component{Name: "testComponent", Type: "cloudbuild", Properties: yaml.Node{
				Kind: yaml.MappingNode,
				Content: []*yaml.Node{
					{Kind: yaml.ScalarNode, Value: "path"},
					{Kind: yaml.ScalarNode, Value: "./testdata/cloudbuild.yaml"},
					{Kind: yaml.ScalarNode, Value: "tests"},
					{
						Kind: yaml.SequenceNode,
						Content: []*yaml.Node{
							{
								Kind: yaml.MappingNode,
								Content: []*yaml.Node{
									{Kind: yaml.ScalarNode, Value: "path"},
									{Kind: yaml.ScalarNode, Value: "./testdata/cloudbuild_test.yaml"},
									{Kind: yaml.ScalarNode, Value: "required"},
									{Kind: yaml.ScalarNode, Value: "true"},
								},
							},
						},
					},
				},
			}},
			setupMock: func(mockCloudBuildClient *MockCloudBuildClient) {
			},
			noOutput: true,
			wantErr:  true,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mockCloudBuildClient := new(MockCloudBuildClient)
			tt.setupMock(mockCloudBuildClient)

			var p *Pipeline
			if tt.noOutput {
				p = &Pipeline{
					component:        tt.component,
					cloudBuildClient: mockCloudBuildClient,
				}
			} else {
				p = &Pipeline{
					component:        tt.component,
					cloudBuildClient: mockCloudBuildClient,
					cioutput:         "whatever",
				}
			}

			err := p.Deploy(context.Background())

			if tt.wantErr {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
			}

			mockCloudBuildClient.AssertExpectations(t)
		})
	}
}
