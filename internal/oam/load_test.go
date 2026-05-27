package oam

import (
	"testing"

	"github.com/stretchr/testify/require"
)

func TestLoad(t *testing.T) {
	tests := []struct {
		name    string
		path    string
		want    *App
		wantErr bool
	}{
		{
			name:    "successful_load",
			path:    "testdata/oam_test_success.yaml",
			wantErr: false,
			want: &App{
				APIVersion: "core.oam.dev/v1beta1",
				Kind:       "Application",
				Metadata: Metadata{
					Name: "test",
					Annotations: map[string]string{
						"team":        "ghost-pack",
						"description": "The hammer CLI.",
					},
				},
				Spec: Spec{
					Components: []Component{
						{
							Name: "hammer",
							Type: "goservice",
						},
					},
				},
			},
		},
		{
			name:    "failed_load_failed_file_read",
			path:    "testdata/oam_test_success.sdfasdfasdfasdf",
			wantErr: true,
			want:    nil,
		},
		{
			name:    "failed_load_failed_file_read",
			path:    "testdata/oam_unparseable.yaml",
			wantErr: true,
			want:    nil,
		},
		{
			name:    "failed_load_validation",
			path:    "testdata/oam_test_failed_validation.yaml",
			wantErr: true,
			want:    nil,
		},
		{
			name:    "failed_load_validation",
			path:    "testdata/oam_test_failed_validation_2.yaml",
			wantErr: true,
			want:    nil,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := Load(tt.path)
			if (err != nil) != tt.wantErr {
				require.Error(t, err)
			}
			require.Equal(t, tt.want, got)
		})
	}
}
