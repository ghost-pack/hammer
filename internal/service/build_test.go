package service

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
)

func TestBuildService_Build(t *testing.T) {
	tests := []struct {
		name        string
		setupMock   func(*MockDaggerClient)
		expectError bool
	}{
		{
			name: "successful build",
			setupMock: func(m *MockDaggerClient) {
				m.On("RunCommand",
					mock.Anything, // context
					"alpine:latest",
					[]string{"echo", "Building with Dagger!"},
				).Return("Building with Dagger!\n", nil)
			},
			expectError: false,
		},
		{
			name: "run command fails",
			setupMock: func(m *MockDaggerClient) {
				m.On("RunCommand",
					mock.Anything,
					"alpine:latest",
					[]string{"echo", "Building with Dagger!"},
				).Return("", assert.AnError)
			},
			expectError: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mockClient := new(MockDaggerClient)
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
