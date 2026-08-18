package goservice

import (
	"context"
	"fmt"

	"github.com/ghost-pack/hammer/internal/runner"
)

func (p *Pipeline) test(ctx context.Context) error {
	props, err := parseGoPath(p)
	if err != nil {
		return err
	}

	result, err := p.runner.Run(ctx, "go", []string{"test", "./..."}, runner.Options{Dir: props.Path})
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
