package goservice

import (
	"context"
)

func (p *Pipeline) containerize(ctx context.Context) error {
	err := p.dockerClient.Build(ctx, "cgr.dev/chainguard/static:latest", p.component.Name, "myapp:local")
	if err != nil {
		return err
	}
	return nil
}
