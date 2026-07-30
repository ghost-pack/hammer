package gcp

import (
	"context"
)

func (p *Provisioner) ensureFolderExists(ctx context.Context) error {
	folderName, err := p.clients.ResourceManager.EnsureFolderExists(ctx, p.tenant.Metadata.Name, p.tenant.Spec.ParentFolder)
	if err != nil {
		return err
	}
	p.newState.Parent = folderName
	return nil
}
