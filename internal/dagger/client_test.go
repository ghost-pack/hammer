package dagger

import (
	"context"
	"os"
	"path/filepath"
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
			stdout, err := client.RunCommand(ctx, tt.image, tt.command)

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
				_, err := client.RunCommand(context.Background(), "alpine:latest", []string{"echo", "test"})
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

func TestDaggerClient_RunCommandWithMount(t *testing.T) {
	ctx := context.Background()
	client, err := NewDaggerClient(ctx)
	require.NoError(t, err, "failed to connect to Dagger engine")
	defer client.Close()

	// Create a temporary directory with a known file for mounting tests
	tmpDir := t.TempDir()
	testFile := filepath.Join(tmpDir, "test.txt")
	err = os.WriteFile(testFile, []byte("hello world"), 0644)
	require.NoError(t, err)

	tests := []struct {
		name        string
		image       string
		command     []string
		mountPath   string
		hostDir     string
		expectedOut string
		expectErr   bool
	}{
		{
			name:        "list files in mounted directory",
			image:       "alpine:latest",
			command:     []string{"ls"},
			mountPath:   "/mnt",
			hostDir:     tmpDir,
			expectedOut: "test.txt\n",
			expectErr:   false,
		},
		{
			name:        "cat file content from mounted directory",
			image:       "alpine:latest",
			command:     []string{"cat", "/mnt/test.txt"},
			mountPath:   "/mnt",
			hostDir:     tmpDir,
			expectedOut: "hello world",
			expectErr:   false,
		},
		{
			name:        "working directory is mount point",
			image:       "alpine:latest",
			command:     []string{"pwd"},
			mountPath:   "/workspace",
			hostDir:     tmpDir,
			expectedOut: "/workspace\n",
			expectErr:   false,
		},
		{
			name:        "non‑zero exit code in mounted container",
			image:       "alpine:latest",
			command:     []string{"sh", "-c", "exit 1"},
			mountPath:   "/mnt",
			hostDir:     tmpDir,
			expectedOut: "",
			expectErr:   true,
		},
		{
			name:        "host directory does not exist",
			image:       "alpine:latest",
			command:     []string{"ls"},
			mountPath:   "/mnt",
			hostDir:     "/nonexistent/path",
			expectedOut: "",
			expectErr:   true, // Dagger will fail to mount
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			stdout, err := client.RunCommandWithMount(ctx, tt.image, tt.command, tt.mountPath, tt.hostDir)

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
