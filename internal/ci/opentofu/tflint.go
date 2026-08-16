package opentofu

import (
	"context"
	"fmt"
)

func (p *Pipeline) tflint(ctx context.Context, env string) error {
	result, err := p.runner.RunWithoutOptions(ctx, "tflint", []string{"--recursive"})
	if err != nil {
		if result == nil {
			return err
		}
		return fmt.Errorf("tflint failed with error %w: %s", err, result.Stdout)
	}
	if result.ExitCode != 0 {
		return fmt.Errorf("tflint failed with exit code %d: %s", result.ExitCode, result.Stderr)
	}
	return nil
}
