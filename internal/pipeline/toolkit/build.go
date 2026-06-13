package toolkit

import (
	"context"
	"fmt"
	"os"
)

type Properties struct {
	Path string `yaml:"path"`
}

func (p *Pipeline) build(ctx context.Context) error {
	var properties Properties
	if err := p.component.Properties.Decode(&properties); err != nil {
		return fmt.Errorf("decoding properties: %w", err)
	}
	if properties.Path == "" {
		properties.Path = "./wolfi-base.yaml"
	}

	var tag string
	if sha := os.Getenv("COMMIT_SHA"); sha != "" {
		tag = sha[:7]
	} else {
		tag = "dev"
	}

	result, err := p.runner.RunWithoutOptions(ctx, "apko", []string{"build", properties.Path, p.component.Name + ":" + tag, "apko-wolfi.tar"})
	if err != nil {
		if result == nil {
			return err
		}
		return fmt.Errorf("build failed with error %w: %s", err, result.Stdout)
	}
	if result.ExitCode != 0 {
		return fmt.Errorf("build failed with exit code %d: %s", result.ExitCode, result.Stderr)
	}
	return nil
}
