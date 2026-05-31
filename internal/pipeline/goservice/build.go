package goservice

import (
	"context"
	"fmt"
)

func (p *Pipeline) build(ctx context.Context) error {
	result, err := p.runner.RunWithoutOptions(ctx, "go", []string{"build", "."})
	if err != nil {
		if result == nil {
			return err
		}
		return fmt.Errorf("test failed with error %w: %s", err, result.Stdout)
	}
	if result.ExitCode != 0 {
		return fmt.Errorf("test failed with exit code %d: %s", result.ExitCode, result.Stderr)
	}
	return nil
}
