package tenant

import (
	"testing"

	"github.com/stretchr/testify/require"
)

func TestLoad(t *testing.T) {
	tests := []struct {
		name    string
		path    string
		want    *Tenant
		wantErr bool
	}{
		{
			name:    "successful_load",
			path:    "testdata/tenant_test_success.yaml",
			wantErr: false,
			want: &Tenant{
				APIVersion: "platform.hammerplatform.dev/v1alpha1",
				Kind:       "Tenant",
				Metadata: Metadata{
					Name: "acme-corp",
				},
				Spec: Spec{
					BillingAccount: "ABCDE-12345-FGHIJ",
					ParentFolder:   "folders/123456789",
					AllowedApis:    []string{"run.googleapis.com", "artifactregistry.googleapis.com", "logging.googleapis.com", "monitoring.googleapis.com"},
					Environments:   []string{"dev", "prod"},
				},
			},
		},
		{
			name:    "failed_load_failed_file_read",
			path:    "testdata/tenant_test_success.sdfasdfasdfasdf",
			wantErr: true,
			want:    nil,
		},
		{
			name:    "failed_load_failed_file_read",
			path:    "testdata/tenant_test_unparseable.yaml",
			wantErr: true,
			want:    nil,
		},
		{
			name:    "failed_load_validation",
			path:    "testdata/tenant_test_failed_validation.yaml",
			wantErr: true,
			want:    nil,
		},
		{
			name:    "failed_load_validation",
			path:    "testdata/tenant_test_failed_validation_2.yaml",
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
