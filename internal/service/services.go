package service

import (
	"context"

	"github.com/ghost-pack/hammer/internal/dagger"
	"github.com/ghost-pack/hammer/internal/service/golang"
)

type BuildService interface {
	Build(ctx context.Context) error
}

type TestService interface {
	Test(ctx context.Context) error
}

type Services struct {
	Build BuildService
	Test  TestService
	// Add other services as needed, e.g.:
	//Terraform TerraformService
	//Node      NodeService
	//Trivy     TrivyService
}

func NewServices(client dagger.DaggerClient) *Services {
	return &Services{
		Build: golang.NewBuildService(client),
		Test:  golang.NewTestService(client),
		//Terraform: NewTerraformService(client),
		//Node:      NewNodeService(client),
		//Trivy:     NewTrivyService(client),
	}
}
