package gcp

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"reflect"
	"testing"

	"cloud.google.com/go/storage"
	"github.com/ghost-pack/hammer/internal/provisioner"
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

func TestNewProvisioner(t *testing.T) {
	type args struct {
		tenant *tenant.Tenant
		client *provisioner.DependencyClients
	}
	tests := []struct {
		name    string
		args    args
		want    provisioner.Provisioner
		wantErr bool
	}{
		{
			name: "SuccessfulNewProvisioner",
			args: args{tenant: &tenant.Tenant{Metadata: tenant.Metadata{Name: "acme-corp"}}, client: &provisioner.DependencyClients{}},
			want: &Provisioner{
				tenant:           &tenant.Tenant{Metadata: tenant.Metadata{Name: "acme-corp"}},
				clients:          &provisioner.DependencyClients{},
				registryBucket:   "hammer-platform-registry",
				platformProject:  "hammer-central-prod",
				defaultRegion:    "us-central1",
				newState:         &TenantState{},
				lastAppliedState: &TenantState{},
			},
			wantErr: false,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := New(tt.args.tenant, &provisioner.DependencyClients{})
			if err != nil {
				if tt.wantErr {
					require.Error(t, err)
				} else {
					t.Errorf("New() error = %v, wantErr %v", err, tt.wantErr)
				}
			}
			require.Equal(t, tt.want, got)
		})
	}
}

func TestProvisionerApply(t *testing.T) {
	tests := []struct {
		name      string
		tenant    *tenant.Tenant
		setupMock func(*MockResourceManager, *MockServiceUsage, *MockOrgPolicy, *MockIam, *MockBilling, *MockCloudStorage)
		wantErr   bool
	}{
		{
			name: "Successful Apply",
			tenant: &tenant.Tenant{
				APIVersion: "core.oam.dev/v1beta1",
				Kind:       "Tenant",
				Metadata:   tenant.Metadata{Name: "acme-corp"},
				Spec: tenant.Spec{
					BillingAccount: "ABCDE-12345-FGHIJ",
					ParentFolder:   "937506553540",
					AllowedApis: []string{
						"run.googleapis.com",
						"artifactregistry.googleapis.com",
						"logging.googleapis.com",
						"monitoring.googleapis.com",
					},
					Environments: []string{
						"dev",
						"prod",
					},
				},
			},
			setupMock: func(mockResourceManager *MockResourceManager, mockServiceUsage *MockServiceUsage, mockOrgPolicy *MockOrgPolicy, mockIam *MockIam, mockBilling *MockBilling, mockCloudStorage *MockCloudStorage) {
				mockCloudStorage.On("EnsureBucketExists", mock.Anything, mock.Anything, mock.Anything, mock.Anything).
					Return(nil)
				mockCloudStorage.On("GetObject", mock.Anything, mock.Anything, mock.Anything).
					Return(nil, storage.ErrObjectNotExist)
				mockResourceManager.On("EnsureFolderExists", mock.Anything, mock.Anything, mock.Anything).
					Return("acme-corp", nil)
				mockResourceManager.On("EnsureProjectExists", mock.Anything, mock.Anything, mock.Anything, mock.Anything).
					Return("12345", nil)
				mockBilling.On("LinkBillingAccount", mock.Anything, mock.Anything, mock.Anything).
					Return(nil)
				mockOrgPolicy.On("EnforcePolicy", mock.Anything, mock.Anything, mock.Anything).
					Return(nil)
				mockServiceUsage.On("EnableAPIs", mock.Anything, mock.Anything, mock.Anything).
					Return(nil)
				mockIam.On("EnsureServiceAccountExists", mock.Anything, mock.Anything, mock.Anything, mock.Anything).
					Return("sa-pipeline@acme-corp-dev.iam.gserviceaccount.com", nil).Once()
				mockIam.On("EnsureServiceAccountExists", mock.Anything, mock.Anything, mock.Anything, mock.Anything).
					Return("sa-pipeline@acme-corp-prod.iam.gserviceaccount.com", nil).Once()
				mockIam.On("BindProjectRoles", mock.Anything, mock.Anything, mock.Anything, mock.Anything, mock.Anything).
					Return(nil)
				stateBytes, _ := os.ReadFile("testdata/expectedFinalState.json")
				mockCloudStorage.On("WriteObject",
					mock.Anything,
					mock.Anything,
					"tenants/acme-corp/state.json",
					mock.MatchedBy(func(data []byte) bool {
						var expected, actual TenantState
						if err := json.Unmarshal(stateBytes, &expected); err != nil {
							return false
						}
						if err := json.Unmarshal(data, &actual); err != nil {
							return false
						}
						actual.AppliedAt = expected.AppliedAt
						t.Logf("expected: %+v", expected)
						t.Logf("actual:   %+v", actual)
						return reflect.DeepEqual(expected, actual)
					}),
					mock.Anything,
				).Return(nil).Once()
				mockCloudStorage.On("WriteObject", mock.Anything, mock.Anything, mock.Anything, mock.Anything, mock.Anything).
					Return(nil)
			},
			wantErr: false,
		},
		{
			name: "Successful Apply with existing previous state",
			tenant: &tenant.Tenant{
				APIVersion: "core.oam.dev/v1beta1",
				Kind:       "Tenant",
				Metadata:   tenant.Metadata{Name: "acme-corp"},
				Spec: tenant.Spec{
					BillingAccount: "ABCDE-12345-FGHIJ",
					ParentFolder:   "937506553540",
					AllowedApis: []string{
						"run.googleapis.com",
						"artifactregistry.googleapis.com",
						"logging.googleapis.com",
						"monitoring.googleapis.com",
					},
					Environments: []string{
						"dev",
						"prod",
					},
				},
			},
			setupMock: func(mockResourceManager *MockResourceManager, mockServiceUsage *MockServiceUsage, mockOrgPolicy *MockOrgPolicy, mockIam *MockIam, mockBilling *MockBilling, mockCloudStorage *MockCloudStorage) {
				mockCloudStorage.On("EnsureBucketExists", mock.Anything, mock.Anything, mock.Anything, mock.Anything).
					Return(nil)
				stateBytes, _ := os.ReadFile("testdata/lastAppliedState.json")
				mockCloudStorage.On("GetObject", mock.Anything, mock.Anything, mock.Anything).
					Return(stateBytes, nil)
				mockResourceManager.On("EnsureFolderExists", mock.Anything, mock.Anything, mock.Anything).
					Return("acme-corp", nil)
				mockResourceManager.On("EnsureProjectExists", mock.Anything, mock.Anything, mock.Anything, mock.Anything).
					Return("12345", nil)
				mockBilling.On("LinkBillingAccount", mock.Anything, mock.Anything, mock.Anything).
					Return(nil)
				mockOrgPolicy.On("EnforcePolicy", mock.Anything, mock.Anything, mock.Anything).
					Return(nil)
				mockServiceUsage.On("EnableAPIs", mock.Anything, mock.Anything, mock.Anything).
					Return(nil)
				mockIam.On("EnsureServiceAccountExists", mock.Anything, mock.Anything, mock.Anything, mock.Anything).
					Return("sa-pipeline@acme-corp-dev.iam.gserviceaccount.com", nil).Once()
				mockIam.On("EnsureServiceAccountExists", mock.Anything, mock.Anything, mock.Anything, mock.Anything).
					Return("sa-pipeline@acme-corp-prod.iam.gserviceaccount.com", nil).Once()
				mockIam.On("BindProjectRoles", mock.Anything, mock.Anything, mock.Anything, mock.Anything, mock.Anything).
					Return(nil)
				mockIam.On("UnbindProjectRoles", mock.Anything, mock.Anything, mock.Anything, mock.Anything, mock.Anything).
					Return(nil)
				expectedFinalState, _ := os.ReadFile("testdata/expectedFinalState.json")
				mockCloudStorage.On("WriteObject",
					mock.Anything,
					mock.Anything,
					"tenants/acme-corp/state.json",
					mock.MatchedBy(func(data []byte) bool {
						var expected, actual TenantState
						if err := json.Unmarshal(expectedFinalState, &expected); err != nil {
							return false
						}
						if err := json.Unmarshal(data, &actual); err != nil {
							return false
						}
						actual.AppliedAt = expected.AppliedAt
						return reflect.DeepEqual(expected, actual)
					}),
					mock.Anything,
				).Return(nil).Once()
				mockCloudStorage.On("WriteObject", mock.Anything, mock.Anything, mock.Anything, mock.Anything, mock.Anything).
					Return(nil)
			},
			wantErr: false,
		},
		{
			name: "failed apply unsupported api",
			tenant: &tenant.Tenant{
				APIVersion: "core.oam.dev/v1beta1",
				Kind:       "Tenant",
				Metadata:   tenant.Metadata{Name: "acme-corp"},
				Spec: tenant.Spec{
					BillingAccount: "ABCDE-12345-FGHIJ",
					ParentFolder:   "937506553540",
					AllowedApis: []string{
						"run.googleapis.com",
						"artifactregistry.googleapis.com",
						"logging.googleapis.com",
						"monitoringasdfasdfasdf.googleapis.com",
					},
					Environments: []string{
						"dev",
						"prod",
					},
				},
			},
			setupMock: func(mockResourceManager *MockResourceManager, mockServiceUsage *MockServiceUsage, mockOrgPolicy *MockOrgPolicy, mockIam *MockIam, mockBilling *MockBilling, mockCloudStorage *MockCloudStorage) {
				mockCloudStorage.On("EnsureBucketExists", mock.Anything, mock.Anything, mock.Anything, mock.Anything).
					Return(nil)
				stateBytes, _ := os.ReadFile("testdata/lastAppliedState.json")
				mockCloudStorage.On("GetObject", mock.Anything, mock.Anything, mock.Anything).
					Return(stateBytes, nil)
				mockResourceManager.On("EnsureFolderExists", mock.Anything, mock.Anything, mock.Anything).
					Return("acme-corp", nil)
				mockResourceManager.On("EnsureProjectExists", mock.Anything, mock.Anything, mock.Anything, mock.Anything).
					Return("12345", nil)
				mockBilling.On("LinkBillingAccount", mock.Anything, mock.Anything, mock.Anything).
					Return(nil)
				mockIam.On("EnsureServiceAccountExists", mock.Anything, mock.Anything, mock.Anything, mock.Anything).
					Return("sa-pipeline@acme-corp-dev.iam.gserviceaccount.com", nil).Once()
				mockIam.On("EnsureServiceAccountExists", mock.Anything, mock.Anything, mock.Anything, mock.Anything).
					Return("sa-pipeline@acme-corp-prod.iam.gserviceaccount.com", nil).Once()
			},
			wantErr: true,
		},
		{
			name: "Successful Apply central tenant",
			tenant: &tenant.Tenant{
				APIVersion: "core.oam.dev/v1beta1",
				Kind:       "Tenant",
				Metadata:   tenant.Metadata{Name: "hammer-central"},
				Spec: tenant.Spec{
					BillingAccount: "ABCDE-12345-FGHIJ",
					ParentFolder:   "937506553540",
					AllowedApis: []string{
						"cloudbuild.googleapis.com",
						"artifactregistry.googleapis.com",
						"storage.googleapis.com",
						"secretmanager.googleapis.com",
						"logging.googleapis.com",
						"monitoring.googleapis.com",
						"pubsub.googleapis.com",
						"cloudtrace.googleapis.com",
					},
					Environments: []string{
						"prod",
					},
					ServiceAccounts: []tenant.ServiceAccountSpec{
						{
							Name:        "sa-provisioner",
							Description: "provisioner",
							Roles: tenant.SARoleBinding{
								Project: []string{"roles/storage.objectAdmin"},
								Organization: []string{
									"roles/resourcemanager.projectCreator",
									"roles/resourcemanager.folderCreator",
									"roles/orgpolicy.policyAdmin",
									"roles/iam.serviceAccountAdmin",
									"roles/serviceusage.serviceUsageAdmin",
									"roles/billing.user",
									"roles/resourcemanager.organizationAdmin",
								},
							},
						},
						{
							Name:        "sa-oam",
							Description: "Runs CI/CD pipelines interpreting OAM files",
							Roles: tenant.SARoleBinding{
								Project: []string{
									"roles/iam.serviceAccountTokenCreator",
									"roles/artifactregistry.writer",
									"roles/cloudbuild.builds.editor",
									"roles/storage.objectAdmin",
									"roles/pubsub.publisher",
									"roles/pubsub.subscriber",
								},
							},
						},
					},
				},
			},
			setupMock: func(mockResourceManager *MockResourceManager, mockServiceUsage *MockServiceUsage, mockOrgPolicy *MockOrgPolicy, mockIam *MockIam, mockBilling *MockBilling, mockCloudStorage *MockCloudStorage) {
				mockCloudStorage.On("EnsureBucketExists", mock.Anything, mock.Anything, mock.Anything, mock.Anything).
					Return(nil)
				mockCloudStorage.On("GetObject", mock.Anything, mock.Anything, mock.Anything).
					Return(nil, storage.ErrObjectNotExist)
				mockResourceManager.On("EnsureProjectExists", mock.Anything, mock.Anything, mock.Anything, mock.Anything).
					Return("12345", nil)
				mockBilling.On("LinkBillingAccount", mock.Anything, mock.Anything, mock.Anything).
					Return(nil)
				mockOrgPolicy.On("EnforcePolicy", mock.Anything, mock.Anything, mock.Anything).
					Return(nil)
				mockServiceUsage.On("EnableAPIs", mock.Anything, mock.Anything, mock.Anything).
					Return(nil)
				mockIam.On("EnsureServiceAccountExists", mock.Anything, mock.Anything, mock.Anything, mock.Anything).
					Return("sa-pipeline@hammer-central-prod.iam.gserviceaccount.com", nil).Once()
				mockIam.On("EnsureServiceAccountExists", mock.Anything, mock.Anything, mock.Anything, mock.Anything).
					Return("sa-provisioner@hammer-central-prod.iam.gserviceaccount.com", nil).Once()
				mockIam.On("EnsureServiceAccountExists", mock.Anything, mock.Anything, mock.Anything, mock.Anything).
					Return("sa-oam@hammer-central-prod.iam.gserviceaccount.com", nil).Once()
				mockIam.On("BindProjectRoles", mock.Anything, mock.Anything, mock.Anything, mock.Anything, mock.Anything).
					Return(nil)
				mockIam.On("BindOrgRoles", mock.Anything, mock.Anything, mock.Anything, mock.Anything, mock.Anything).
					Return(nil)
				stateBytes, _ := os.ReadFile("testdata/expectedFinalState_central.json")
				mockCloudStorage.On("WriteObject",
					mock.Anything,
					mock.Anything,
					"tenants/hammer-central/state.json",
					mock.MatchedBy(func(data []byte) bool {
						var expected, actual TenantState
						if err := json.Unmarshal(stateBytes, &expected); err != nil {
							return false
						}
						if err := json.Unmarshal(data, &actual); err != nil {
							return false
						}
						actual.AppliedAt = expected.AppliedAt
						equal := reflect.DeepEqual(expected, actual)
						if !equal {
							t.Logf("expected: %+v", expected)
							t.Logf("actual:   %+v", actual)
						}
						return equal
					}),
					mock.Anything,
				).Return(nil).Once()
				mockCloudStorage.On("WriteObject", mock.Anything, mock.Anything, mock.Anything, mock.Anything, mock.Anything).
					Return(nil)
			},
			wantErr: false,
		},
		{
			name: "Successful Apply central tenant existing previous state",
			tenant: &tenant.Tenant{
				APIVersion: "core.oam.dev/v1beta1",
				Kind:       "Tenant",
				Metadata:   tenant.Metadata{Name: "hammer-central"},
				Spec: tenant.Spec{
					BillingAccount: "ABCDE-12345-FGHIJ",
					ParentFolder:   "937506553540",
					AllowedApis: []string{
						"cloudbuild.googleapis.com",
						"artifactregistry.googleapis.com",
						"storage.googleapis.com",
						"secretmanager.googleapis.com",
						"logging.googleapis.com",
						"monitoring.googleapis.com",
						"pubsub.googleapis.com",
						"cloudtrace.googleapis.com",
					},
					Environments: []string{
						"prod",
					},
					ServiceAccounts: []tenant.ServiceAccountSpec{
						{
							Name:        "sa-provisioner",
							Description: "provisioner",
							Roles: tenant.SARoleBinding{
								Project: []string{"roles/storage.objectAdmin"},
								Organization: []string{
									"roles/resourcemanager.projectCreator",
									"roles/resourcemanager.folderCreator",
									"roles/orgpolicy.policyAdmin",
									"roles/iam.serviceAccountAdmin",
									"roles/serviceusage.serviceUsageAdmin",
									"roles/billing.user",
									"roles/resourcemanager.organizationAdmin",
								},
							},
						},
						{
							Name:        "sa-oam",
							Description: "Runs CI/CD pipelines interpreting OAM files",
							Roles: tenant.SARoleBinding{
								Project: []string{
									"roles/iam.serviceAccountTokenCreator",
									"roles/artifactregistry.writer",
									"roles/cloudbuild.builds.editor",
									"roles/storage.objectAdmin",
									"roles/pubsub.publisher",
									"roles/pubsub.subscriber",
								},
							},
						},
					},
				},
			},
			setupMock: func(mockResourceManager *MockResourceManager, mockServiceUsage *MockServiceUsage, mockOrgPolicy *MockOrgPolicy, mockIam *MockIam, mockBilling *MockBilling, mockCloudStorage *MockCloudStorage) {
				mockCloudStorage.On("EnsureBucketExists", mock.Anything, mock.Anything, mock.Anything, mock.Anything).
					Return(nil)
				lastAppliedState, _ := os.ReadFile("testdata/lastAppliedState_central.json")
				mockCloudStorage.On("GetObject", mock.Anything, mock.Anything, mock.Anything).
					Return(lastAppliedState, nil)
				mockResourceManager.On("EnsureProjectExists", mock.Anything, mock.Anything, mock.Anything, mock.Anything).
					Return("12345", nil)
				mockBilling.On("LinkBillingAccount", mock.Anything, mock.Anything, mock.Anything).
					Return(nil)
				mockOrgPolicy.On("EnforcePolicy", mock.Anything, mock.Anything, mock.Anything).
					Return(nil)
				mockServiceUsage.On("EnableAPIs", mock.Anything, mock.Anything, mock.Anything).
					Return(nil)
				mockIam.On("EnsureServiceAccountExists", mock.Anything, mock.Anything, mock.Anything, mock.Anything).
					Return("sa-pipeline@hammer-central-prod.iam.gserviceaccount.com", nil).Once()
				mockIam.On("EnsureServiceAccountExists", mock.Anything, mock.Anything, mock.Anything, mock.Anything).
					Return("sa-provisioner@hammer-central-prod.iam.gserviceaccount.com", nil).Once()
				mockIam.On("EnsureServiceAccountExists", mock.Anything, mock.Anything, mock.Anything, mock.Anything).
					Return("sa-oam@hammer-central-prod.iam.gserviceaccount.com", nil).Once()
				mockIam.On("BindProjectRoles", mock.Anything, mock.Anything, mock.Anything, mock.Anything, mock.Anything).
					Return(nil)
				mockIam.On("UnbindProjectRoles", mock.Anything, mock.Anything, mock.Anything, mock.Anything, mock.Anything).
					Return(nil)
				mockIam.On("BindOrgRoles", mock.Anything, mock.Anything, mock.Anything, mock.Anything, mock.Anything).
					Return(nil)
				mockIam.On("UnbindOrgRoles", mock.Anything, mock.Anything, mock.Anything, mock.Anything, mock.Anything).
					Return(nil)
				stateBytes, _ := os.ReadFile("testdata/expectedFinalState_central.json")
				mockCloudStorage.On("WriteObject",
					mock.Anything,
					mock.Anything,
					"tenants/hammer-central/state.json",
					mock.MatchedBy(func(data []byte) bool {
						var expected, actual TenantState
						if err := json.Unmarshal(stateBytes, &expected); err != nil {
							return false
						}
						if err := json.Unmarshal(data, &actual); err != nil {
							return false
						}
						actual.AppliedAt = expected.AppliedAt
						equal := reflect.DeepEqual(expected, actual)
						if !equal {
							t.Logf("expected: %+v", expected)
							t.Logf("actual:   %+v", actual)
						}
						return equal
					}),
					mock.Anything,
				).Return(nil).Once()
				mockCloudStorage.On("WriteObject", mock.Anything, mock.Anything, mock.Anything, mock.Anything, mock.Anything).
					Return(nil)
			},
			wantErr: false,
		},
		{
			name: "failed apply due to custom iam account creation failure",
			tenant: &tenant.Tenant{
				APIVersion: "core.oam.dev/v1beta1",
				Kind:       "Tenant",
				Metadata:   tenant.Metadata{Name: "hammer-central"},
				Spec: tenant.Spec{
					BillingAccount: "ABCDE-12345-FGHIJ",
					ParentFolder:   "937506553540",
					AllowedApis: []string{
						"cloudbuild.googleapis.com",
						"artifactregistry.googleapis.com",
						"storage.googleapis.com",
						"secretmanager.googleapis.com",
						"logging.googleapis.com",
						"monitoring.googleapis.com",
						"pubsub.googleapis.com",
						"cloudtrace.googleapis.com",
					},
					Environments: []string{
						"prod",
					},
					ServiceAccounts: []tenant.ServiceAccountSpec{
						{
							Name:        "sa-provisioner",
							Description: "provisioner",
							Roles: tenant.SARoleBinding{
								Project: []string{"roles/storage.objectAdmin"},
								Organization: []string{
									"roles/resourcemanager.projectCreator",
									"roles/resourcemanager.folderCreator",
									"roles/orgpolicy.policyAdmin",
									"roles/iam.serviceAccountAdmin",
									"roles/serviceusage.serviceUsageAdmin",
									"roles/billing.user",
									"roles/resourcemanager.organizationAdmin",
								},
							},
						},
						{
							Name:        "sa-oam",
							Description: "Runs CI/CD pipelines interpreting OAM files",
							Roles: tenant.SARoleBinding{
								Project: []string{
									"roles/iam.serviceAccountTokenCreator",
									"roles/artifactregistry.writer",
									"roles/cloudbuild.builds.editor",
									"roles/storage.objectAdmin",
									"roles/pubsub.publisher",
									"roles/pubsub.subscriber",
								},
							},
						},
					},
				},
			},
			setupMock: func(mockResourceManager *MockResourceManager, mockServiceUsage *MockServiceUsage, mockOrgPolicy *MockOrgPolicy, mockIam *MockIam, mockBilling *MockBilling, mockCloudStorage *MockCloudStorage) {
				mockCloudStorage.On("EnsureBucketExists", mock.Anything, mock.Anything, mock.Anything, mock.Anything).
					Return(nil)
				mockCloudStorage.On("GetObject", mock.Anything, mock.Anything, mock.Anything).
					Return(nil, storage.ErrObjectNotExist)
				mockResourceManager.On("EnsureProjectExists", mock.Anything, mock.Anything, mock.Anything, mock.Anything).
					Return("12345", nil)
				mockBilling.On("LinkBillingAccount", mock.Anything, mock.Anything, mock.Anything).
					Return(nil)
				mockIam.On("EnsureServiceAccountExists", mock.Anything, mock.Anything, mock.Anything, mock.Anything).
					Return("sa-pipeline@hammer-central-prod.iam.gserviceaccount.com", nil).Once()
				mockIam.On("EnsureServiceAccountExists", mock.Anything, mock.Anything, mock.Anything, mock.Anything).
					Return("sa-provisioner@hammer-central-prod.iam.gserviceaccount.com", nil).Once()
				mockIam.On("EnsureServiceAccountExists", mock.Anything, mock.Anything, mock.Anything, mock.Anything).
					Return("", fmt.Errorf("error")).Once()
			},
			wantErr: true,
		},
		{
			name: "failure due to failed project role binding on custom service accounts",
			tenant: &tenant.Tenant{
				APIVersion: "core.oam.dev/v1beta1",
				Kind:       "Tenant",
				Metadata:   tenant.Metadata{Name: "hammer-central"},
				Spec: tenant.Spec{
					BillingAccount: "ABCDE-12345-FGHIJ",
					ParentFolder:   "937506553540",
					AllowedApis: []string{
						"cloudbuild.googleapis.com",
						"artifactregistry.googleapis.com",
						"storage.googleapis.com",
						"secretmanager.googleapis.com",
						"logging.googleapis.com",
						"monitoring.googleapis.com",
						"pubsub.googleapis.com",
						"cloudtrace.googleapis.com",
					},
					Environments: []string{
						"prod",
					},
					ServiceAccounts: []tenant.ServiceAccountSpec{
						{
							Name:        "custom",
							Description: "provisioner",
							Roles: tenant.SARoleBinding{
								Project: []string{"roles/storage.objectAdmin"},
								Organization: []string{
									"roles/resourcemanager.projectCreator",
									"roles/resourcemanager.folderCreator",
									"roles/orgpolicy.policyAdmin",
									"roles/iam.serviceAccountAdmin",
									"roles/serviceusage.serviceUsageAdmin",
									"roles/billing.user",
									"roles/resourcemanager.organizationAdmin",
								},
							},
						},
					},
				},
			},
			setupMock: func(mockResourceManager *MockResourceManager, mockServiceUsage *MockServiceUsage, mockOrgPolicy *MockOrgPolicy, mockIam *MockIam, mockBilling *MockBilling, mockCloudStorage *MockCloudStorage) {
				mockCloudStorage.On("EnsureBucketExists", mock.Anything, mock.Anything, mock.Anything, mock.Anything).
					Return(nil)
				mockCloudStorage.On("GetObject", mock.Anything, mock.Anything, mock.Anything).
					Return(nil, storage.ErrObjectNotExist)
				mockResourceManager.On("EnsureProjectExists", mock.Anything, mock.Anything, mock.Anything, mock.Anything).
					Return("12345", nil)
				mockBilling.On("LinkBillingAccount", mock.Anything, mock.Anything, mock.Anything).
					Return(nil)
				mockIam.On("EnsureServiceAccountExists", mock.Anything, mock.Anything, mock.Anything, mock.Anything).
					Return("sa-pipeline@hammer-central-prod.iam.gserviceaccount.com", nil).Once()
				mockIam.On("EnsureServiceAccountExists", mock.Anything, mock.Anything, mock.Anything, mock.Anything).
					Return("custom@hammer-central-prod.iam.gserviceaccount.com", nil).Once()
				mockIam.On("BindProjectRoles", mock.Anything, mock.Anything, mock.Anything, mock.Anything, mock.Anything).
					Return(nil).Once()
				mockIam.On("BindProjectRoles", mock.Anything, mock.Anything, mock.Anything, mock.Anything, mock.Anything).
					Return(fmt.Errorf("error")).Once()
			},
			wantErr: true,
		},
		{
			name: "failure due to failed org role binding on custom service accounts",
			tenant: &tenant.Tenant{
				APIVersion: "core.oam.dev/v1beta1",
				Kind:       "Tenant",
				Metadata:   tenant.Metadata{Name: "hammer-central"},
				Spec: tenant.Spec{
					BillingAccount: "ABCDE-12345-FGHIJ",
					ParentFolder:   "937506553540",
					AllowedApis: []string{
						"cloudbuild.googleapis.com",
						"artifactregistry.googleapis.com",
						"storage.googleapis.com",
						"secretmanager.googleapis.com",
						"logging.googleapis.com",
						"monitoring.googleapis.com",
						"pubsub.googleapis.com",
						"cloudtrace.googleapis.com",
					},
					Environments: []string{
						"prod",
					},
					ServiceAccounts: []tenant.ServiceAccountSpec{
						{
							Name:        "custom",
							Description: "provisioner",
							Roles: tenant.SARoleBinding{
								Project: []string{"roles/storage.objectAdmin"},
								Organization: []string{
									"roles/resourcemanager.projectCreator",
									"roles/resourcemanager.folderCreator",
									"roles/orgpolicy.policyAdmin",
									"roles/iam.serviceAccountAdmin",
									"roles/serviceusage.serviceUsageAdmin",
									"roles/billing.user",
									"roles/resourcemanager.organizationAdmin",
								},
							},
						},
					},
				},
			},
			setupMock: func(mockResourceManager *MockResourceManager, mockServiceUsage *MockServiceUsage, mockOrgPolicy *MockOrgPolicy, mockIam *MockIam, mockBilling *MockBilling, mockCloudStorage *MockCloudStorage) {
				mockCloudStorage.On("EnsureBucketExists", mock.Anything, mock.Anything, mock.Anything, mock.Anything).
					Return(nil)
				mockCloudStorage.On("GetObject", mock.Anything, mock.Anything, mock.Anything).
					Return(nil, storage.ErrObjectNotExist)
				mockResourceManager.On("EnsureProjectExists", mock.Anything, mock.Anything, mock.Anything, mock.Anything).
					Return("12345", nil)
				mockBilling.On("LinkBillingAccount", mock.Anything, mock.Anything, mock.Anything).
					Return(nil)
				mockIam.On("EnsureServiceAccountExists", mock.Anything, mock.Anything, mock.Anything, mock.Anything).
					Return("sa-pipeline@hammer-central-prod.iam.gserviceaccount.com", nil).Once()
				mockIam.On("EnsureServiceAccountExists", mock.Anything, mock.Anything, mock.Anything, mock.Anything).
					Return("custom@hammer-central-prod.iam.gserviceaccount.com", nil).Once()
				mockIam.On("BindProjectRoles", mock.Anything, mock.Anything, mock.Anything, mock.Anything, mock.Anything).
					Return(nil).Once()
				mockIam.On("BindProjectRoles", mock.Anything, mock.Anything, mock.Anything, mock.Anything, mock.Anything).
					Return(nil).Once()
				mockIam.On("BindOrgRoles", mock.Anything, mock.Anything, mock.Anything, mock.Anything, mock.Anything).
					Return(fmt.Errorf("error")).Once()
			},
			wantErr: true,
		},
		{
			name: "failed Apply unable to unbind existing project role",
			tenant: &tenant.Tenant{
				APIVersion: "core.oam.dev/v1beta1",
				Kind:       "Tenant",
				Metadata:   tenant.Metadata{Name: "hammer-central"},
				Spec: tenant.Spec{
					BillingAccount: "ABCDE-12345-FGHIJ",
					ParentFolder:   "937506553540",
					AllowedApis: []string{
						"cloudbuild.googleapis.com",
						"artifactregistry.googleapis.com",
						"storage.googleapis.com",
						"secretmanager.googleapis.com",
						"logging.googleapis.com",
						"monitoring.googleapis.com",
						"pubsub.googleapis.com",
						"cloudtrace.googleapis.com",
					},
					Environments: []string{
						"prod",
					},
					ServiceAccounts: []tenant.ServiceAccountSpec{
						{
							Name:        "sa-provisioner",
							Description: "provisioner",
							Roles: tenant.SARoleBinding{
								Project: []string{"roles/storage.objectAdmin"},
								Organization: []string{
									"roles/resourcemanager.projectCreator",
									"roles/resourcemanager.folderCreator",
									"roles/orgpolicy.policyAdmin",
									"roles/iam.serviceAccountAdmin",
									"roles/serviceusage.serviceUsageAdmin",
									"roles/billing.user",
									"roles/resourcemanager.organizationAdmin",
								},
							},
						},
						{
							Name:        "sa-oam",
							Description: "Runs CI/CD pipelines interpreting OAM files",
							Roles: tenant.SARoleBinding{
								Project: []string{
									"roles/iam.serviceAccountTokenCreator",
									"roles/artifactregistry.writer",
									"roles/cloudbuild.builds.editor",
									"roles/storage.objectAdmin",
									"roles/pubsub.publisher",
									"roles/pubsub.subscriber",
								},
							},
						},
					},
				},
			},
			setupMock: func(mockResourceManager *MockResourceManager, mockServiceUsage *MockServiceUsage, mockOrgPolicy *MockOrgPolicy, mockIam *MockIam, mockBilling *MockBilling, mockCloudStorage *MockCloudStorage) {
				mockCloudStorage.On("EnsureBucketExists", mock.Anything, mock.Anything, mock.Anything, mock.Anything).
					Return(nil)
				lastAppliedState, _ := os.ReadFile("testdata/lastAppliedState_central.json")
				mockCloudStorage.On("GetObject", mock.Anything, mock.Anything, mock.Anything).
					Return(lastAppliedState, nil)
				mockResourceManager.On("EnsureProjectExists", mock.Anything, mock.Anything, mock.Anything, mock.Anything).
					Return("12345", nil)
				mockBilling.On("LinkBillingAccount", mock.Anything, mock.Anything, mock.Anything).
					Return(nil)
				mockIam.On("EnsureServiceAccountExists", mock.Anything, mock.Anything, mock.Anything, mock.Anything).
					Return("sa-pipeline@hammer-central-prod.iam.gserviceaccount.com", nil).Once()
				mockIam.On("EnsureServiceAccountExists", mock.Anything, mock.Anything, mock.Anything, mock.Anything).
					Return("sa-provisioner@hammer-central-prod.iam.gserviceaccount.com", nil).Once()
				mockIam.On("EnsureServiceAccountExists", mock.Anything, mock.Anything, mock.Anything, mock.Anything).
					Return("sa-oam@hammer-central-prod.iam.gserviceaccount.com", nil).Once()
				mockIam.On("BindProjectRoles", mock.Anything, mock.Anything, mock.Anything, mock.Anything, mock.Anything).
					Return(nil)
				mockIam.On("UnbindProjectRoles", mock.Anything, mock.Anything, mock.Anything, mock.Anything, mock.Anything).
					Return(fmt.Errorf("error"))
			},
			wantErr: true,
		},
		{
			name: "failed Apply unable to unbind existing org role",
			tenant: &tenant.Tenant{
				APIVersion: "core.oam.dev/v1beta1",
				Kind:       "Tenant",
				Metadata:   tenant.Metadata{Name: "hammer-central"},
				Spec: tenant.Spec{
					BillingAccount: "ABCDE-12345-FGHIJ",
					ParentFolder:   "937506553540",
					AllowedApis: []string{
						"cloudbuild.googleapis.com",
						"artifactregistry.googleapis.com",
						"storage.googleapis.com",
						"secretmanager.googleapis.com",
						"logging.googleapis.com",
						"monitoring.googleapis.com",
						"pubsub.googleapis.com",
						"cloudtrace.googleapis.com",
					},
					Environments: []string{
						"prod",
					},
					ServiceAccounts: []tenant.ServiceAccountSpec{
						{
							Name:        "sa-provisioner",
							Description: "provisioner",
							Roles: tenant.SARoleBinding{
								Project: []string{"roles/storage.objectAdmin"},
								Organization: []string{
									"roles/resourcemanager.projectCreator",
									"roles/resourcemanager.folderCreator",
									"roles/orgpolicy.policyAdmin",
									"roles/iam.serviceAccountAdmin",
									"roles/serviceusage.serviceUsageAdmin",
									"roles/billing.user",
									"roles/resourcemanager.organizationAdmin",
								},
							},
						},
						{
							Name:        "sa-oam",
							Description: "Runs CI/CD pipelines interpreting OAM files",
							Roles: tenant.SARoleBinding{
								Project: []string{
									"roles/iam.serviceAccountTokenCreator",
									"roles/artifactregistry.writer",
									"roles/cloudbuild.builds.editor",
									"roles/storage.objectAdmin",
									"roles/pubsub.publisher",
									"roles/pubsub.subscriber",
								},
							},
						},
					},
				},
			},
			setupMock: func(mockResourceManager *MockResourceManager, mockServiceUsage *MockServiceUsage, mockOrgPolicy *MockOrgPolicy, mockIam *MockIam, mockBilling *MockBilling, mockCloudStorage *MockCloudStorage) {
				mockCloudStorage.On("EnsureBucketExists", mock.Anything, mock.Anything, mock.Anything, mock.Anything).
					Return(nil)
				lastAppliedState, _ := os.ReadFile("testdata/lastAppliedState_central.json")
				mockCloudStorage.On("GetObject", mock.Anything, mock.Anything, mock.Anything).
					Return(lastAppliedState, nil)
				mockResourceManager.On("EnsureProjectExists", mock.Anything, mock.Anything, mock.Anything, mock.Anything).
					Return("12345", nil)
				mockBilling.On("LinkBillingAccount", mock.Anything, mock.Anything, mock.Anything).
					Return(nil)
				mockIam.On("EnsureServiceAccountExists", mock.Anything, mock.Anything, mock.Anything, mock.Anything).
					Return("sa-pipeline@hammer-central-prod.iam.gserviceaccount.com", nil).Once()
				mockIam.On("EnsureServiceAccountExists", mock.Anything, mock.Anything, mock.Anything, mock.Anything).
					Return("sa-provisioner@hammer-central-prod.iam.gserviceaccount.com", nil).Once()
				mockIam.On("EnsureServiceAccountExists", mock.Anything, mock.Anything, mock.Anything, mock.Anything).
					Return("sa-oam@hammer-central-prod.iam.gserviceaccount.com", nil).Once()
				mockIam.On("BindProjectRoles", mock.Anything, mock.Anything, mock.Anything, mock.Anything, mock.Anything).
					Return(nil)
				mockIam.On("UnbindProjectRoles", mock.Anything, mock.Anything, mock.Anything, mock.Anything, mock.Anything).
					Return(nil)
				mockIam.On("BindOrgRoles", mock.Anything, mock.Anything, mock.Anything, mock.Anything, mock.Anything).
					Return(nil)
				mockIam.On("UnbindOrgRoles", mock.Anything, mock.Anything, mock.Anything, mock.Anything, mock.Anything).
					Return(fmt.Errorf("error"))
			},
			wantErr: true,
		},
		{
			name: "failed apply unsupported api to remove",
			tenant: &tenant.Tenant{
				APIVersion: "core.oam.dev/v1beta1",
				Kind:       "Tenant",
				Metadata:   tenant.Metadata{Name: "acme-corp"},
				Spec: tenant.Spec{
					BillingAccount: "ABCDE-12345-FGHIJ",
					ParentFolder:   "937506553540",
					AllowedApis: []string{
						"run.googleapis.com",
						"artifactregistry.googleapis.com",
						"logging.googleapis.com",
						"monitoring.googleapis.com",
					},
					Environments: []string{
						"dev",
						"prod",
					},
				},
			},
			setupMock: func(mockResourceManager *MockResourceManager, mockServiceUsage *MockServiceUsage, mockOrgPolicy *MockOrgPolicy, mockIam *MockIam, mockBilling *MockBilling, mockCloudStorage *MockCloudStorage) {
				mockCloudStorage.On("EnsureBucketExists", mock.Anything, mock.Anything, mock.Anything, mock.Anything).
					Return(nil)
				stateBytes, _ := os.ReadFile("testdata/lastAppliedState_unsupportedapi.json")
				mockCloudStorage.On("GetObject", mock.Anything, mock.Anything, mock.Anything).
					Return(stateBytes, nil)
				mockResourceManager.On("EnsureFolderExists", mock.Anything, mock.Anything, mock.Anything).
					Return("acme-corp", nil)
				mockResourceManager.On("EnsureProjectExists", mock.Anything, mock.Anything, mock.Anything, mock.Anything).
					Return("12345", nil)
				mockBilling.On("LinkBillingAccount", mock.Anything, mock.Anything, mock.Anything).
					Return(nil)
				mockIam.On("EnsureServiceAccountExists", mock.Anything, mock.Anything, mock.Anything, mock.Anything).
					Return("sa-pipeline@acme-corp-dev.iam.gserviceaccount.com", nil).Once()
				mockIam.On("EnsureServiceAccountExists", mock.Anything, mock.Anything, mock.Anything, mock.Anything).
					Return("sa-pipeline@acme-corp-prod.iam.gserviceaccount.com", nil).Once()
			},
			wantErr: true,
		},
		{
			name: "failed apply fails to unbind role",
			tenant: &tenant.Tenant{
				APIVersion: "core.oam.dev/v1beta1",
				Kind:       "Tenant",
				Metadata:   tenant.Metadata{Name: "acme-corp"},
				Spec: tenant.Spec{
					BillingAccount: "ABCDE-12345-FGHIJ",
					ParentFolder:   "937506553540",
					AllowedApis: []string{
						"run.googleapis.com",
						"artifactregistry.googleapis.com",
						"logging.googleapis.com",
						"monitoring.googleapis.com",
					},
					Environments: []string{
						"dev",
						"prod",
					},
				},
			},
			setupMock: func(mockResourceManager *MockResourceManager, mockServiceUsage *MockServiceUsage, mockOrgPolicy *MockOrgPolicy, mockIam *MockIam, mockBilling *MockBilling, mockCloudStorage *MockCloudStorage) {
				mockCloudStorage.On("EnsureBucketExists", mock.Anything, mock.Anything, mock.Anything, mock.Anything).
					Return(nil)
				stateBytes, _ := os.ReadFile("testdata/lastAppliedState.json")
				mockCloudStorage.On("GetObject", mock.Anything, mock.Anything, mock.Anything).
					Return(stateBytes, nil)
				mockResourceManager.On("EnsureFolderExists", mock.Anything, mock.Anything, mock.Anything).
					Return("acme-corp", nil)
				mockResourceManager.On("EnsureProjectExists", mock.Anything, mock.Anything, mock.Anything, mock.Anything).
					Return("12345", nil)
				mockBilling.On("LinkBillingAccount", mock.Anything, mock.Anything, mock.Anything).
					Return(nil)
				mockIam.On("EnsureServiceAccountExists", mock.Anything, mock.Anything, mock.Anything, mock.Anything).
					Return("sa-pipeline@acme-corp-dev.iam.gserviceaccount.com", nil).Once()
				mockIam.On("EnsureServiceAccountExists", mock.Anything, mock.Anything, mock.Anything, mock.Anything).
					Return("sa-pipeline@acme-corp-prod.iam.gserviceaccount.com", nil).Once()
				mockIam.On("BindProjectRoles", mock.Anything, mock.Anything, mock.Anything, mock.Anything, mock.Anything).
					Return(nil)
				mockIam.On("UnbindProjectRoles", mock.Anything, mock.Anything, mock.Anything, mock.Anything, mock.Anything).
					Return(fmt.Errorf("error"))
			},
			wantErr: true,
		},
		{
			name: "failed Apply with existing previous state bad unmarshal",
			tenant: &tenant.Tenant{
				APIVersion: "core.oam.dev/v1beta1",
				Kind:       "Tenant",
				Metadata:   tenant.Metadata{Name: "acme-corp"},
				Spec: tenant.Spec{
					BillingAccount: "ABCDE-12345-FGHIJ",
					ParentFolder:   "937506553540",
					AllowedApis: []string{
						"run.googleapis.com",
						"artifactregistry.googleapis.com",
						"logging.googleapis.com",
						"monitoring.googleapis.com",
					},
					Environments: []string{
						"dev",
						"prod",
					},
				},
			},
			setupMock: func(mockResourceManager *MockResourceManager, mockServiceUsage *MockServiceUsage, mockOrgPolicy *MockOrgPolicy, mockIam *MockIam, mockBilling *MockBilling, mockCloudStorage *MockCloudStorage) {
				mockCloudStorage.On("EnsureBucketExists", mock.Anything, mock.Anything, mock.Anything, mock.Anything).
					Return(nil)
				stateBytes, _ := os.ReadFile("testdata/lastAppliedState_bad.json")
				mockCloudStorage.On("GetObject", mock.Anything, mock.Anything, mock.Anything).
					Return(stateBytes, nil)
			},
			wantErr: true,
		},
		{
			name: "Failed Apply Bad Cloud Storage Ensure",
			tenant: &tenant.Tenant{
				APIVersion: "core.oam.dev/v1beta1",
				Kind:       "Tenant",
				Metadata:   tenant.Metadata{Name: "acme-corp"},
				Spec: tenant.Spec{
					BillingAccount: "ABCDE-12345-FGHIJ",
					ParentFolder:   "937506553540",
					AllowedApis: []string{
						"run.googleapis.com",
						"artifactregistry.googleapis.com",
						"logging.googleapis.com",
						"monitoring.googleapis.com",
					},
					Environments: []string{
						"dev",
						"prod",
					},
				},
			},
			setupMock: func(mockResourceManager *MockResourceManager, mockServiceUsage *MockServiceUsage, mockOrgPolicy *MockOrgPolicy, mockIam *MockIam, mockBilling *MockBilling, mockCloudStorage *MockCloudStorage) {
				mockCloudStorage.On("EnsureBucketExists", mock.Anything, mock.Anything, mock.Anything, mock.Anything).
					Return(fmt.Errorf("error"))
			},
			wantErr: true,
		},
		{
			name: "Failed apply can't get cloud storage object",
			tenant: &tenant.Tenant{
				APIVersion: "core.oam.dev/v1beta1",
				Kind:       "Tenant",
				Metadata:   tenant.Metadata{Name: "acme-corp"},
				Spec: tenant.Spec{
					BillingAccount: "ABCDE-12345-FGHIJ",
					ParentFolder:   "937506553540",
					AllowedApis: []string{
						"run.googleapis.com",
						"artifactregistry.googleapis.com",
						"logging.googleapis.com",
						"monitoring.googleapis.com",
					},
					Environments: []string{
						"dev",
						"prod",
					},
				},
			},
			setupMock: func(mockResourceManager *MockResourceManager, mockServiceUsage *MockServiceUsage, mockOrgPolicy *MockOrgPolicy, mockIam *MockIam, mockBilling *MockBilling, mockCloudStorage *MockCloudStorage) {
				mockCloudStorage.On("EnsureBucketExists", mock.Anything, mock.Anything, mock.Anything, mock.Anything).
					Return(nil)
				mockCloudStorage.On("GetObject", mock.Anything, mock.Anything, mock.Anything).
					Return(nil, fmt.Errorf("error"))
			},
			wantErr: true,
		},
		{
			name: "Failed apply folder creation fails",
			tenant: &tenant.Tenant{
				APIVersion: "core.oam.dev/v1beta1",
				Kind:       "Tenant",
				Metadata:   tenant.Metadata{Name: "acme-corp"},
				Spec: tenant.Spec{
					BillingAccount: "ABCDE-12345-FGHIJ",
					ParentFolder:   "937506553540",
					AllowedApis: []string{
						"run.googleapis.com",
						"artifactregistry.googleapis.com",
						"logging.googleapis.com",
						"monitoring.googleapis.com",
					},
					Environments: []string{
						"dev",
						"prod",
					},
				},
			},
			setupMock: func(mockResourceManager *MockResourceManager, mockServiceUsage *MockServiceUsage, mockOrgPolicy *MockOrgPolicy, mockIam *MockIam, mockBilling *MockBilling, mockCloudStorage *MockCloudStorage) {
				mockCloudStorage.On("EnsureBucketExists", mock.Anything, mock.Anything, mock.Anything, mock.Anything).
					Return(nil)
				mockCloudStorage.On("GetObject", mock.Anything, mock.Anything, mock.Anything).
					Return(nil, storage.ErrObjectNotExist)
				mockResourceManager.On("EnsureFolderExists", mock.Anything, mock.Anything, mock.Anything).
					Return(nil, fmt.Errorf("error"))
			},
			wantErr: true,
		},
		{
			name: "failed apply project creation fails",
			tenant: &tenant.Tenant{
				APIVersion: "core.oam.dev/v1beta1",
				Kind:       "Tenant",
				Metadata:   tenant.Metadata{Name: "acme-corp"},
				Spec: tenant.Spec{
					BillingAccount: "ABCDE-12345-FGHIJ",
					ParentFolder:   "937506553540",
					AllowedApis: []string{
						"run.googleapis.com",
						"artifactregistry.googleapis.com",
						"logging.googleapis.com",
						"monitoring.googleapis.com",
					},
					Environments: []string{
						"dev",
						"prod",
					},
				},
			},
			setupMock: func(mockResourceManager *MockResourceManager, mockServiceUsage *MockServiceUsage, mockOrgPolicy *MockOrgPolicy, mockIam *MockIam, mockBilling *MockBilling, mockCloudStorage *MockCloudStorage) {
				mockCloudStorage.On("EnsureBucketExists", mock.Anything, mock.Anything, mock.Anything, mock.Anything).
					Return(nil)
				mockCloudStorage.On("GetObject", mock.Anything, mock.Anything, mock.Anything).
					Return(nil, storage.ErrObjectNotExist)
				mockResourceManager.On("EnsureFolderExists", mock.Anything, mock.Anything, mock.Anything).
					Return("acme-corp", nil)
				mockResourceManager.On("EnsureProjectExists", mock.Anything, mock.Anything, mock.Anything, mock.Anything).
					Return("", fmt.Errorf("error"))
			},
			wantErr: true,
		},
		{
			name: "failed apply billing account linking fails",
			tenant: &tenant.Tenant{
				APIVersion: "core.oam.dev/v1beta1",
				Kind:       "Tenant",
				Metadata:   tenant.Metadata{Name: "acme-corp"},
				Spec: tenant.Spec{
					BillingAccount: "ABCDE-12345-FGHIJ",
					ParentFolder:   "937506553540",
					AllowedApis: []string{
						"run.googleapis.com",
						"artifactregistry.googleapis.com",
						"logging.googleapis.com",
						"monitoring.googleapis.com",
					},
					Environments: []string{
						"dev",
						"prod",
					},
				},
			},
			setupMock: func(mockResourceManager *MockResourceManager, mockServiceUsage *MockServiceUsage, mockOrgPolicy *MockOrgPolicy, mockIam *MockIam, mockBilling *MockBilling, mockCloudStorage *MockCloudStorage) {
				mockCloudStorage.On("EnsureBucketExists", mock.Anything, mock.Anything, mock.Anything, mock.Anything).
					Return(nil)
				mockCloudStorage.On("GetObject", mock.Anything, mock.Anything, mock.Anything).
					Return(nil, storage.ErrObjectNotExist)
				mockResourceManager.On("EnsureFolderExists", mock.Anything, mock.Anything, mock.Anything).
					Return("acme-corp", nil)
				mockResourceManager.On("EnsureProjectExists", mock.Anything, mock.Anything, mock.Anything, mock.Anything).
					Return("12345", nil)
				mockBilling.On("LinkBillingAccount", mock.Anything, mock.Anything, mock.Anything).
					Return(fmt.Errorf("error"))
			},
			wantErr: true,
		},
		{
			name: "failed apply org policy application fails",
			tenant: &tenant.Tenant{
				APIVersion: "core.oam.dev/v1beta1",
				Kind:       "Tenant",
				Metadata:   tenant.Metadata{Name: "acme-corp"},
				Spec: tenant.Spec{
					BillingAccount: "ABCDE-12345-FGHIJ",
					ParentFolder:   "937506553540",
					AllowedApis: []string{
						"run.googleapis.com",
						"artifactregistry.googleapis.com",
						"logging.googleapis.com",
						"monitoring.googleapis.com",
					},
					Environments: []string{
						"dev",
						"prod",
					},
				},
			},
			setupMock: func(mockResourceManager *MockResourceManager, mockServiceUsage *MockServiceUsage, mockOrgPolicy *MockOrgPolicy, mockIam *MockIam, mockBilling *MockBilling, mockCloudStorage *MockCloudStorage) {
				mockCloudStorage.On("EnsureBucketExists", mock.Anything, mock.Anything, mock.Anything, mock.Anything).
					Return(nil)
				mockCloudStorage.On("GetObject", mock.Anything, mock.Anything, mock.Anything).
					Return(nil, storage.ErrObjectNotExist)
				mockResourceManager.On("EnsureFolderExists", mock.Anything, mock.Anything, mock.Anything).
					Return("acme-corp", nil)
				mockResourceManager.On("EnsureProjectExists", mock.Anything, mock.Anything, mock.Anything, mock.Anything).
					Return("12345", nil)
				mockBilling.On("LinkBillingAccount", mock.Anything, mock.Anything, mock.Anything).
					Return(nil)
				mockServiceUsage.On("EnableAPIs", mock.Anything, mock.Anything, mock.Anything).
					Return(nil)
				mockIam.On("EnsureServiceAccountExists", mock.Anything, mock.Anything, mock.Anything, mock.Anything).
					Return("sa-pipeline@acme-corp-dev.iam.gserviceaccount.com", nil).Once()
				mockIam.On("EnsureServiceAccountExists", mock.Anything, mock.Anything, mock.Anything, mock.Anything).
					Return("sa-pipeline@acme-corp-prod.iam.gserviceaccount.com", nil).Once()
				mockIam.On("BindProjectRoles", mock.Anything, mock.Anything, mock.Anything, mock.Anything, mock.Anything).
					Return(nil)
				mockOrgPolicy.On("EnforcePolicy", mock.Anything, mock.Anything, mock.Anything).
					Return(fmt.Errorf("error"))
			},
			wantErr: true,
		},
		{
			name: "Failed Apply unable to enable APIs",
			tenant: &tenant.Tenant{
				APIVersion: "core.oam.dev/v1beta1",
				Kind:       "Tenant",
				Metadata:   tenant.Metadata{Name: "acme-corp"},
				Spec: tenant.Spec{
					BillingAccount: "ABCDE-12345-FGHIJ",
					ParentFolder:   "937506553540",
					AllowedApis: []string{
						"run.googleapis.com",
						"artifactregistry.googleapis.com",
						"logging.googleapis.com",
						"monitoring.googleapis.com",
					},
					Environments: []string{
						"dev",
						"prod",
					},
				},
			},
			setupMock: func(mockResourceManager *MockResourceManager, mockServiceUsage *MockServiceUsage, mockOrgPolicy *MockOrgPolicy, mockIam *MockIam, mockBilling *MockBilling, mockCloudStorage *MockCloudStorage) {
				mockCloudStorage.On("EnsureBucketExists", mock.Anything, mock.Anything, mock.Anything, mock.Anything).
					Return(nil)
				mockCloudStorage.On("GetObject", mock.Anything, mock.Anything, mock.Anything).
					Return(nil, storage.ErrObjectNotExist)
				mockResourceManager.On("EnsureFolderExists", mock.Anything, mock.Anything, mock.Anything).
					Return("acme-corp", nil)
				mockResourceManager.On("EnsureProjectExists", mock.Anything, mock.Anything, mock.Anything, mock.Anything).
					Return("12345", nil)
				mockBilling.On("LinkBillingAccount", mock.Anything, mock.Anything, mock.Anything).
					Return(nil)
				mockIam.On("EnsureServiceAccountExists", mock.Anything, mock.Anything, mock.Anything, mock.Anything).
					Return("sa-pipeline@acme-corp-dev.iam.gserviceaccount.com", nil).Once()
				mockIam.On("EnsureServiceAccountExists", mock.Anything, mock.Anything, mock.Anything, mock.Anything).
					Return("sa-pipeline@acme-corp-prod.iam.gserviceaccount.com", nil).Once()
				mockIam.On("BindProjectRoles", mock.Anything, mock.Anything, mock.Anything, mock.Anything, mock.Anything).
					Return(nil)
				mockServiceUsage.On("EnableAPIs", mock.Anything, mock.Anything, mock.Anything).
					Return(fmt.Errorf("error"))
			},
			wantErr: true,
		},
		{
			name: "failed apply unable to create service accounts",
			tenant: &tenant.Tenant{
				APIVersion: "core.oam.dev/v1beta1",
				Kind:       "Tenant",
				Metadata:   tenant.Metadata{Name: "acme-corp"},
				Spec: tenant.Spec{
					BillingAccount: "ABCDE-12345-FGHIJ",
					ParentFolder:   "937506553540",
					AllowedApis: []string{
						"run.googleapis.com",
						"artifactregistry.googleapis.com",
						"logging.googleapis.com",
						"monitoring.googleapis.com",
					},
					Environments: []string{
						"dev",
						"prod",
					},
				},
			},
			setupMock: func(mockResourceManager *MockResourceManager, mockServiceUsage *MockServiceUsage, mockOrgPolicy *MockOrgPolicy, mockIam *MockIam, mockBilling *MockBilling, mockCloudStorage *MockCloudStorage) {
				mockCloudStorage.On("EnsureBucketExists", mock.Anything, mock.Anything, mock.Anything, mock.Anything).
					Return(nil)
				mockCloudStorage.On("GetObject", mock.Anything, mock.Anything, mock.Anything).
					Return(nil, storage.ErrObjectNotExist)
				mockResourceManager.On("EnsureFolderExists", mock.Anything, mock.Anything, mock.Anything).
					Return("acme-corp", nil)
				mockResourceManager.On("EnsureProjectExists", mock.Anything, mock.Anything, mock.Anything, mock.Anything).
					Return("12345", nil)
				mockBilling.On("LinkBillingAccount", mock.Anything, mock.Anything, mock.Anything).
					Return(nil)
				mockIam.On("EnsureServiceAccountExists", mock.Anything, mock.Anything, mock.Anything, mock.Anything).
					Return("", fmt.Errorf("error")).Once()
			},
			wantErr: true,
		},
		{
			name: "failed apply unable to create service accounts due to org roles on non-central tenant",
			tenant: &tenant.Tenant{
				APIVersion: "core.oam.dev/v1beta1",
				Kind:       "Tenant",
				Metadata:   tenant.Metadata{Name: "acme-corp"},
				Spec: tenant.Spec{
					BillingAccount: "ABCDE-12345-FGHIJ",
					ParentFolder:   "937506553540",
					AllowedApis: []string{
						"run.googleapis.com",
						"artifactregistry.googleapis.com",
						"logging.googleapis.com",
						"monitoring.googleapis.com",
					},
					Environments: []string{
						"dev",
						"prod",
					},
					ServiceAccounts: []tenant.ServiceAccountSpec{
						{
							Name: "sa-bad",
							Roles: tenant.SARoleBinding{
								Organization: []string{
									"bad-role",
								},
							},
						},
					},
				},
			},
			setupMock: func(mockResourceManager *MockResourceManager, mockServiceUsage *MockServiceUsage, mockOrgPolicy *MockOrgPolicy, mockIam *MockIam, mockBilling *MockBilling, mockCloudStorage *MockCloudStorage) {
				mockCloudStorage.On("EnsureBucketExists", mock.Anything, mock.Anything, mock.Anything, mock.Anything).
					Return(nil)
				mockCloudStorage.On("GetObject", mock.Anything, mock.Anything, mock.Anything).
					Return(nil, storage.ErrObjectNotExist)
				mockResourceManager.On("EnsureFolderExists", mock.Anything, mock.Anything, mock.Anything).
					Return("acme-corp", nil)
				mockResourceManager.On("EnsureProjectExists", mock.Anything, mock.Anything, mock.Anything, mock.Anything).
					Return("12345", nil)
				mockBilling.On("LinkBillingAccount", mock.Anything, mock.Anything, mock.Anything).
					Return(nil)
			},
			wantErr: true,
		},
		{
			name: "failed apply unable to bind roles",
			tenant: &tenant.Tenant{
				APIVersion: "core.oam.dev/v1beta1",
				Kind:       "Tenant",
				Metadata:   tenant.Metadata{Name: "acme-corp"},
				Spec: tenant.Spec{
					BillingAccount: "ABCDE-12345-FGHIJ",
					ParentFolder:   "937506553540",
					AllowedApis: []string{
						"run.googleapis.com",
						"artifactregistry.googleapis.com",
						"logging.googleapis.com",
						"monitoring.googleapis.com",
					},
					Environments: []string{
						"dev",
						"prod",
					},
				},
			},
			setupMock: func(mockResourceManager *MockResourceManager, mockServiceUsage *MockServiceUsage, mockOrgPolicy *MockOrgPolicy, mockIam *MockIam, mockBilling *MockBilling, mockCloudStorage *MockCloudStorage) {
				mockCloudStorage.On("EnsureBucketExists", mock.Anything, mock.Anything, mock.Anything, mock.Anything).
					Return(nil)
				mockCloudStorage.On("GetObject", mock.Anything, mock.Anything, mock.Anything).
					Return(nil, storage.ErrObjectNotExist)
				mockResourceManager.On("EnsureFolderExists", mock.Anything, mock.Anything, mock.Anything).
					Return("acme-corp", nil)
				mockResourceManager.On("EnsureProjectExists", mock.Anything, mock.Anything, mock.Anything, mock.Anything).
					Return("12345", nil)
				mockBilling.On("LinkBillingAccount", mock.Anything, mock.Anything, mock.Anything).
					Return(nil)
				mockIam.On("EnsureServiceAccountExists", mock.Anything, mock.Anything, mock.Anything, mock.Anything).
					Return("sa-pipeline@acme-corp-dev.iam.gserviceaccount.com", nil).Once()
				mockIam.On("EnsureServiceAccountExists", mock.Anything, mock.Anything, mock.Anything, mock.Anything).
					Return("sa-pipeline@acme-corp-prod.iam.gserviceaccount.com", nil).Once()
				mockIam.On("BindProjectRoles", mock.Anything, mock.Anything, mock.Anything, mock.Anything, mock.Anything).
					Return(fmt.Errorf("error"))
			},
			wantErr: true,
		},
		{
			name: "failed apply unable to write to cloud storage",
			tenant: &tenant.Tenant{
				APIVersion: "core.oam.dev/v1beta1",
				Kind:       "Tenant",
				Metadata:   tenant.Metadata{Name: "acme-corp"},
				Spec: tenant.Spec{
					BillingAccount: "ABCDE-12345-FGHIJ",
					ParentFolder:   "937506553540",
					AllowedApis: []string{
						"run.googleapis.com",
						"artifactregistry.googleapis.com",
						"logging.googleapis.com",
						"monitoring.googleapis.com",
					},
					Environments: []string{
						"dev",
						"prod",
					},
				},
			},
			setupMock: func(mockResourceManager *MockResourceManager, mockServiceUsage *MockServiceUsage, mockOrgPolicy *MockOrgPolicy, mockIam *MockIam, mockBilling *MockBilling, mockCloudStorage *MockCloudStorage) {
				mockCloudStorage.On("EnsureBucketExists", mock.Anything, mock.Anything, mock.Anything, mock.Anything).
					Return(nil)
				mockCloudStorage.On("GetObject", mock.Anything, mock.Anything, mock.Anything).
					Return(nil, storage.ErrObjectNotExist)
				mockResourceManager.On("EnsureFolderExists", mock.Anything, mock.Anything, mock.Anything).
					Return("acme-corp", nil)
				mockResourceManager.On("EnsureProjectExists", mock.Anything, mock.Anything, mock.Anything, mock.Anything).
					Return("12345", nil)
				mockBilling.On("LinkBillingAccount", mock.Anything, mock.Anything, mock.Anything).
					Return(nil)
				mockOrgPolicy.On("EnforcePolicy", mock.Anything, mock.Anything, mock.Anything).
					Return(nil)
				mockServiceUsage.On("EnableAPIs", mock.Anything, mock.Anything, mock.Anything).
					Return(nil)
				mockIam.On("EnsureServiceAccountExists", mock.Anything, mock.Anything, mock.Anything, mock.Anything).
					Return("sa-pipeline@acme-corp-dev.iam.gserviceaccount.com", nil).Once()
				mockIam.On("EnsureServiceAccountExists", mock.Anything, mock.Anything, mock.Anything, mock.Anything).
					Return("sa-pipeline@acme-corp-prod.iam.gserviceaccount.com", nil).Once()
				mockIam.On("BindProjectRoles", mock.Anything, mock.Anything, mock.Anything, mock.Anything, mock.Anything).
					Return(nil)
				mockCloudStorage.On("WriteObject", mock.Anything, mock.Anything, mock.Anything, mock.Anything, mock.Anything).
					Return(fmt.Errorf("error"))
			},
			wantErr: true,
		},
		{
			name: "failed to write to bucket second time",
			tenant: &tenant.Tenant{
				APIVersion: "core.oam.dev/v1beta1",
				Kind:       "Tenant",
				Metadata:   tenant.Metadata{Name: "acme-corp"},
				Spec: tenant.Spec{
					BillingAccount: "ABCDE-12345-FGHIJ",
					ParentFolder:   "937506553540",
					AllowedApis: []string{
						"run.googleapis.com",
						"artifactregistry.googleapis.com",
						"logging.googleapis.com",
						"monitoring.googleapis.com",
					},
					Environments: []string{
						"dev",
						"prod",
					},
				},
			},
			setupMock: func(mockResourceManager *MockResourceManager, mockServiceUsage *MockServiceUsage, mockOrgPolicy *MockOrgPolicy, mockIam *MockIam, mockBilling *MockBilling, mockCloudStorage *MockCloudStorage) {
				mockCloudStorage.On("EnsureBucketExists", mock.Anything, mock.Anything, mock.Anything, mock.Anything).
					Return(nil)
				mockCloudStorage.On("GetObject", mock.Anything, mock.Anything, mock.Anything).
					Return(nil, storage.ErrObjectNotExist)
				mockResourceManager.On("EnsureFolderExists", mock.Anything, mock.Anything, mock.Anything).
					Return("acme-corp", nil)
				mockResourceManager.On("EnsureProjectExists", mock.Anything, mock.Anything, mock.Anything, mock.Anything).
					Return("12345", nil)
				mockBilling.On("LinkBillingAccount", mock.Anything, mock.Anything, mock.Anything).
					Return(nil)
				mockOrgPolicy.On("EnforcePolicy", mock.Anything, mock.Anything, mock.Anything).
					Return(nil)
				mockServiceUsage.On("EnableAPIs", mock.Anything, mock.Anything, mock.Anything).
					Return(nil)
				mockIam.On("EnsureServiceAccountExists", mock.Anything, mock.Anything, mock.Anything, mock.Anything).
					Return("sa-pipeline@acme-corp-dev.iam.gserviceaccount.com", nil).Once()
				mockIam.On("EnsureServiceAccountExists", mock.Anything, mock.Anything, mock.Anything, mock.Anything).
					Return("sa-pipeline@acme-corp-prod.iam.gserviceaccount.com", nil).Once()
				mockIam.On("BindProjectRoles", mock.Anything, mock.Anything, mock.Anything, mock.Anything, mock.Anything).
					Return(nil)
				mockCloudStorage.On("WriteObject", mock.Anything, mock.Anything, mock.Anything, mock.Anything, mock.Anything).
					Return(nil).Once()
				mockCloudStorage.On("WriteObject", mock.Anything, mock.Anything, mock.Anything, mock.Anything, mock.Anything).
					Return(fmt.Errorf("error")).Once()
			},
			wantErr: true,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mockCloudStorage := new(MockCloudStorage)
			mockResourceManager := new(MockResourceManager)
			mockBilling := new(MockBilling)
			mockOrgPolicy := new(MockOrgPolicy)
			mockServiceUsage := new(MockServiceUsage)
			mockIam := new(MockIam)
			tt.setupMock(mockResourceManager, mockServiceUsage, mockOrgPolicy, mockIam, mockBilling, mockCloudStorage)

			provision, _ := New(tt.tenant, &provisioner.DependencyClients{
				CloudStorage:    mockCloudStorage,
				ResourceManager: mockResourceManager,
				Billing:         mockBilling,
				OrgPolicy:       mockOrgPolicy,
				ServiceUsage:    mockServiceUsage,
				IAM:             mockIam,
			})

			err := provision.Apply(context.Background())

			if tt.wantErr {
				require.Error(t, err)
			} else {
				require.NoError(t, err)
			}

			mockCloudStorage.AssertExpectations(t)
			mockResourceManager.AssertExpectations(t)
			mockBilling.AssertExpectations(t)
			mockOrgPolicy.AssertExpectations(t)
			mockServiceUsage.AssertExpectations(t)
			mockIam.AssertExpectations(t)
		})
	}
}
