// internal/service/services.go
package service

import "dagger.io/dagger"

type Services struct {
	Build BuildService
	// Add other services as needed, e.g.:
	//Terraform TerraformService
	//Node      NodeService
	//Trivy     TrivyService
}

func NewServices(client *dagger.Client) *Services {
	return &Services{
		Build: NewBuildService(client),
		//Terraform: NewTerraformService(client),
		//Node:      NewNodeService(client),
		//Trivy:     NewTrivyService(client),
	}
}
