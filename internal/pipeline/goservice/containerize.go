package goservice

import (
	"context"
)

func (p *Pipeline) containerize(ctx context.Context) error {
	var baseImage string
	if p.ComponentType() == "gocli" {
		baseImage = "cgr.dev/chainguard/go:latest"
	} else {
		baseImage = "cgr.dev/chainguard/static:latest"
	}
	err := p.dockerClient.Build(ctx, baseImage, p.component.Name, "./localimage.tar")
	if err != nil {
		return err
	}
	return nil
}
