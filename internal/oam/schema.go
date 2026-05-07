package oam

import "gopkg.in/yaml.v3"

type App struct {
	APIVersion string   `yaml:"apiVersion"`
	Kind       string   `yaml:"kind"`
	Metadata   Metadata `yaml:"metadata"`
	Spec       Spec     `yaml:"spec"`
	Source     string   `yaml:"-"`
}

type Metadata struct {
	Name        string            `yaml:"name"`
	Annotations map[string]string `json:"annotations,omitempty"`
}

type Spec struct {
	Components []Component `json:"components"`
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
