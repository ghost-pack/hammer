package golang

import "github.com/ghost-pack/hammer/internal/service/observability"

var (
	tracer = observability.Tracer("github.com/ghost-pack/hammer/golang")
)
