package gcp

import (
	"context"
)

func (p *Provisioner) ensureBucketExists(ctx context.Context) error {
	err := p.clients.CloudStorage.EnsureBucketExists(ctx, p.platformProject, p.defaultRegion, p.registryBucket)
	if err != nil {
		return err
	}
	return nil
}
