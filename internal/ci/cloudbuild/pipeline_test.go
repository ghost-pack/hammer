package cloudbuild

import (
	"context"
	"errors"
	"os"
	"testing"

	"github.com/ghost-pack/hammer/internal/ci"
	"github.com/ghost-pack/hammer/internal/gcp"
	"github.com/ghost-pack/hammer/internal/oam"
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

func (m *MockCloudBuildClient) CreateOrUpdateCloudBuildTrigger(ctx context.Context, projectID, projectNumber, location, cloudBuildPath, triggerName, triggerType, pubsubTopic, serviceAccount string, manuallyApproved bool) error {
	callArgs := m.Called(ctx, projectID, projectNumber, location, cloudBuildPath, triggerName, triggerType, pubsubTopic, serviceAccount, manuallyApproved)
	return callArgs.Error(0)
}

func (m *MockCloudBuildClient) Close() error {
	callArgs := m.Called()
	return callArgs.Error(0)
}

type MockCloudStorage struct {
	mock.Mock
}

func (m *MockCloudStorage) EnsureBucketExists(ctx context.Context, projectId, location, bucketName string) error {
	callArgs := m.Called(ctx, projectId, location, bucketName)
	return callArgs.Error(0)
}

func (m *MockCloudStorage) GetObject(ctx context.Context, bucket, object string) ([]byte, error) {
	callArgs := m.Called(ctx, bucket, object)
	res, _ := callArgs.Get(0).([]byte)
	return res, callArgs.Error(1)
}

func (m *MockCloudStorage) WriteObject(ctx context.Context, bucket, object string, data []byte, metadata map[string]string) error {
	callArgs := m.Called(ctx, bucket, object, data, metadata)
	return callArgs.Error(0)
}

func (m *MockCloudStorage) ListPrefixes(ctx context.Context, bucket, prefix, delimiter string) ([]string, error) {
	callArgs := m.Called(ctx, bucket, prefix, delimiter)
	res, _ := callArgs.Get(0).([]string)
	return res, callArgs.Error(1)
}

func (m *MockCloudStorage) Close() error {
	callArgs := m.Called()
	return callArgs.Error(0)
}

type MockPubSubClient struct {
	mock.Mock
}

func (m *MockPubSubClient) EnsureTopic(ctx context.Context, projectID, topicID, publisherSA string) error {
	callArgs := m.Called(ctx, projectID, topicID, publisherSA)
	return callArgs.Error(0)
}

func (m *MockPubSubClient) DeleteTopic(ctx context.Context, projectID, topicID string) error {
	callArgs := m.Called(ctx, projectID, topicID)
	return callArgs.Error(0)
}

func (m *MockPubSubClient) PublishMessage(ctx context.Context, projectID, topicID string, data []byte, attributes map[string]string) (string, error) {
	callArgs := m.Called(ctx, projectID, topicID, data, attributes)
	res, _ := callArgs.Get(0).(string)
	return res, callArgs.Error(1)
}

func (m *MockPubSubClient) Close() error {
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
		want    ci.Pipeline
		wantErr bool
	}{
		{
			name: "SuccessfulNewPipeline",
			args: args{component: oam.Component{Name: "testComponent", Type: "cloudbuild"}, cloudBuildClient: &MockCloudBuildClient{}},
			want: &Pipeline{
				component:             &oam.Component{Name: "testComponent", Type: "cloudbuild"},
				app:                   &oam.App{Metadata: oam.Metadata{Name: "acme-corp"}},
				releaseBucket:         "hammer-release",
				shortCommitSha:        "1234567",
				cloudBuildClient:      &MockCloudBuildClient{},
				platformProject:       "hammer-central-prod",
				platformProjectNumber: "598451979611",
			},
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
			os.Setenv("COMMIT_SHA", "12345678910")
			got, err := New(tt.args.component, oam.App{Metadata: oam.Metadata{Name: "acme-corp"}}, ci.DependencyClients{CloudBuild: tt.args.cloudBuildClient})
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
		name             string
		component        *oam.Component
		app              *oam.App
		setupMock        func(*MockCloudBuildClient, *MockCloudStorage)
		expectedArtifact *ci.Artifact
		wantErr          bool
	}{
		{
			name: "SuccessfulCIPipeline",
			app:  &oam.App{Metadata: oam.Metadata{Name: "acme-corp"}},
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
			setupMock: func(mockCloudBuildClient *MockCloudBuildClient, mockCloudStorage *MockCloudStorage) {
				mockCloudBuildClient.On("TestCloudBuild", mock.Anything, mock.Anything, mock.Anything, mock.Anything, mock.Anything).
					Return(nil)
				mockCloudStorage.On("WriteObject", mock.Anything, mock.Anything, mock.Anything, mock.Anything, mock.Anything).
					Return(nil)
			},
			expectedArtifact: &ci.Artifact{
				Type: ci.ArtifactTypeCloudBuild,
				Properties: map[string]string{
					"cloudBuildYaml": "gs://hammer-release/acme-corp/deployments/cloudbuild/1234567.yaml",
				},
			},
			wantErr: false,
		},
		{
			name: "FailedCIPipeline_bad_component",
			app:  &oam.App{Metadata: oam.Metadata{Name: "acme-corp"}},
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
			setupMock: func(mockCloudBuildClient *MockCloudBuildClient, mockCloudStorage *MockCloudStorage) {
			},
			wantErr: true,
		},
		{
			name: "FailedCIPipeline_failed_cloudbuild_read",
			app:  &oam.App{Metadata: oam.Metadata{Name: "acme-corp"}},
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
			setupMock: func(mockCloudBuildClient *MockCloudBuildClient, mockCloudStorage *MockCloudStorage) {
			},
			wantErr: true,
		},
		{
			name: "FailedCIPipeline_bad_cloudbuild_yaml",
			app:  &oam.App{Metadata: oam.Metadata{Name: "acme-corp"}},
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
			setupMock: func(mockCloudBuildClient *MockCloudBuildClient, mockCloudStorage *MockCloudStorage) {
			},
			wantErr: true,
		},
		{
			name: "FailedCIPipeline_bad_schema",
			app:  &oam.App{Metadata: oam.Metadata{Name: "acme-corp"}},
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
			setupMock: func(mockCloudBuildClient *MockCloudBuildClient, mockCloudStorage *MockCloudStorage) {
			},
			wantErr: true,
		},
		{
			name: "FailedCIPipeline_cloud_build_failure",
			app:  &oam.App{Metadata: oam.Metadata{Name: "acme-corp"}},
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
			setupMock: func(mockCloudBuildClient *MockCloudBuildClient, mockCloudStorage *MockCloudStorage) {
				mockCloudBuildClient.On("TestCloudBuild", mock.Anything, mock.Anything, mock.Anything, mock.Anything, mock.Anything).
					Return(errors.New("test error"))
			},
			wantErr: true,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mockCloudBuildClient := new(MockCloudBuildClient)
			mockCloudStorageClient := new(MockCloudStorage)
			tt.setupMock(mockCloudBuildClient, mockCloudStorageClient)

			p := &Pipeline{
				app:                tt.app,
				component:          tt.component,
				cloudBuildClient:   mockCloudBuildClient,
				cloudStorageClient: mockCloudStorageClient,
				releaseBucket:      "hammer-release",
				shortCommitSha:     "1234567",
			}

			// TODO: test for artifact
			artifact, err := p.CI(context.Background())

			if tt.wantErr {
				require.Error(t, err)
			} else {
				require.NoError(t, err)
				require.Equal(t, tt.expectedArtifact, artifact)
			}

			mockCloudBuildClient.AssertExpectations(t)
			mockCloudStorageClient.AssertExpectations(t)
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
