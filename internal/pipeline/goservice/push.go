package goservice

import (
	"context"
	"fmt"
)

func (p *Pipeline) push(ctx context.Context) error {
	err := p.dockerClient.Push(ctx, fmt.Sprintf("us-central1-docker.pkg.dev/cloud-build-pipeline-396819/%s/%s:latest", p.component.Name, p.component.Name))
	if err != nil {
		return err
	}
	return nil
}
