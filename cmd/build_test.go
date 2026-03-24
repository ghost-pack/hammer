package cmd

import (
	"context"
	"errors"
	"testing"

	"github.com/ghost-pack/hammer/internal/service"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
)

type mockBuildService struct {
	mock.Mock
}

func (m *mockBuildService) Build(ctx context.Context) error {
	args := m.Called(ctx)
	return args.Error(0)
}

func TestNewBuildCmd(t *testing.T) {
	tests := []struct {
		name      string
		mockError error
		expectErr bool
	}{
		{
			name:      "successful build",
			mockError: nil,
			expectErr: false,
		},
		{
			name:      "build fails",
			mockError: errors.New("build failed"),
			expectErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mockSvc := new(mockBuildService)
			mockSvc.On("Build", mock.Anything).Return(tt.mockError)

			cmd := NewBuildCmd(&service.Services{Build: mockSvc})

			err := cmd.RunE(cmd, nil)

			if tt.expectErr {
				assert.Error(t, err, "expected an error but got none")
			} else {
				assert.NoError(t, err, "expected no error but got one")
			}

			mockSvc.AssertExpectations(t)
		})
	}
}
