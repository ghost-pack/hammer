package goservice

import (
	"context"
	"fmt"
	"log/slog"

	"github.com/ghost-pack/hammer/internal/oam"
	"github.com/ghost-pack/hammer/internal/pipeline"
)

func init() {
	pipeline.Register("goservice", New)
}

func New(component oam.Component) (pipeline.Pipeline, error) {
	if component.Type != "goservice" {
		return nil, fmt.Errorf("goservice component must be of type goservice")
	}

	return &Pipeline{
		component: &component,
	}, nil
}

type Pipeline struct {
	component *oam.Component
}

func (p *Pipeline) ComponentType() string {
	return "goservice"
}

func (p *Pipeline) CI(ctx context.Context) error {
	slog.InfoContext(ctx, "oh my god finally we made it here")
	return nil
}
