package pipeline

import (
	"context"
)

type Pipeline interface {
	ComponentType() string
	CI(ctx context.Context) error
	Analyze(ctx context.Context) error
	Deploy(ctx context.Context) error
}
