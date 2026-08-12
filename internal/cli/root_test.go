package cli

import (
	"context"
	"fmt"
	"os"
	"testing"

	"github.com/ghost-pack/hammer/internal/oam"
	"github.com/ghost-pack/hammer/internal/pipeline"
	"github.com/stretchr/testify/require"
)

func init() {
	pipeline.Register("bad", NewBadPipeline)
	pipeline.Register("GoodCiBadDeployPipeline", NewGoodCiBadDeployPipeline)
}

func NewBadPipeline(component oam.Component, client pipeline.DependencyClients) (pipeline.Pipeline, error) {
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

func (p *BadPipeline) CI(ctx context.Context) (*pipeline.Artifact, error) {
	return nil, fmt.Errorf("bad pipeline")
}

func (p *BadPipeline) Deploy(ctx context.Context) error {
	return fmt.Errorf("bad pipeline")
}

func (p *BadPipeline) Analyze(ctx context.Context) error {
	return fmt.Errorf("bad pipeline")
}

func NewGoodCiBadDeployPipeline(component oam.Component, client pipeline.DependencyClients) (pipeline.Pipeline, error) {
	if component.Type != "GoodCiBadDeployPipeline" {
		return nil, fmt.Errorf("GoodCiBadDeployPipeline component must be of type GoodCiBadDeployPipeline")
	}

	return &GoodCiBadDeployPipeline{
		component: &component,
	}, nil
}

type GoodCiBadDeployPipeline struct {
	component *oam.Component
}

func (p *GoodCiBadDeployPipeline) ComponentType() string {
	return "bad"
}

func (p *GoodCiBadDeployPipeline) CI(ctx context.Context) (*pipeline.Artifact, error) {
	return nil, nil
}

func (p *GoodCiBadDeployPipeline) Deploy(ctx context.Context) error {
	return fmt.Errorf("bad pipeline")
}

func (p *GoodCiBadDeployPipeline) Analyze(ctx context.Context) error {
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
			name:    "failed CI execution due to unmockable object",
			oamFile: "testdata/oam_test_success.yaml",
			wantErr: true,
		},
		{
			name:    "failed unparseable oam file",
			oamFile: "testdata/oam_unparseable.yaml",
			wantErr: true,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			os.Setenv("BRANCH_NAME", "main")
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

func TestAnalyze(t *testing.T) {
	tests := []struct {
		name    string
		oamFile string
		wantErr bool
	}{
		{
			name:    "failed CI execution due to unmockable clients",
			oamFile: "testdata/oam_test_success.yaml",
			wantErr: true,
		},
		{
			name:    "failed unparseable oam file",
			oamFile: "testdata/oam_unparseable.yaml",
			wantErr: true,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			os.Setenv("BRANCH_NAME", "notmain")
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
