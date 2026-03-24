package dagger

import (
	"context"
	"os"

	"dagger.io/dagger"
)

type DaggerClient interface {
	Container() *dagger.Container
	Close() error
}

type RealDaggerClient struct {
	client *dagger.Client
}

func NewDaggerClient(ctx context.Context) (DaggerClient, error) {
	client, err := dagger.Connect(ctx, dagger.WithLogOutput(os.Stderr))
	if err != nil {
		return nil, err
	}
	return &RealDaggerClient{client}, nil
}

func (r *RealDaggerClient) Container() *dagger.Container {
	return r.client.Container()
}

func (r *RealDaggerClient) Close() error {
	return r.client.Close()
}
