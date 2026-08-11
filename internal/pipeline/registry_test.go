package pipeline

import (
	"context"
	"testing"

	"github.com/ghost-pack/hammer/internal/docker"
	"github.com/ghost-pack/hammer/internal/gcp"
	"github.com/ghost-pack/hammer/internal/gcp/project"
	"github.com/ghost-pack/hammer/internal/oam"
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

type MockCloudBuildClient struct {
	mock.Mock
}

func (m *MockCloudBuildClient) TestCloudBuild(ctx context.Context, projectID, location, cloudbuildPath, cloudBuildTestPath string) error {
	callArgs := m.Called(ctx, projectID, location, cloudbuildPath, cloudBuildTestPath)
	return callArgs.Error(0)
}

func (m *MockCloudBuildClient) Close() error {
	callArgs := m.Called()
	return callArgs.Error(0)
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

type MockPipeline struct {
	mock.Mock
}

func (m *MockPipeline) ComponentType() string {
	args := m.Called()
	return args.String(0)
}

func (m *MockPipeline) CI(ctx context.Context) (*Artifact, error) {
	args := m.Called(ctx)
	op, _ := args.Get(0).(*Artifact)
	return op, args.Error(1)
}

func (m *MockPipeline) Deploy(ctx context.Context) error {
	args := m.Called(ctx)
	return args.Error(0)
}

func (m *MockPipeline) Analyze(ctx context.Context) error {
	args := m.Called(ctx)
	return args.Error(0)
}

func TestFor(t *testing.T) {
	type args struct {
		component        oam.Component
		client           docker.Client
		garClient        project.GarClient
		cloudBuildClient gcp.CloudBuildClient
		cloudStorage     gcp.CloudStorageClient
	}
	tests := []struct {
		name    string
		args    args
		setup   func()
		want    Pipeline
		wantErr bool
	}{
		{
			name: "SuccessfulFor",
			args: args{
				component: oam.Component{Name: "testComponent", Type: "goservice"},
				client:    &MockDockerClient{},
				garClient: &MockGarClient{},
			},
			want: &MockPipeline{},
			setup: func() {
				Register("goservice", func(component oam.Component, client DependencyClients) (Pipeline, error) {
					return &MockPipeline{}, nil
				})
			},
			wantErr: false,
		},
		{
			name: "Failure_NilType",
			args: args{
				component: oam.Component{Name: "testComponent", Type: ""},
				client:    &MockDockerClient{},
				garClient: &MockGarClient{},
			},
			want:    &MockPipeline{},
			setup:   func() {},
			wantErr: true,
		},
		{
			name: "Failure_UnregisteredType",
			args: args{
				component: oam.Component{Name: "testComponent", Type: "testtype"},
				client:    &MockDockerClient{},
				garClient: &MockGarClient{},
			},
			want:    &MockPipeline{},
			setup:   func() {},
			wantErr: true,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			registry = map[string]Factory{}
			tt.setup()
			got, err := For(tt.args.component, DependencyClients{DockerClient: tt.args.client, GarClient: tt.args.garClient, CloudBuild: tt.args.cloudBuildClient, CloudStorage: tt.args.cloudStorage})
			if err != nil {
				if tt.wantErr {
					require.Error(t, err)
				} else {
					require.Nil(t, err)
				}
			} else {
				require.Equal(t, tt.want, got)
			}
		})
	}
}

func TestRegister(t *testing.T) {
	type args struct {
		componentType string
		f             Factory
	}
	tests := []struct {
		name         string
		args         args
		preRegister  bool
		panicMessage string
	}{
		{
			name: "SuccessRegister",
			args: args{
				componentType: "goservice",
				f: func(component oam.Component, client DependencyClients) (Pipeline, error) {
					return &MockPipeline{}, nil
				},
			},
			preRegister:  false,
			panicMessage: "",
		},
		{
			name: "FailedRegister_duplicate",
			args: args{
				componentType: "goservice",
				f: func(component oam.Component, client DependencyClients) (Pipeline, error) {
					return &MockPipeline{}, nil
				},
			},
			preRegister:  true,
			panicMessage: "pipeline.Registry: componentType already registered",
		},
		{
			name: "FailedRegister_noFactory",
			args: args{
				componentType: "goservice",
				f:             nil,
			},
			preRegister:  false,
			panicMessage: "pipeline.Registry: factory is nil",
		},
		{
			name: "FailedRegister_noComponentType",
			args: args{
				componentType: "",
				f: func(component oam.Component, client DependencyClients) (Pipeline, error) {
					return &MockPipeline{}, nil
				},
			},
			preRegister:  false,
			panicMessage: "pipeline.Registry: componentType is empty",
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			registry = map[string]Factory{}
			if tt.preRegister {
				Register(tt.args.componentType, tt.args.f)
			}

			call := func() {
				Register(tt.args.componentType, tt.args.f)
			}

			if tt.panicMessage != "" {
				require.PanicsWithValue(t, tt.panicMessage, call)
			} else {
				require.NotPanics(t, call)
			}
		})
	}
}
