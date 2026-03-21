package cmd

import (
	"context"
	"os"

	"dagger.io/dagger"
)

func NewDaggerClient(ctx context.Context) (*dagger.Client, error) {
	return dagger.Connect(ctx, dagger.WithLogOutput(os.Stderr))
}
