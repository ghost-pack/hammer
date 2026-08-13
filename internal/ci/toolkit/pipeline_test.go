package toolkit

import (
	"context"
	"errors"
	"testing"

	"github.com/ghost-pack/hammer/internal/ci"
	"github.com/ghost-pack/hammer/internal/docker"
	"github.com/ghost-pack/hammer/internal/gcp"
	"github.com/ghost-pack/hammer/internal/gcp/project"
	"github.com/ghost-pack/hammer/internal/oam"
	"github.com/ghost-pack/hammer/internal/runner"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
	"gopkg.in/yaml.v3"
)

type MockDockerClient struct {
	mock.Mock
}

func (m *MockDockerClient) Build(ctx context.Context, baseImage, binaryPath, tarPath string) error {
	callArgs := m.Called(ctx, baseImage, binaryPath, tarPath)
	return callArgs.Error(0)
}

func (m *MockDockerClient) Push(ctx context.Context, tarPath, imageTag string) error {
	callArgs := m.Called(ctx, tarPath, imageTag)
	return callArgs.Error(0)
}

type MockRunner struct {
	mock.Mock
}

func (m *MockRunner) Run(ctx context.Context, name string, args []string, opts runner.Options) (*runner.Result, error) {
	callArgs := m.Called(ctx, name, args, opts)
	res, _ := callArgs.Get(0).(*runner.Result)
	return res, callArgs.Error(1)
}

func (m *MockRunner) RunWithoutOptions(ctx context.Context, name string, args []string) (*runner.Result, error) {
	callArgs := m.Called(ctx, name, args)
	res, _ := callArgs.Get(0).(*runner.Result)
	return res, callArgs.Error(1)
}

type MockGarClient struct {
	mock.Mock
}

func (m *MockGarClient) EnsureRepository(ctx context.Context, projectID, location, repoID string) error {
	callArgs := m.Called(ctx, projectID, location, repoID)
	return callArgs.Error(0)
}

func (m *MockGarClient) GrantRepositoryReader(ctx context.Context, projectID, location, repoID, saEmail string) error {
	callArgs := m.Called(ctx, projectID, location, repoID, saEmail)
	return callArgs.Error(0)
}

func (m *MockGarClient) Close() error {
	callArgs := m.Called()
	return callArgs.Error(0)
}

func TestNewPipeline(t *testing.T) {
	type args struct {
		component    oam.Component
		client       docker.Client
		garClient    project.GarClient
		dockerClient gcp.CloudBuildClient
	}
	tests := []struct {
		name    string
		args    args
		want    ci.Pipeline
		wantErr bool
	}{
		{
			name:    "SuccessfulNewPipeline",
			args:    args{component: oam.Component{Name: "testComponent", Type: "toolkit"}, client: &MockDockerClient{}, garClient: &MockGarClient{}},
			want:    &Pipeline{component: &oam.Component{Name: "testComponent", Type: "toolkit"}, runner: runner.New(), dockerClient: &MockDockerClient{}, garClient: &MockGarClient{}},
			wantErr: false,
		},
		{
			name:    "FailedNewPipeline",
			args:    args{component: oam.Component{Name: "testComponent", Type: "nottoolkit"}, client: &MockDockerClient{}, garClient: &MockGarClient{}},
			want:    nil,
			wantErr: true,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := New(tt.args.component, oam.App{}, ci.DependencyClients{DockerClient: tt.args.client, GarClient: tt.args.garClient, CloudBuild: tt.args.dockerClient})
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
		setupMock func(*MockRunner, *MockDockerClient, *MockGarClient)
		wantErr   bool
	}{
		{
			name:      "SuccessfulCIPipeline",
			component: &oam.Component{Name: "testComponent", Type: "goservice"},
			setupMock: func(mockRunner *MockRunner, mockDockerClient *MockDockerClient, mockGarClient *MockGarClient) {
				mockRunner.On("RunWithoutOptions", mock.Anything, "apko", mock.Anything).
					Return(&runner.Result{ExitCode: 0}, nil)
				mockGarClient.On("EnsureRepository", mock.Anything, mock.Anything, mock.Anything, mock.Anything).
					Return(nil)
				mockDockerClient.On("Push", mock.Anything, mock.Anything, mock.Anything).
					Return(nil)
				t.Setenv("COMMIT_SHA", "abc1234")
			},
			wantErr: false,
		},
		{
			name:      "FailedCIPipeline_apko_fails",
			component: &oam.Component{Name: "testComponent", Type: "goservice"},
			setupMock: func(mockRunner *MockRunner, mockDockerClient *MockDockerClient, mockGarClient *MockGarClient) {
				mockRunner.On("RunWithoutOptions", mock.Anything, "apko", mock.Anything).
					Return(nil, errors.New("apko error"))
			},
			wantErr: true,
		},
		{
			name:      "FailedCIPipeline_apko_fails_and_returns_result",
			component: &oam.Component{Name: "testComponent", Type: "goservice"},
			setupMock: func(mockRunner *MockRunner, mockDockerClient *MockDockerClient, mockGarClient *MockGarClient) {
				mockRunner.On("RunWithoutOptions", mock.Anything, "apko", mock.Anything).
					Return(&runner.Result{ExitCode: 0}, errors.New("apko error"))
			},
			wantErr: true,
		},
		{
			name: "FailedCIPipeline_apko_fails_and_with_bad_properties",
			component: &oam.Component{Name: "testComponent", Type: "goservice",
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
			setupMock: func(mockRunner *MockRunner, mockDockerClient *MockDockerClient, mockGarClient *MockGarClient) {
			},
			wantErr: true,
		},
		{
			name:      "FailedCIPipeline_createGarFails",
			component: &oam.Component{Name: "testComponent", Type: "goservice"},
			setupMock: func(mockRunner *MockRunner, mockDockerClient *MockDockerClient, mockGarClient *MockGarClient) {
				mockRunner.On("RunWithoutOptions", mock.Anything, "apko", mock.Anything).
					Return(&runner.Result{ExitCode: 0}, nil)
				mockGarClient.On("EnsureRepository", mock.Anything, mock.Anything, mock.Anything, mock.Anything).
					Return(errors.New("gar error"))
			},
			wantErr: true,
		},
		{
			name:      "FailedCIPipeline_PushFails",
			component: &oam.Component{Name: "testComponent", Type: "goservice"},
			setupMock: func(mockRunner *MockRunner, mockDockerClient *MockDockerClient, mockGarClient *MockGarClient) {
				mockRunner.On("RunWithoutOptions", mock.Anything, "apko", mock.Anything).
					Return(&runner.Result{ExitCode: 0}, nil)
				mockGarClient.On("EnsureRepository", mock.Anything, mock.Anything, mock.Anything, mock.Anything).
					Return(nil)
				mockDockerClient.On("Push", mock.Anything, mock.Anything, mock.Anything).
					Return(errors.New("push error"))
				t.Setenv("COMMIT_SHA", "")
			},
			wantErr: true,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mockRunner := new(MockRunner)
			mockDockerClient := new(MockDockerClient)
			mockGarClient := new(MockGarClient)
			tt.setupMock(mockRunner, mockDockerClient, mockGarClient)

			p := &Pipeline{
				component:    tt.component,
				runner:       mockRunner,
				dockerClient: mockDockerClient,
				garClient:    mockGarClient,
			}

			// Don't need to test for artifact here since toolkit never makes one.
			_, err := p.CI(context.Background())

			if tt.wantErr {
				require.Error(t, err)
			} else {
				require.NoError(t, err)
			}

			// Ensure all expected mock calls were made
			mockRunner.AssertExpectations(t)
			mockDockerClient.AssertExpectations(t)
			mockGarClient.AssertExpectations(t)
		})
	}
}

func TestPipeline_Analyze(t *testing.T) {
	tests := []struct {
		name      string
		component *oam.Component
		setupMock func(*MockRunner, *MockDockerClient, *MockGarClient)
		wantErr   bool
	}{
		{
			name:      "SuccessfulAnalyzePipeline",
			component: &oam.Component{Name: "testComponent", Type: "goservice"},
			setupMock: func(mockRunner *MockRunner, mockDockerClient *MockDockerClient, mockGarClient *MockGarClient) {
				mockRunner.On("RunWithoutOptions", mock.Anything, "apko", mock.Anything).
					Return(&runner.Result{ExitCode: 0}, nil)
			},
			wantErr: false,
		},
		{
			name:      "FailedAnalyzePipeline",
			component: &oam.Component{Name: "testComponent", Type: "goservice"},
			setupMock: func(mockRunner *MockRunner, mockDockerClient *MockDockerClient, mockGarClient *MockGarClient) {
				mockRunner.On("RunWithoutOptions", mock.Anything, "apko", mock.Anything).
					Return(&runner.Result{ExitCode: 1}, nil)
			},
			wantErr: true,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mockRunner := new(MockRunner)
			mockDockerClient := new(MockDockerClient)
			mockGarClient := new(MockGarClient)
			tt.setupMock(mockRunner, mockDockerClient, mockGarClient)

			p := &Pipeline{
				component:    tt.component,
				runner:       mockRunner,
				dockerClient: mockDockerClient,
				garClient:    mockGarClient,
			}

			err := p.Analyze(context.Background())

			if tt.wantErr {
				require.Error(t, err)
			} else {
				require.NoError(t, err)
			}

			// Ensure all expected mock calls were made
			mockRunner.AssertExpectations(t)
			mockDockerClient.AssertExpectations(t)
			mockGarClient.AssertExpectations(t)
		})
	}
}

func TestPipeline_Deploy(t *testing.T) {
	tests := []struct {
		name      string
		component *oam.Component
		setupMock func(*MockRunner, *MockDockerClient, *MockGarClient)
		wantErr   bool
	}{
		{
			name:      "SuccessfulDeployPipeline",
			component: &oam.Component{Name: "testComponent", Type: "goservice"},
			setupMock: func(mockRunner *MockRunner, mockDockerClient *MockDockerClient, mockGarClient *MockGarClient) {
			},
			wantErr: false,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mockRunner := new(MockRunner)
			mockDockerClient := new(MockDockerClient)
			mockGarClient := new(MockGarClient)
			tt.setupMock(mockRunner, mockDockerClient, mockGarClient)

			p := &Pipeline{
				component:    tt.component,
				runner:       mockRunner,
				dockerClient: mockDockerClient,
				garClient:    mockGarClient,
			}

			err := p.Deploy(context.Background())

			if tt.wantErr {
				require.Error(t, err)
			} else {
				require.NoError(t, err)
			}

			// Ensure all expected mock calls were made
			mockRunner.AssertExpectations(t)
			mockDockerClient.AssertExpectations(t)
			mockGarClient.AssertExpectations(t)
		})
	}
}
