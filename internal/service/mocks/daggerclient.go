package mocks

import (
	"context"

	"github.com/ghost-pack/hammer/internal/dagger"
	"github.com/stretchr/testify/mock"
)

type MockDaggerClient struct {
	dagger.DaggerClient
	mock.Mock
}

func (m *MockDaggerClient) RunCommand(ctx context.Context, image string, command []string) (string, error) {
	args := m.Called(ctx, image, command)
	if args.Get(0) == nil {
		return "", args.Error(1)
	}
	return args.Get(0).(string), args.Error(1)
}

func (m *MockDaggerClient) RunCommandWithMount(ctx context.Context, image string, command []string, mountPath, hostDir string) (string, error) {
	args := m.Called(ctx, image, command, mountPath, hostDir)
	if args.Get(0) == nil {
		return "", args.Error(1)
	}
	return args.Get(0).(string), args.Error(1)
}

func (m *MockDaggerClient) Close() error {
	args := m.Called()
	return args.Error(0)
}
