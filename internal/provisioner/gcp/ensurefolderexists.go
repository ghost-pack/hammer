package gcp

import (
	"context"
)

func (p *Provisioner) ensureFolderExists(ctx context.Context) error {
	if p.tenant.Metadata.Name == platformTenant {
		p.newState.Parent = p.tenant.Spec.ParentFolder
		return nil
	}
	folderName, err := p.clients.ResourceManager.EnsureFolderExists(ctx, p.tenant.Metadata.Name, p.tenant.Spec.ParentFolder)
	if err != nil {
		return err
	}
	p.newState.Parent = folderName
	return nil
}
