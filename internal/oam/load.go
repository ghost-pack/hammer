package oam

import (
	"bytes"
	"fmt"
	"os"
	"path/filepath"

	"gopkg.in/yaml.v3"
)

const DefaultFilename = "oam.yaml"

func Load(path string) (*App, error) {
	if path == "" {
		path = DefaultFilename
	}

	abs, err := filepath.Abs(path)
	if err != nil {
		return nil, err
	}

	data, err := os.ReadFile(abs)
	if err != nil {
		return nil, err
	}

	app, err := parse(data)
	if err != nil {
		return nil, err
	}

	err = Validate(app)
	if err != nil {
		fmt.Printf("error: %v\n", err)
		return nil, err
	}
	return app, nil
}

func parse(data []byte) (*App, error) {
	var app App
	dec := yaml.NewDecoder(bytes.NewReader(data))
	dec.KnownFields(true)
	if err := dec.Decode(&app); err != nil {
		return nil, err
	}
	return &app, nil
}
