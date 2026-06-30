package pipeline

import (
	"fmt"

	"github.com/ghost-pack/hammer/internal/docker"
	"github.com/ghost-pack/hammer/internal/gcp"
	"github.com/ghost-pack/hammer/internal/oam"
)

type Factory func(oam.Component, DependencyClients) (Pipeline, error)

type DependencyClients struct {
	DockerClient docker.Client
	GarClient    gcp.GarClient
	CloudBuild   gcp.CloudBuildClient
}

var registry = map[string]Factory{}

func Register(componentType string, f Factory) {
	if componentType == "" {
		panic("pipeline.Registry: componentType is empty")
	}
	if f == nil {
		panic("pipeline.Registry: factory is nil")
	}
	if _, exists := registry[componentType]; exists {
		panic("pipeline.Registry: componentType already registered")
	}
	registry[componentType] = f
}

func For(component oam.Component, dockerClient docker.Client, garClient gcp.GarClient, cloudBuildClient gcp.CloudBuildClient) (Pipeline, error) {
	if component.Type == "" {
		return nil, fmt.Errorf("componentType is nil")
	}

	f, ok := registry[component.Type]
	if !ok {
		return nil, fmt.Errorf("componentType %s is not registered", component.Type)
	}
	return f(component, DependencyClients{DockerClient: dockerClient, GarClient: garClient, CloudBuild: cloudBuildClient})
}
