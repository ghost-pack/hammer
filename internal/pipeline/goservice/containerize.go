package goservice

import (
	"context"
)

func (p *Pipeline) containerize(ctx context.Context) error {
	err := p.dockerClient.Build(ctx, p.component.Name, "testImageTagWhatever")
	if err != nil {
		return err
	}
	return nil
}
