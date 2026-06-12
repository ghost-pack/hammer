package goservice

import (
	"context"
	"errors"
	"testing"

	"github.com/ghost-pack/hammer/internal/docker"
	"github.com/ghost-pack/hammer/internal/gcp"
	"github.com/ghost-pack/hammer/internal/oam"
	"github.com/ghost-pack/hammer/internal/pipeline"
	"github.com/ghost-pack/hammer/internal/runner"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
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

func (m *MockGarClient) Close() error {
	callArgs := m.Called()
	return callArgs.Error(0)
}

func TestNewPipeline(t *testing.T) {
	type args struct {
		component oam.Component
		client    docker.Client
		garClient gcp.GarClient
	}
	tests := []struct {
		name    string
		args    args
		want    pipeline.Pipeline
		wantErr bool
	}{
		{
			name:    "SuccessfulNewPipeline",
			args:    args{component: oam.Component{Name: "testComponent", Type: "goservice"}, client: &MockDockerClient{}, garClient: &MockGarClient{}},
			want:    &Pipeline{component: &oam.Component{Name: "testComponent", Type: "goservice"}, runner: runner.New(), dockerClient: &MockDockerClient{}, garClient: &MockGarClient{}},
			wantErr: false,
		},
		{
			name:    "FailedNewPipeline",
			args:    args{component: oam.Component{Name: "testComponent", Type: "notgoservice"}, client: &MockDockerClient{}, garClient: &MockGarClient{}},
			want:    nil,
			wantErr: true,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := New(tt.args.component, tt.args.client, tt.args.garClient)
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
				mockRunner.On("RunWithoutOptions", mock.Anything, "go", []string{"test", "./..."}).
					Return(&runner.Result{ExitCode: 0}, nil)
				mockRunner.On("Run", mock.Anything, "go", []string{"build", "-o", "testComponent", "."}, mock.Anything).
					Return(&runner.Result{ExitCode: 0}, nil)
				mockDockerClient.On("Build", mock.Anything, mock.Anything, mock.Anything, mock.Anything).
					Return(nil)
				mockGarClient.On("EnsureRepository", mock.Anything, mock.Anything, mock.Anything, mock.Anything).
					Return(nil)
				mockDockerClient.On("Push", mock.Anything, mock.Anything, mock.Anything).
					Return(nil)
			},
			wantErr: false,
		},
		{
			name:      "SuccessfulCIPipeline",
			component: &oam.Component{Name: "testComponent", Type: "gocli"},
			setupMock: func(mockRunner *MockRunner, mockDockerClient *MockDockerClient, mockGarClient *MockGarClient) {
				mockRunner.On("RunWithoutOptions", mock.Anything, "go", []string{"test", "./..."}).
					Return(&runner.Result{ExitCode: 0}, nil)
				mockRunner.On("Run", mock.Anything, "go", []string{"build", "-o", "testComponent", "."}, mock.Anything).
					Return(&runner.Result{ExitCode: 0}, nil)
				mockDockerClient.On("Build", mock.Anything, mock.Anything, mock.Anything, mock.Anything).
					Return(nil)
				mockGarClient.On("EnsureRepository", mock.Anything, mock.Anything, mock.Anything, mock.Anything).
					Return(nil)
				mockDockerClient.On("Push", mock.Anything, mock.Anything, mock.Anything).
					Return(nil)
			},
			wantErr: false,
		},
		{
			name:      "SuccessfulCIPipelineWithCommitSha",
			component: &oam.Component{Name: "testComponent", Type: "goservice"},
			setupMock: func(mockRunner *MockRunner, mockDockerClient *MockDockerClient, mockGarClient *MockGarClient) {
				mockRunner.On("RunWithoutOptions", mock.Anything, "go", []string{"test", "./..."}).
					Return(&runner.Result{ExitCode: 0}, nil)
				mockRunner.On("Run", mock.Anything, "go", []string{"build", "-o", "testComponent", "."}, mock.Anything).
					Return(&runner.Result{ExitCode: 0}, nil)
				mockDockerClient.On("Build", mock.Anything, mock.Anything, mock.Anything, mock.Anything).
					Return(nil)
				mockGarClient.On("EnsureRepository", mock.Anything, mock.Anything, mock.Anything, mock.Anything).
					Return(nil)
				mockDockerClient.On("Push", mock.Anything, mock.Anything, mock.Anything).
					Return(nil)
				t.Setenv("COMMIT_SHA", "abc1234")
			},
			wantErr: false,
		},
		{
			name:      "FailedCIPipeline Containerization",
			component: &oam.Component{Name: "testComponent", Type: "goservice"},
			setupMock: func(mockRunner *MockRunner, mockDockerClient *MockDockerClient, mockGarClient *MockGarClient) {
				mockRunner.On("RunWithoutOptions", mock.Anything, "go", []string{"test", "./..."}).
					Return(&runner.Result{ExitCode: 0}, nil)
				mockRunner.On("Run", mock.Anything, "go", []string{"build", "-o", "testComponent", "."}, mock.Anything).
					Return(&runner.Result{ExitCode: 0}, nil)
				mockDockerClient.On("Build", mock.Anything, mock.Anything, mock.Anything, mock.Anything).
					Return(errors.New("test error"))
			},
			wantErr: true,
		},
		{
			name:      "FailedCIPipeline_CommandHadError",
			component: &oam.Component{Name: "testComponent", Type: "goservice"},
			setupMock: func(mockRunner *MockRunner, mockDockerClient *MockDockerClient, mockGarClient *MockGarClient) {
				mockRunner.On("RunWithoutOptions", mock.Anything, "go", []string{"test", "./..."}).
					Return(&runner.Result{ExitCode: 1}, errors.New("test error"))
			},
			wantErr: true,
		},
		{
			name:      "FailedCIPipeline_NoResultError",
			component: &oam.Component{Name: "testComponent", Type: "goservice"},
			setupMock: func(mockRunner *MockRunner, mockDockerClient *MockDockerClient, mockGarClient *MockGarClient) {
				mockRunner.On("RunWithoutOptions", mock.Anything, "go", []string{"test", "./..."}).
					Return(nil, errors.New("test error"))
			},
			wantErr: true,
		},
		{
			name:      "FailedCIPipeline_ResultNoError",
			component: &oam.Component{Name: "testComponent", Type: "goservice"},
			setupMock: func(mockRunner *MockRunner, mockDockerClient *MockDockerClient, mockGarClient *MockGarClient) {
				mockRunner.On("RunWithoutOptions", mock.Anything, "go", []string{"test", "./..."}).
					Return(&runner.Result{ExitCode: 1}, nil)
			},
			wantErr: true,
		},
		{
			name:      "FailedCI_BuildHadError",
			component: &oam.Component{Name: "testComponent", Type: "goservice"},
			setupMock: func(mockRunner *MockRunner, mockDockerClient *MockDockerClient, mockGarClient *MockGarClient) {
				mockRunner.On("RunWithoutOptions", mock.Anything, "go", []string{"test", "./..."}).
					Return(&runner.Result{ExitCode: 0}, nil)
				mockRunner.On("Run", mock.Anything, "go", []string{"build", "-o", "testComponent", "."}, mock.Anything).
					Return(&runner.Result{ExitCode: 1}, errors.New("test error"))
			},
			wantErr: true,
		},
		{
			name:      "FailedCI_BuildHadErrorNoResult",
			component: &oam.Component{Name: "testComponent", Type: "goservice"},
			setupMock: func(mockRunner *MockRunner, mockDockerClient *MockDockerClient, mockGarClient *MockGarClient) {
				mockRunner.On("RunWithoutOptions", mock.Anything, "go", []string{"test", "./..."}).
					Return(&runner.Result{ExitCode: 0}, nil)
				mockRunner.On("Run", mock.Anything, "go", []string{"build", "-o", "testComponent", "."}, mock.Anything).
					Return(nil, errors.New("test error"))
			},
			wantErr: true,
		},
		{
			name:      "FailedCI_BuildHadBadResult",
			component: &oam.Component{Name: "testComponent", Type: "goservice"},
			setupMock: func(mockRunner *MockRunner, mockDockerClient *MockDockerClient, mockGarClient *MockGarClient) {
				mockRunner.On("RunWithoutOptions", mock.Anything, "go", []string{"test", "./..."}).
					Return(&runner.Result{ExitCode: 0}, nil)
				mockRunner.On("Run", mock.Anything, "go", []string{"build", "-o", "testComponent", "."}, mock.Anything).
					Return(&runner.Result{ExitCode: 1}, nil)
			},
			wantErr: true,
		},
		{
			name:      "Failed gar ensure",
			component: &oam.Component{Name: "testComponent", Type: "goservice"},
			setupMock: func(mockRunner *MockRunner, mockDockerClient *MockDockerClient, mockGarClient *MockGarClient) {
				mockRunner.On("RunWithoutOptions", mock.Anything, "go", []string{"test", "./..."}).
					Return(&runner.Result{ExitCode: 0}, nil)
				mockRunner.On("Run", mock.Anything, "go", []string{"build", "-o", "testComponent", "."}, mock.Anything).
					Return(&runner.Result{ExitCode: 0}, nil)
				mockDockerClient.On("Build", mock.Anything, mock.Anything, mock.Anything, mock.Anything).
					Return(nil)
				mockGarClient.On("EnsureRepository", mock.Anything, mock.Anything, mock.Anything, mock.Anything).
					Return(errors.New("ensure error"))
			},
			wantErr: true,
		},
		{
			name:      "FailedCIPushHadError",
			component: &oam.Component{Name: "testComponent", Type: "goservice"},
			setupMock: func(mockRunner *MockRunner, mockDockerClient *MockDockerClient, mockGarClient *MockGarClient) {
				mockRunner.On("RunWithoutOptions", mock.Anything, "go", []string{"test", "./..."}).
					Return(&runner.Result{ExitCode: 0}, nil)
				mockRunner.On("Run", mock.Anything, "go", []string{"build", "-o", "testComponent", "."}, mock.Anything).
					Return(&runner.Result{ExitCode: 0}, nil)
				mockDockerClient.On("Build", mock.Anything, mock.Anything, mock.Anything, mock.Anything).
					Return(nil)
				mockGarClient.On("EnsureRepository", mock.Anything, mock.Anything, mock.Anything, mock.Anything).
					Return(nil)
				mockDockerClient.On("Push", mock.Anything, mock.Anything, mock.Anything).
					Return(errors.New("push error"))
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

			err := p.CI(context.Background())

			if tt.wantErr {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
			}

			// Ensure all expected mock calls were made
			mockRunner.AssertExpectations(t)
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
				mockRunner.On("RunWithoutOptions", mock.Anything, "go", []string{"test", "./..."}).
					Return(&runner.Result{ExitCode: 0}, nil)
			},
			wantErr: false,
		},
		{
			name:      "FailedAnalyzePipeline",
			component: &oam.Component{Name: "testComponent", Type: "goservice"},
			setupMock: func(mockRunner *MockRunner, mockDockerClient *MockDockerClient, mockGarClient *MockGarClient) {
				mockRunner.On("RunWithoutOptions", mock.Anything, "go", []string{"test", "./..."}).
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
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
			}

			// Ensure all expected mock calls were made
			mockRunner.AssertExpectations(t)
		})
	}
}

func TestPipeline_ComponentType(t *testing.T) {
	p := &Pipeline{
		component: &oam.Component{Name: "testComponent", Type: "goservice"},
		runner:    &MockRunner{},
	}
	require.Equal(t, "goservice", p.ComponentType())
}
