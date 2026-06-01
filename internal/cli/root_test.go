package cli

import (
	"context"
	"fmt"
	"testing"

	"github.com/ghost-pack/hammer/internal/docker"
	"github.com/ghost-pack/hammer/internal/oam"
	"github.com/ghost-pack/hammer/internal/pipeline"
	"github.com/stretchr/testify/require"
)

func init() {
	pipeline.Register("noop", New)
	pipeline.Register("bad", NewBadPipeline)
}

// The noop component is only used for testing.
func New(component oam.Component, client docker.DockerClient) (pipeline.Pipeline, error) {
	if component.Type != "noop" {
		return nil, fmt.Errorf("noop component must be of type noop")
	}

	return &Pipeline{
		component: &component,
	}, nil
}

type Pipeline struct {
	component *oam.Component
}

func (p *Pipeline) ComponentType() string {
	return "noop"
}

func (p *Pipeline) CI(ctx context.Context) error {
	return nil
}

func NewBadPipeline(component oam.Component, client docker.DockerClient) (pipeline.Pipeline, error) {
	if component.Type != "bad" {
		return nil, fmt.Errorf("bad component must be of type bad")
	}

	return &BadPipeline{
		component: &component,
	}, nil
}

type BadPipeline struct {
	component *oam.Component
}

func (p *BadPipeline) ComponentType() string {
	return "bad"
}

func (p *BadPipeline) CI(ctx context.Context) error {
	return fmt.Errorf("bad pipeline")
}

func Test_newRootCmd(t *testing.T) {
	tests := []struct {
		name string
	}{
		{
			name: "success root cmd creation",
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			rootCmd := newRootCmd()
			require.Equal(t, "hammer", rootCmd.Use)
			require.NotEmpty(t, rootCmd.Short)
		})
	}
}

func TestCI(t *testing.T) {
	tests := []struct {
		name    string
		oamFile string
		wantErr bool
	}{
		{
			name:    "successful CI execution",
			oamFile: "testdata/oam_test_success.yaml",
		},
		{
			name:    "failed unparseable oam file",
			oamFile: "testdata/oam_unparseable.yaml",
			wantErr: true,
		},
		{
			name:    "failed unparseable oam file",
			oamFile: "testdata/oam_test_fail_bad_component.yaml",
			wantErr: true,
		},
		{
			name:    "failed unparseable oam file",
			oamFile: "testdata/oam_test_fail_ci_failure.yaml",
			wantErr: true,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			rootCmd := newRootCmd()

			rootCmd.SetArgs([]string{"ci", "--file", tt.oamFile,
				"--env", "dev"})

			err := rootCmd.Execute()
			if tt.wantErr {
				require.Error(t, err)
			} else {
				require.NoError(t, err)
			}
		})
	}
}
