package goservice

import (
	"context"
)

func (p *Pipeline) createGar(ctx context.Context) error {
	err := p.garClient.EnsureRepository(ctx, "hammer-bootstrap", "us-central1", p.component.Name)
	if err != nil {
		return err
	}
	return nil
}
