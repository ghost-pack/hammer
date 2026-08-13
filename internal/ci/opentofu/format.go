package opentofu

import (
	"context"
	"fmt"
)

func (p *Pipeline) format(ctx context.Context, env string) error {
	result, err := p.runner.RunWithoutOptions(ctx, "tofu", []string{"fmt", "-check", "-recursive"})
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
