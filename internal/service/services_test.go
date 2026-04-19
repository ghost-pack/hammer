package service

import (
	"testing"

	"github.com/ghost-pack/hammer/internal/dagger"
	"github.com/ghost-pack/hammer/internal/service/mocks"
	"github.com/stretchr/testify/assert"
)

func TestNewServices(t *testing.T) {
	tests := []struct {
		name string
		args struct{ client dagger.DaggerClient }
	}{
		{
			name: "creates services with a mock client",
			args: struct{ client dagger.DaggerClient }{client: &mocks.MockDaggerClient{}},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := NewServices(tt.args.client)

			assert.NotNil(t, got, "NewServices should return a Services instance")

			assert.NotNil(t, got.Build, "Build service should be initialised")
		})
	}
}
