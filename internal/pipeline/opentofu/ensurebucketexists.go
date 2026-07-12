package opentofu

import (
	"context"
)

func (p *Pipeline) ensureBucketExists(ctx context.Context) error {
	err := p.cloudStorageClient.EnsureBucketExists(ctx, "hammer-bootstrap", "us-central1", p.component.Name)
	if err != nil {
		return err
	}
	return nil
}
