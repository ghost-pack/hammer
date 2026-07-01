package opentofu

import (
	"context"
)

func (p *Pipeline) ensureBucketExists(ctx context.Context) error {
	err := p.cloudStorageClient.EnsureBucketExists(ctx, "cloud-build-pipeline-396819", "us-central1", p.component.Name)
	if err != nil {
		return err
	}
	return nil
}
