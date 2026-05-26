package goservice

import (
	"context"
	"errors"
	"testing"

	"github.com/ghost-pack/hammer/internal/oam"
	"github.com/ghost-pack/hammer/internal/pipeline"
	"github.com/ghost-pack/hammer/internal/runner"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
)

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

func TestNewPipeline(t *testing.T) {
	type args struct {
		component oam.Component
	}
	tests := []struct {
		name    string
		args    args
		want    pipeline.Pipeline
		wantErr bool
	}{
		{
			name:    "SuccessfulNewPipeline",
			args:    args{oam.Component{Name: "testComponent", Type: "goservice"}},
			want:    &Pipeline{component: &oam.Component{Name: "testComponent", Type: "goservice"}, runner: runner.New()},
			wantErr: false,
		},
		{
			name:    "FailedNewPipeline",
			args:    args{oam.Component{Name: "testComponent", Type: "notgoservice"}},
			want:    nil,
			wantErr: true,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := New(tt.args.component)
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
		setupMock func(*MockRunner)
		wantErr   bool
	}{
		{
			name:      "SuccessfulCIPipeline",
			component: &oam.Component{Name: "testComponent", Type: "goservice"},
			setupMock: func(m *MockRunner) {
				m.On("RunWithoutOptions", mock.Anything, "go", []string{"test", "./..."}).
					Return(&runner.Result{ExitCode: 0}, nil)
			},
			wantErr: false,
		},
		{
			name:      "FailedCIPipeline_CommandHadError",
			component: &oam.Component{Name: "testComponent", Type: "goservice"},
			setupMock: func(m *MockRunner) {
				m.On("RunWithoutOptions", mock.Anything, "go", []string{"test", "./..."}).
					Return(&runner.Result{ExitCode: 1}, errors.New("test error"))
			},
			wantErr: true,
		},
		{
			name:      "FailedCIPipeline_NoResultError",
			component: &oam.Component{Name: "testComponent", Type: "goservice"},
			setupMock: func(m *MockRunner) {
				m.On("RunWithoutOptions", mock.Anything, "go", []string{"test", "./..."}).
					Return(nil, errors.New("test error"))
			},
			wantErr: true,
		},
		{
			name:      "FailedCIPipeline_ResultNoError",
			component: &oam.Component{Name: "testComponent", Type: "goservice"},
			setupMock: func(m *MockRunner) {
				m.On("RunWithoutOptions", mock.Anything, "go", []string{"test", "./..."}).
					Return(&runner.Result{ExitCode: 1}, nil)
			},
			wantErr: true,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mockRunner := new(MockRunner)
			tt.setupMock(mockRunner)

			p := &Pipeline{
				component: tt.component,
				runner:    mockRunner,
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

func TestPipeline_ComponentType(t *testing.T) {
	p := &Pipeline{
		component: &oam.Component{Name: "testComponent", Type: "goservice"},
		runner:    &MockRunner{},
	}
	require.Equal(t, "goservice", p.ComponentType())
}
