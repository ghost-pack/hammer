package toolkit

import (
	"context"
)

func (p *Pipeline) createGar(ctx context.Context) error {
	err := p.garClient.EnsureRepository(ctx, "cloud-build-pipeline-396819", "us-central1", p.component.Name)
	if err != nil {
		return err
	}
	return nil
}
