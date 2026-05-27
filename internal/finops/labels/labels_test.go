package labels

import (
	"strconv"
	"testing"

	"github.com/ghost-pack/hammer/internal/oam"
	"github.com/stretchr/testify/require"
)

func TestBuilder_Build(t *testing.T) {
	type fields struct {
		App           *oam.App
		Env           string
		HammerVersion string
		Repo          string
		Commit        string
	}
	tests := []struct {
		name    string
		fields  fields
		want    Labels
		wantErr bool
	}{
		{
			name: "successful_label",
			fields: fields{
				App: &oam.App{
					APIVersion: "core.oam.dev/v1beta1",
					Kind:       "Applicationijpuapsdubhpasdufhbpasdiufhbpasdfiuhasdpfouhasdpfiubasdpfiubasdfpiubyasdfpihbasdfasdfasdf",
					Metadata: oam.Metadata{
						Name: "test",
						Annotations: map[string]string{
							"team":        "ghost-pack",
							"description": "The hammer CLI.",
						},
					},
					Spec: oam.Spec{
						Components: []oam.Component{
							{
								Name: "hammer",
								Type: "goservice",
							},
						},
					},
				},
				Env:           "dev",
				HammerVersion: "v1.0.0",
				Repo:          "ghost-pack",
				Commit:        "1234",
			},
			want: Labels{
				KeyApp:           "test",
				KeyKind:          "applicationijpuapsdubhpasdufhbpasdiufhbpasdfiuhasdpfouhasdpfiub",
				KeyEnv:           "dev",
				KeyTeam:          "ghost-pack",
				KeyManagedBy:     "hammer",
				KeyHammerVersion: "v1-0-0",
				KeyRepo:          "ghost-pack",
				KeyCommit:        "1234",
			},
		},
		{
			name: "failed_label_validation",
			fields: fields{
				App: &oam.App{
					APIVersion: "core.oam.dev/v1beta1",
					Kind:       "Application",
					Metadata: oam.Metadata{
						Annotations: map[string]string{
							"team":        "ghost-pack",
							"description": "The hammer CLI.",
						},
					},
					Spec: oam.Spec{
						Components: []oam.Component{
							{
								Name: "hammer",
								Type: "goservice",
							},
						},
					},
				},
				Env:           "dev",
				HammerVersion: "v1.0.0",
				Repo:          "ghost-pack",
				Commit:        "1234",
			},
			want:    nil,
			wantErr: true,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			b := &Builder{
				App:           tt.fields.App,
				Env:           tt.fields.Env,
				HammerVersion: tt.fields.HammerVersion,
				Repo:          tt.fields.Repo,
				Commit:        tt.fields.Commit,
			}
			got, err := b.Build()
			if (err != nil) != tt.wantErr {
				require.Error(t, err)
			}
			require.Equal(t, tt.want, got)
		})
	}
}

func TestBuilder_Validate(t *testing.T) {
	tests := []struct {
		name    string
		labels  Labels
		wantErr bool
		tooMany bool
	}{
		{
			name: "failed_validation_bad_key",
			labels: Labels{
				KeyApp:           "test",
				KeyKind:          "applicationijpuapsdubhpasdufhbpasdiufhbpasdfiuhasdpfouhasdpfiub",
				KeyEnv:           "dev",
				KeyTeam:          "ghost-pack",
				KeyManagedBy:     "hammer",
				KeyHammerVersion: "v1-0-0",
				KeyRepo:          "ghost-pack",
				KeyCommit:        "1234",
				"~~~~~~":         "whatever",
			},
			wantErr: true,
		},
		{
			name: "failed_validation_bad_value",
			labels: Labels{
				KeyApp:           "test",
				KeyKind:          "applicationijpuapsdubhpasdufhbpasdiufhbpasdfiuhasdpfouhasdpfiub",
				KeyEnv:           "dev",
				KeyTeam:          "ghost-pack",
				KeyManagedBy:     "hammer",
				KeyHammerVersion: "v1-0-0",
				KeyRepo:          "ghost-pack",
				KeyCommit:        "1234",
				"whatever":       "~~~~",
			},
			wantErr: true,
		},
		{
			name: "failed_validation_too_many_keys",
			labels: Labels{
				KeyApp:           "test",
				KeyKind:          "applicationijpuapsdubhpasdufhbpasdiufhbpasdfiuhasdpfouhasdpfiub",
				KeyEnv:           "dev",
				KeyTeam:          "ghost-pack",
				KeyManagedBy:     "hammer",
				KeyHammerVersion: "v1-0-0",
				KeyRepo:          "ghost-pack",
				KeyCommit:        "1234",
				"whatever":       "~~~~",
			},
			tooMany: true,
			wantErr: true,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if tt.tooMany {
				for i := 0; i < 100; i++ {
					tt.labels[strconv.Itoa(i)] = "value"
				}
			}
			err := tt.labels.Validate()
			if (err != nil) != tt.wantErr {
				require.Error(t, err)
			}
		})
	}
}
