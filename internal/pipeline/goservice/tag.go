package goservice

import (
	"context"
	"fmt"
	"os"
)

func (p *Pipeline) tag(ctx context.Context) error {
	var tag string
	if sha := os.Getenv("COMMIT_SHA"); sha != "" {
		tag = sha[:7]
	} else {
		tag = "dev"
	}

	fullImageTag := fmt.Sprintf("us-central1-docker.pkg.dev/cloud-build-pipeline-396819/%s/%s:%s", p.component.Name, p.component.Name, tag)
	err := p.dockerClient.Tag(ctx, "myapp:local", fullImageTag)
	if err != nil {
		return err
	}
	return nil
}
