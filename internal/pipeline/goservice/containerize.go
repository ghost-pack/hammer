package goservice

import (
	"context"
)

func (p *Pipeline) containerize(ctx context.Context) error {
	var baseImage string
	if p.ComponentType() == "gocli" {
		baseImage = "us-central1-docker.pkg.dev/cloud-build-pipeline-396819/hammer-toolkit/hammer-toolkit@sha256:09869a84ba42cb6da5af60f344a38f5ed9ebf2a459a5a24f2b49c47442439d64"
	} else {
		baseImage = "cgr.dev/chainguard/static:latest"
	}
	err := p.dockerClient.Build(ctx, baseImage, p.component.Name, "./localimage.tar")
	if err != nil {
		return err
	}
	return nil
}
