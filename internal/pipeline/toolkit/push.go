package toolkit

import (
	"context"
	"fmt"
	"os"
)

func (p *Pipeline) push(ctx context.Context) error {
	var tag string
	if sha := os.Getenv("COMMIT_SHA"); sha != "" {
		tag = sha[:7]
	} else {
		tag = "dev"
	}

	fullImageTag := fmt.Sprintf("us-central1-docker.pkg.dev/hammer-bootstrap/%s/%s:%s", p.component.Name, p.component.Name, tag)

	err := p.dockerClient.Push(ctx, "./apko-wolfi.tar", fullImageTag)
	if err != nil {
		return err
	}
	return nil
}
