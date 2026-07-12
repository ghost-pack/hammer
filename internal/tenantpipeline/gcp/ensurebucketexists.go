package gcp

import (
	"context"
)

func (p *Provisioner) ensureBucketExists(ctx context.Context) error {
	err := p.clients.CloudStorage.EnsureBucketExists(ctx, "hammer-bootstrap", "us-central1", p.registryBucket)
	if err != nil {
		return err
	}
	return nil
}
