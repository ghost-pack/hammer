package golang

import (
	"context"
	"testing"

	"github.com/ghost-pack/hammer/internal/service/mocks"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
)

func TestBuildService_Build(t *testing.T) {
	tests := []struct {
		name        string
		setupMock   func(*mocks.MockDaggerClient)
		expectError bool
	}{
		{
			name: "successful build",
			setupMock: func(m *mocks.MockDaggerClient) {
				m.On("RunCommandWithMount",
					mock.Anything,
					"cgr.dev/chainguard/go",
					[]string{"go", "build", "."},
					"/src",
					".",
				).Return("Building with Dagger!\n", nil)
			},
			expectError: false,
		},
		{
			name: "run command fails",
			setupMock: func(m *mocks.MockDaggerClient) {
				m.On("RunCommandWithMount",
					mock.Anything,
					"cgr.dev/chainguard/go",
					[]string{"go", "build", "."},
					"/src",
					".",
				).Return("", assert.AnError)
			},
			expectError: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mockClient := new(mocks.MockDaggerClient)
			tt.setupMock(mockClient)

			svc := NewBuildService(mockClient)

			err := svc.Build(context.Background())

			if tt.expectError {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
			}

			mockClient.AssertExpectations(t)
		})
	}
}
