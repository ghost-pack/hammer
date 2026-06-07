package pipeline

import (
	"context"
	"testing"

	"github.com/ghost-pack/hammer/internal/docker"
	"github.com/ghost-pack/hammer/internal/gcp"
	"github.com/ghost-pack/hammer/internal/oam"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
)

type MockDockerClient struct {
	mock.Mock
}

func (m *MockDockerClient) Build(ctx context.Context, baseImage, binaryPath, imageTag string) error {
	callArgs := m.Called(ctx, baseImage, binaryPath, imageTag)
	return callArgs.Error(0)
}

func (m *MockDockerClient) Tag(ctx context.Context, source, target string) error {
	callArgs := m.Called(ctx, source, target)
	return callArgs.Error(0)
}

func (m *MockDockerClient) Push(ctx context.Context, image string) error {
	callArgs := m.Called(ctx, image)
	return callArgs.Error(0)
}

func (m *MockDockerClient) Close() error {
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

func (m *MockPipeline) CI(ctx context.Context) error {
	args := m.Called(ctx)
	return args.Error(0)
}

func TestFor(t *testing.T) {
	type args struct {
		component oam.Component
		client    docker.DockerClient
		garClient gcp.GarClient
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
				Register("goservice", func(component oam.Component, client docker.DockerClient, garClient gcp.GarClient) (Pipeline, error) {
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
			got, err := For(tt.args.component, tt.args.client, tt.args.garClient)
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
				f: func(component oam.Component, client docker.DockerClient, garClient gcp.GarClient) (Pipeline, error) {
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
				f: func(component oam.Component, client docker.DockerClient, garClient gcp.GarClient) (Pipeline, error) {
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
				f: func(component oam.Component, client docker.DockerClient, garClient gcp.GarClient) (Pipeline, error) {
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
