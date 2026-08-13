package goservice

import (
	"context"
	"fmt"
)

func (p *Pipeline) push(ctx context.Context) error {
	var tag string
	if p.shortCommitSha == "" {
		tag = "dev"
	} else {
		tag = p.shortCommitSha
	}

	fullImageTag := fmt.Sprintf("us-central1-docker.pkg.dev/hammer-central-prod/%s/%s:%s", p.component.Name, p.component.Name, tag)

	err := p.dockerClient.Push(ctx, "./localimage.tar", fullImageTag)
	if err != nil {
		return err
	}
	return nil
}
