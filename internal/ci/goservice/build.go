package goservice

import (
	"context"
	"fmt"
	"os"

	"github.com/ghost-pack/hammer/internal/runner"
)

func (p *Pipeline) build(ctx context.Context) error {
	result, err := p.runner.Run(ctx, "go", []string{"build", "-o", p.component.Name, "."}, runner.Options{
		Env: append(os.Environ(),
			"CGO_ENABLED=0",
			"GOOS=linux",
			"GOARCH=amd64",
		),
	})
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
