package pipeline

import (
	"context"
)

type Pipeline interface {
	// OAM Kind this pipeline can handle (GoService, Terraform, etc)
	ComponentType() string
	CI(ctx context.Context) error
}
