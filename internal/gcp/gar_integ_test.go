package gcp

import (
	"context"
	"testing"

	"github.com/stretchr/testify/require"
	"google.golang.org/api/option"
)

func TestGarClientImpl_Close(t *testing.T) {
	tests := []struct {
		name           string
		setupGarClient func(ctx context.Context, opts ...option.ClientOption) (GarClient, error)
		wantErr        bool
	}{
		{
			name: "successful close",
			setupGarClient: func(ctx context.Context, opts ...option.ClientOption) (GarClient, error) {
				client, err := NewGarClient(ctx, opts...)
				if err != nil {
					return nil, err
				}
				return client, nil
			},
			wantErr: false,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			g, creationErr := tt.setupGarClient(context.Background())
			defer g.Close()
			require.NoError(t, creationErr)
			if err := g.Close(); (err != nil) != tt.wantErr {
				require.Error(t, err)
			} else {
				require.NoError(t, err)
			}
		})
	}
}

func TestGarClientImpl_EnsureRepository(t *testing.T) {
	tests := []struct {
		name           string
		setupGarClient func(ctx context.Context, opts ...option.ClientOption) (GarClient, error)
		projectID      string
		wantErr        bool
	}{
		{
			name: "Repository Exists",
			setupGarClient: func(ctx context.Context, opts ...option.ClientOption) (GarClient, error) {
				client, err := NewGarClient(ctx, opts...)
				if err != nil {
					return nil, err
				}
				return client, nil
			},
			projectID: "cloud-build-pipeline-396819",
			wantErr:   false,
		},
		{
			name: "Repository does not exist",
			setupGarClient: func(ctx context.Context, opts ...option.ClientOption) (GarClient, error) {
				client, err := NewGarClient(ctx, opts...)
				if err != nil {
					return nil, err
				}
				return client, nil
			},
			projectID: "adsfasdf",
			wantErr:   true,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			g, creationErr := tt.setupGarClient(context.Background())
			defer g.Close()
			require.NoError(t, creationErr)

			err := g.EnsureRepository(context.Background(), tt.projectID, "us-central1", "hammer")
			if err != nil && tt.wantErr {
				require.Error(t, err)
			} else {
				require.NoError(t, err)
			}
		})
	}
}

func TestNewGarClient(t *testing.T) {
	tests := []struct {
		name           string
		setupGarClient func(ctx context.Context, opts ...option.ClientOption) (GarClient, error)
		wantErr        bool
	}{
		{
			name: "failed client creation",
			setupGarClient: func(ctx context.Context, opts ...option.ClientOption) (GarClient, error) {
				client, err := NewGarClient(ctx, opts...)
				if err != nil {
					return nil, err
				}
				return client, nil
			},
			wantErr: true,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, creationErr := tt.setupGarClient(context.Background(), option.WithCredentialsFile("/nonexistent/credentials.json"))
			if creationErr != nil && tt.wantErr {
				require.Error(t, creationErr)
			} else {
				require.NoError(t, creationErr)
			}
		})
	}
}
