package provisioner

import (
	"context"
	"testing"

	"github.com/ghost-pack/hammer/internal/gcp"
	"github.com/ghost-pack/hammer/internal/gcp/project"
	"github.com/ghost-pack/hammer/internal/tenant"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
)

type MockResourceManager struct {
	mock.Mock
}

func (m *MockResourceManager) EnsureFolderExists(ctx context.Context, displayName, parent string) (string, error) {
	callArgs := m.Called(ctx, displayName, parent)
	res, _ := callArgs.Get(0).(string)
	return res, callArgs.Error(1)
}

func (m *MockResourceManager) EnsureProjectExists(ctx context.Context, projectID, displayName, parent string) (string, error) {
	callArgs := m.Called(ctx, projectID, displayName, parent)
	res, _ := callArgs.Get(0).(string)
	return res, callArgs.Error(1)
}

func (m *MockResourceManager) Close() error {
	callArgs := m.Called()
	return callArgs.Error(0)
}

type MockServiceUsage struct {
	mock.Mock
}

func (m *MockServiceUsage) EnableAPIs(ctx context.Context, projectID string, apis []string) error {
	callArgs := m.Called(ctx, projectID, apis)
	return callArgs.Error(0)
}

func (m *MockServiceUsage) Close() error {
	callArgs := m.Called()
	return callArgs.Error(0)
}

type MockOrgPolicy struct {
	mock.Mock
}

func (m *MockOrgPolicy) EnforcePolicy(ctx context.Context, resource, constraint string) error {
	callArgs := m.Called(ctx, resource, constraint)
	return callArgs.Error(0)
}

func (m *MockOrgPolicy) Close() error {
	callArgs := m.Called()
	return callArgs.Error(0)
}

type MockIam struct {
	mock.Mock
}

func (m *MockIam) EnsureServiceAccountExists(ctx context.Context, projectID, name, displayName string) (string, error) {
	callArgs := m.Called(ctx, projectID, name, displayName)
	res, _ := callArgs.Get(0).(string)
	return res, callArgs.Error(1)
}

func (m *MockIam) BindProjectRoles(ctx context.Context, projectID, saEmail string, roles []string) error {
	callArgs := m.Called(ctx, projectID, projectID, saEmail, roles)
	return callArgs.Error(0)
}

func (m *MockIam) UnbindProjectRoles(ctx context.Context, projectID, saEmail string, roles []string) error {
	callArgs := m.Called(ctx, projectID, projectID, saEmail, roles)
	return callArgs.Error(0)
}

func (m *MockIam) BindOrgRoles(ctx context.Context, projectID, saEmail string, roles []string) error {
	callArgs := m.Called(ctx, projectID, projectID, saEmail, roles)
	return callArgs.Error(0)
}

func (m *MockIam) UnbindOrgRoles(ctx context.Context, projectID, saEmail string, roles []string) error {
	callArgs := m.Called(ctx, projectID, projectID, saEmail, roles)
	return callArgs.Error(0)
}

func (m *MockIam) AllowImpersonation(ctx context.Context, projectID, targetSAEmail, impersonatorEmail string) error {
	callArgs := m.Called(ctx, projectID, projectID, targetSAEmail, impersonatorEmail)
	return callArgs.Error(0)
}

func (m *MockIam) Close() error {
	callArgs := m.Called()
	return callArgs.Error(0)
}

type MockCloudStorage struct {
	mock.Mock
}

func (m *MockCloudStorage) EnsureBucketExists(ctx context.Context, projectId, location, bucketName string) error {
	callArgs := m.Called(ctx, projectId, location, bucketName)
	return callArgs.Error(0)
}

func (m *MockCloudStorage) GetObject(ctx context.Context, bucket, object string) ([]byte, error) {
	callArgs := m.Called(ctx, bucket, object)
	res, _ := callArgs.Get(0).([]byte)
	return res, callArgs.Error(1)
}

func (m *MockCloudStorage) WriteObject(ctx context.Context, bucket, object string, data []byte, metadata map[string]string) error {
	callArgs := m.Called(ctx, bucket, object, data, metadata)
	return callArgs.Error(0)
}

func (m *MockCloudStorage) Close() error {
	callArgs := m.Called()
	return callArgs.Error(0)
}

type MockBilling struct {
	mock.Mock
}

func (m *MockBilling) LinkBillingAccount(ctx context.Context, projectID, billingAccount string) error {
	callArgs := m.Called(ctx, projectID, billingAccount)
	return callArgs.Error(0)
}

func (m *MockBilling) Close() error {
	callArgs := m.Called()
	return callArgs.Error(0)
}

type MockProvisioner struct {
	mock.Mock
}

func (m *MockProvisioner) Apply(ctx context.Context) error {
	args := m.Called()
	return args.Error(0)
}

func TestFor(t *testing.T) {
	type args struct {
		tenant          tenant.Tenant
		resourceManager project.ResourceManagerClient
		serviceUsage    project.ServiceUsageClient
		orgPolicy       project.OrgPolicyClient
		billing         project.BillingClient
		iam             project.IAMClient
		cloudStorage    gcp.CloudStorageClient
	}
	tests := []struct {
		name    string
		args    args
		setup   func()
		want    Provisioner
		wantErr bool
	}{
		{
			name: "SuccessfulFor",
			args: args{
				tenant:          tenant.Tenant{Metadata: tenant.Metadata{Name: "test"}, Kind: "tenant"},
				resourceManager: &MockResourceManager{},
				serviceUsage:    &MockServiceUsage{},
				orgPolicy:       &MockOrgPolicy{},
				billing:         &MockBilling{},
				iam:             &MockIam{},
				cloudStorage:    &MockCloudStorage{},
			},
			want: &MockProvisioner{},
			setup: func() {
				Register("tenant", func(tenant *tenant.Tenant, client *DependencyClients) (Provisioner, error) {
					return &MockProvisioner{}, nil
				})
			},
			wantErr: false,
		},
		{
			name: "Failure_NilType",
			args: args{
				tenant:          tenant.Tenant{Metadata: tenant.Metadata{Name: "test"}, Kind: ""},
				resourceManager: &MockResourceManager{},
				serviceUsage:    &MockServiceUsage{},
				orgPolicy:       &MockOrgPolicy{},
				billing:         &MockBilling{},
				iam:             &MockIam{},
				cloudStorage:    &MockCloudStorage{},
			},
			want:    &MockProvisioner{},
			setup:   func() {},
			wantErr: true,
		},
		{
			name: "Failure_UnregisteredType",
			args: args{
				tenant:          tenant.Tenant{Metadata: tenant.Metadata{Name: "test"}, Kind: "tenant"},
				resourceManager: &MockResourceManager{},
				serviceUsage:    &MockServiceUsage{},
				orgPolicy:       &MockOrgPolicy{},
				billing:         &MockBilling{},
				iam:             &MockIam{},
				cloudStorage:    &MockCloudStorage{},
			},
			want:    &MockProvisioner{},
			setup:   func() {},
			wantErr: true,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			registry = map[string]Factory{}
			tt.setup()
			got, err := For(&tt.args.tenant, &DependencyClients{ResourceManager: tt.args.resourceManager, ServiceUsage: tt.args.serviceUsage, OrgPolicy: tt.args.orgPolicy, CloudStorage: tt.args.cloudStorage, IAM: tt.args.iam, Billing: tt.args.billing})
			if err != nil {
				if tt.wantErr {
					require.Error(t, err)
				} else {
					require.Nil(t, err)
				}
			} else {
				require.Equal(t, tt.want, got)
			}
		})
	}
}

func TestRegister(t *testing.T) {
	type args struct {
		kind string
		f    Factory
	}
	tests := []struct {
		name         string
		args         args
		preRegister  bool
		panicMessage string
	}{
		{
			name: "SuccessRegister",
			args: args{
				kind: "Tenant",
				f: func(tenant *tenant.Tenant, client *DependencyClients) (Provisioner, error) {
					return &MockProvisioner{}, nil
				},
			},
			preRegister:  false,
			panicMessage: "",
		},
		{
			name: "FailedRegister_duplicate",
			args: args{
				kind: "Tenant",
				f: func(tenant *tenant.Tenant, client *DependencyClients) (Provisioner, error) {
					return &MockProvisioner{}, nil
				},
			},
			preRegister:  true,
			panicMessage: "provisioner.Register: kind already registered: Tenant",
		},
		{
			name: "FailedRegister_noFactory",
			args: args{
				kind: "goservice",
				f:    nil,
			},
			preRegister:  false,
			panicMessage: "provisioner.Register: factory is nil",
		},
		{
			name: "FailedRegister_noComponentType",
			args: args{
				kind: "",
				f: func(tenant *tenant.Tenant, client *DependencyClients) (Provisioner, error) {
					return &MockProvisioner{}, nil
				},
			},
			preRegister:  false,
			panicMessage: "provisioner.Register: kind is empty",
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			registry = map[string]Factory{}
			if tt.preRegister {
				Register(tt.args.kind, tt.args.f)
			}

			call := func() {
				Register(tt.args.kind, tt.args.f)
			}

			if tt.panicMessage != "" {
				require.PanicsWithValue(t, tt.panicMessage, call)
			} else {
				require.NotPanics(t, call)
			}
		})
	}
}
