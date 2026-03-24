package dagger

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestDaggerClient_Integration(t *testing.T) {
	ctx := context.Background()
	client, err := NewDaggerClient(ctx)
	require.NoError(t, err, "failed to connect to Dagger engine")
	defer client.Close()

	tests := []struct {
		name        string
		image       string
		command     []string
		expectedOut string
		expectErr   bool
	}{
		{
			name:        "simple echo",
			image:       "alpine:latest",
			command:     []string{"echo", "hello"},
			expectedOut: "hello\n",
			expectErr:   false,
		},
		{
			name:        "multi‑line output",
			image:       "alpine:latest",
			command:     []string{"sh", "-c", "echo line1; echo line2"},
			expectedOut: "line1\nline2\n",
			expectErr:   false,
		},
		{
			name:        "non‑zero exit code",
			image:       "alpine:latest",
			command:     []string{"sh", "-c", "exit 1"},
			expectedOut: "",
			expectErr:   true, // Dagger returns an error when the command fails
		},
		{
			name:        "image does not exist",
			image:       "thisimagedoesnotexist:latest",
			command:     []string{"echo", "foo"},
			expectedOut: "",
			expectErr:   true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			stdout, err := client.Container().
				From(tt.image).
				WithExec(tt.command).
				Stdout(ctx)

			if tt.expectErr {
				assert.Error(t, err, "expected an error but got none")
				assert.Equal(t, tt.expectedOut, stdout, "stdout mismatch")
			} else {
				assert.NoError(t, err, "expected no error but got one")
				assert.Equal(t, tt.expectedOut, stdout, "stdout mismatch")
			}
		})
	}
}

func TestDaggerClient_Close(t *testing.T) {
	tests := []struct {
		name      string
		setup     func(t *testing.T) DaggerClient
		run       func(t *testing.T, client DaggerClient) error
		expectErr bool
	}{
		{
			name: "using client after close",
			setup: func(t *testing.T) DaggerClient {
				ctx := context.Background()
				client, err := NewDaggerClient(ctx)
				require.NoError(t, err)
				err = client.Close()
				require.NoError(t, err)
				return client
			},
			run: func(t *testing.T, client DaggerClient) error {
				_, err := client.Container().
					From("alpine:latest").
					WithExec([]string{"echo", "test"}).
					Stdout(context.Background())
				return err
			},
			expectErr: true,
		},
		{
			name: "close twice",
			setup: func(t *testing.T) DaggerClient {
				ctx := context.Background()
				client, err := NewDaggerClient(ctx)
				require.NoError(t, err)
				return client
			},
			run: func(t *testing.T, client DaggerClient) error {
				err := client.Close()
				assert.NoError(t, err)
				err = client.Close()
				return err
			},
			expectErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			client := tt.setup(t)
			err := tt.run(t, client)
			if tt.expectErr {
				assert.Error(t, err, tt.name)
			} else {
				assert.NoError(t, err, tt.name)
			}
		})
	}
}

func TestNewDaggerClient_ConnectionError(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	cancel() // immediately cancel

	client, err := NewDaggerClient(ctx)
	assert.Error(t, err, "expected connection error with cancelled context")
	assert.Nil(t, client, "client should be nil on error")
}
