package oam

import "gopkg.in/yaml.v3"

type App struct {
	APIVersion string   `yaml:"apiVersion"`
	Kind       string   `yaml:"kind"`
	Metadata   Metadata `yaml:"metadata"`
	Spec       Spec     `yaml:"spec"`
}

type Metadata struct {
	Name        string            `yaml:"name"`
	Annotations map[string]string `yaml:"annotations,omitempty"`
}

type Spec struct {
	Components []Component `yaml:"components"`
	Policies   []Policy    `yaml:"policies,omitempty"`
}

type Component struct {
	Name       string    `yaml:"name"`
	Type       string    `yaml:"type"`
	Properties yaml.Node `yaml:"properties,omitempty"`
	Traits     []Trait   `yaml:"traits,omitempty"`
}

type Trait struct {
	Type       string    `yaml:"type"`
	Properties yaml.Node `yaml:"properties,omitempty"`
}

type Policy struct {
	Name       string           `yaml:"name"`
	Type       string           `yaml:"type"`
	Properties PolicyProperties `yaml:"properties,omitempty"`
}

type PolicyProperties struct {
	Environment string     `yaml:"environment"`
	Overrides   []Override `yaml:"overrides,omitempty"`
}

type Override struct {
	Component  string    `yaml:"component"`
	Properties yaml.Node `yaml:"properties,omitempty"`
	Traits     []Trait   `yaml:"traits,omitempty"`
}
