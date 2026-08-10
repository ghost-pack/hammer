package oam

import (
	"testing"

	"github.com/stretchr/testify/require"
	"gopkg.in/yaml.v3"
)

func yamlNode(t *testing.T, value string) yaml.Node {
	t.Helper()

	var document yaml.Node
	if err := yaml.Unmarshal([]byte(value), &document); err != nil {
		t.Fatalf("failed to create yaml node: %v", err)
	}
	// yaml.Unmarshal gives us a DocumentNode containing the actual value.
	return *document.Content[0]
}

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
					Policies: []Policy{
						{
							Name: "prod",
							Type: "environment",
							Properties: PolicyProperties{
								Environment: "prod",
								Overrides: []Override{
									{
										Component:  "hammer",
										Properties: yamlNode(t, `service_account: "sa-oam@hammer-central-prod.iam.gserviceaccount.com"`),
									},
								},
							},
						},
					},
				},
			},
		},
		{
			name:    "failed_load_missing_file",
			path:    "testdata/oam_test_success.sdfasdfasdfasdf",
			wantErr: true,
			want:    nil,
		},
		{
			name:    "failed_load_unparseable",
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
			name:    "failed_load_validation_2",
			path:    "testdata/oam_test_failed_validation_2.yaml",
			wantErr: true,
			want:    nil,
		},
		{
			name:    "failed_load_validation_3",
			path:    "testdata/oam_test_failed.yaml",
			wantErr: true,
			want:    nil,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := Load(tt.path)
			if tt.wantErr {
				require.Error(t, err)
				require.Nil(t, got)
				return
			}
			require.NoError(t, err)

			// Compare YAML output to ignore yaml.Node position/pointer differences.
			wantYAML, err := yaml.Marshal(tt.want)
			require.NoError(t, err)
			gotYAML, err := yaml.Marshal(got)
			require.NoError(t, err)
			require.Equal(t, string(wantYAML), string(gotYAML))
		})
	}
}
