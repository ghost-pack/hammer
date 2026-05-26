package pipeline

import (
	"context"
	"testing"

	"github.com/ghost-pack/hammer/internal/oam"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
)

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
	tests := []struct {
		name      string
		component oam.Component
		setup     func()
		want      Pipeline
		wantErr   bool
	}{
		{
			name:      "SuccessfulFor",
			component: oam.Component{Name: "testComponent", Type: "goservice"},
			want:      &MockPipeline{},
			setup: func() {
				Register("goservice", func(component oam.Component) (Pipeline, error) {
					return &MockPipeline{}, nil
				})
			},
			wantErr: false,
		},
		{
			name:      "Failure_NilType",
			component: oam.Component{Name: "testComponent", Type: ""},
			want:      &MockPipeline{},
			setup:     func() {},
			wantErr:   true,
		},
		{
			name:      "Failure_UnregisteredType",
			component: oam.Component{Name: "testComponent", Type: "testtype"},
			want:      &MockPipeline{},
			setup:     func() {},
			wantErr:   true,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			registry = map[string]Factory{}
			tt.setup()
			got, err := For(tt.component)
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
				f: func(component oam.Component) (Pipeline, error) {
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
				f: func(component oam.Component) (Pipeline, error) {
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
				f: func(component oam.Component) (Pipeline, error) {
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
