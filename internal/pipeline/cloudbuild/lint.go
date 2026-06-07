package cloudbuild

import (
	"context"
	_ "embed"
	"encoding/json"
	"fmt"
	"os"

	"github.com/santhosh-tekuri/jsonschema/v6"
	"gopkg.in/yaml.v3"
)

//go:embed schema/cloudbuild.json
var cloudBuildSchema []byte

type Properties struct {
	Path string `yaml:"path"`
}

func (p *Pipeline) lint(_ context.Context) error {
	// TODO add span here.
	var properties Properties
	if err := p.component.Properties.Decode(&properties); err != nil {
		return fmt.Errorf("decoding properties: %w", err)
	}
	if properties.Path == "" {
		properties.Path = "./cloudbuild.yaml"
	}

	raw, err := os.ReadFile(properties.Path)
	if err != nil {
		return err
	}

	var doc interface{}
	if err := yaml.Unmarshal(raw, &doc); err != nil {
		return fmt.Errorf("invalid yaml: %w", err)
	}

	var schemaDoc any
	if err := json.Unmarshal(cloudBuildSchema, &schemaDoc); err != nil {
		return fmt.Errorf("loading schema: %w", err)
	}

	compiler := jsonschema.NewCompiler()
	if err := compiler.AddResource("cloudbuild.json", schemaDoc); err != nil {
		return fmt.Errorf("adding schema resource: %w", err)
	}

	schema, err := compiler.Compile("cloudbuild.json")
	if err != nil {
		return fmt.Errorf("compiling schema: %w", err)
	}
	return schema.Validate(doc)
}
