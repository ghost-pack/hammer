package goservice

import (
	"context"
	"fmt"

	"github.com/ghost-pack/hammer/internal/runner"
)

func (p *Pipeline) test(ctx context.Context) error {
	result, err := runner.RunWithoutOptions(ctx, "go", []string{"test", "./..."})
	if err != nil {
		return err
	}
	if result.ExitCode != 0 {
		return fmt.Errorf("test failed with exit code %d: %s", result.ExitCode, result.Stderr)
	}
	return nil
}
