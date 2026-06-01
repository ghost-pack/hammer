package pipeline

import (
	"fmt"

	"github.com/ghost-pack/hammer/internal/oam"
	"github.com/moby/moby/client"
)

type Factory func(oam.Component, client.APIClient) (Pipeline, error)

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

func For(component oam.Component, dockerClient client.APIClient) (Pipeline, error) {
	if component.Type == "" {
		return nil, fmt.Errorf("componentType is nil")
	}

	f, ok := registry[component.Type]
	if !ok {
		return nil, fmt.Errorf("componentType %s is not registered", component.Type)
	}
	return f(component, dockerClient)
}
