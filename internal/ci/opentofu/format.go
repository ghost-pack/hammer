package opentofu

import (
	"context"
	"fmt"

	"github.com/ghost-pack/hammer/internal/runner"
)

func (p *Pipeline) format(ctx context.Context, env string) error {
	props, err := parseOpenTofuPath(p)
	if err != nil {
		return err
	}

	result, err := p.runner.Run(ctx, "tofu", []string{"fmt", "-check", "-recursive"}, runner.Options{Dir: props.Path})
	if err != nil {
		if result == nil {
			return err
		}
		return fmt.Errorf("format failed with error %w: %s", err, result.Stdout)
	}
	if result.ExitCode != 0 {
		return fmt.Errorf("format failed with exit code %d: %s", result.ExitCode, result.Stderr)
	}
	return nil
}
