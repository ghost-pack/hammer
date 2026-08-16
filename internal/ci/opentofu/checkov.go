package opentofu

import (
	"context"
	"fmt"
)

func (p *Pipeline) checkov(ctx context.Context, env string) error {
	// probably gotta add like path here...checkov -d . --compact
	result, err := p.runner.RunWithoutOptions(ctx, "checkov", []string{"-d", ".", "--compact"})
	if err != nil {
		if result == nil {
			return err
		}
		return fmt.Errorf("checkov failed with error %w: %s", err, result.Stdout)
	}
	if result.ExitCode != 0 {
		return fmt.Errorf("checkov failed with exit code %d: %s", result.ExitCode, result.Stderr)
	}
	return nil
}
