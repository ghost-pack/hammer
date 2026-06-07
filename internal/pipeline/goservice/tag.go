package goservice

import (
	"context"
	"fmt"
)

func (p *Pipeline) tag(ctx context.Context) error {
	err := p.dockerClient.Tag(ctx, "myapp:local", fmt.Sprintf("us-central1-docker.pkg.dev/cloud-build-pipeline-396819/%s/%s:latest", p.component.Name, p.component.Name))
	if err != nil {
		return err
	}
	return nil
}
