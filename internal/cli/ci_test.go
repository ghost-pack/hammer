package cli

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"reflect"
	"testing"

	"github.com/ghost-pack/hammer/internal/ci"
	"github.com/ghost-pack/hammer/internal/oam"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
)

func init() {
	ci.Register("noop", New)
}

// The noop component is only used for testing.
func New(component oam.Component, app oam.App, client ci.DependencyClients) (ci.Pipeline, error) {
	if component.Type != "noop" {
		return nil, fmt.Errorf("noop component must be of type noop")
	}

	return &Pipeline{
		component: &component,
	}, nil
}

type Pipeline struct {
	component *oam.Component
}

func (p *Pipeline) ComponentType() string {
	return "noop"
}

func (p *Pipeline) CI(ctx context.Context) (*ci.Artifact, error) {
	return &ci.Artifact{
			Type:       ci.ArtifactTypeCloudBuild,
			Properties: map[string]string{"cloudBuildYaml": fmt.Sprintf("gs://hammer-release/acme-corp/deployments/cloudbuild/%s.yaml", os.Getenv("COMMIT_SHA"))}},
		nil
}

func (p *Pipeline) Analyze(ctx context.Context) error {
	return nil
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

func (m *MockCloudStorage) ListPrefixes(ctx context.Context, bucket, prefix, delimiter string) ([]string, error) {
	callArgs := m.Called(ctx, bucket, prefix, delimiter)
	res, _ := callArgs.Get(0).([]string)
	return res, callArgs.Error(1)
}

func (m *MockCloudStorage) BucketExists(ctx context.Context, bucketName string) (bool, error) {
	callArgs := m.Called(ctx, bucketName)
	res, _ := callArgs.Get(0).(bool)
	return res, callArgs.Error(1)
}

func (m *MockCloudStorage) Close() error {
	callArgs := m.Called()
	return callArgs.Error(0)
}

type MockPubSubClient struct {
	mock.Mock
}

func (m *MockPubSubClient) EnsureTopic(ctx context.Context, projectID, topicID, publisherSA string) error {
	callArgs := m.Called(ctx, projectID, topicID, publisherSA)
	return callArgs.Error(0)
}

func (m *MockPubSubClient) DeleteTopic(ctx context.Context, projectID, topicID string) error {
	callArgs := m.Called(ctx, projectID, topicID)
	return callArgs.Error(0)
}

func (m *MockPubSubClient) PublishMessage(ctx context.Context, projectID, topicID string, data []byte, attributes map[string]string) (string, error) {
	callArgs := m.Called(ctx, projectID, topicID, data, attributes)
	res, _ := callArgs.Get(0).(string)
	return res, callArgs.Error(1)
}

func (m *MockPubSubClient) Close() error {
	callArgs := m.Called()
	return callArgs.Error(0)
}

func TestCIExecute(t *testing.T) {
	tests := []struct {
		name       string
		oamFile    string
		setupMocks func(cloudStorageMock *MockCloudStorage, pubSubMock *MockPubSubClient)
		wantErr    bool
	}{
		{
			name:    "successful CI execution",
			oamFile: "testdata/oam_test_success.yaml",
			setupMocks: func(cloudStorageMock *MockCloudStorage, pubSubMock *MockPubSubClient) {
				cloudStorageMock.On("ListPrefixes", mock.Anything, mock.Anything, mock.Anything, mock.Anything).
					Return([]string{"tenants/hammer-central/", "tenants/acme-corp/"}, nil)
				cloudStorageMock.On("EnsureBucketExists", mock.Anything, mock.Anything, mock.Anything, mock.Anything).
					Return(nil)
				cloudStorageMock.On("WriteObject", mock.Anything, mock.Anything, mock.Anything, mock.Anything, mock.Anything).
					Return(nil)
				tenantBytes, _ := os.ReadFile("testdata/tenantGood/tenant_test_regular.yaml")
				cloudStorageMock.On("GetObject", mock.Anything, mock.Anything, mock.Anything).
					Return(tenantBytes, nil)
				expectedBytes, _ := os.ReadFile("testdata/expected/expected_ci_result.json")
				pubSubMock.On(
					"PublishMessage",
					mock.Anything,
					mock.Anything,
					mock.Anything,
					mock.MatchedBy(func(data []byte) bool {
						var expected, actual ci.CIPubSubMessage
						if err := json.Unmarshal(expectedBytes, &expected); err != nil {
							return false
						}
						if err := json.Unmarshal(data, &actual); err != nil {
							return false
						}
						actual.PublishedAt = expected.PublishedAt
						t.Logf("expected: %+v", expected)
						t.Logf("actual:   %+v", actual)
						return reflect.DeepEqual(expected, actual)
					}),
					mock.Anything,
				).Return("messageId", nil)
			},
			wantErr: false,
		},
		{
			name:    "successful CI execution with per-component pubsub",
			oamFile: "testdata/oam_test_success_per_component.yaml",
			setupMocks: func(cloudStorageMock *MockCloudStorage, pubSubMock *MockPubSubClient) {
				cloudStorageMock.On("ListPrefixes", mock.Anything, mock.Anything, mock.Anything, mock.Anything).
					Return([]string{"tenants/hammer-central/", "tenants/acme-corp/"}, nil)
				cloudStorageMock.On("EnsureBucketExists", mock.Anything, mock.Anything, mock.Anything, mock.Anything).
					Return(nil)
				cloudStorageMock.On("WriteObject", mock.Anything, mock.Anything, mock.Anything, mock.Anything, mock.Anything).
					Return(nil)
				tenantBytes, _ := os.ReadFile("testdata/tenantGood/tenant_test_regular.yaml")
				cloudStorageMock.On("GetObject", mock.Anything, mock.Anything, mock.Anything).
					Return(tenantBytes, nil)
				pubSubMock.On(
					"PublishMessage",
					mock.Anything,
					mock.Anything,
					mock.Anything,
					mock.MatchedBy(func(data []byte) bool {
						expectedBytes, _ := os.ReadFile("testdata/expected/expected_ci_result_reconcile_only.json")

						var expected, actual ci.CIPubSubMessage
						if err := json.Unmarshal(expectedBytes, &expected); err != nil {
							return false
						}
						if err := json.Unmarshal(data, &actual); err != nil {
							return false
						}
						actual.PublishedAt = expected.PublishedAt
						t.Logf("expected: %+v", expected)
						t.Logf("actual:   %+v", actual)
						return reflect.DeepEqual(expected, actual)
					}),
					mock.Anything,
				).Return("messageId", nil).Once()
				pubSubMock.On(
					"PublishMessage",
					mock.Anything,
					mock.Anything,
					mock.Anything,
					mock.MatchedBy(func(data []byte) bool {
						expectedBytes, _ := os.ReadFile("testdata/expected/expected_ci_result_per_component.json")

						var expected, actual ci.CIPubSubMessage
						if err := json.Unmarshal(expectedBytes, &expected); err != nil {
							return false
						}
						if err := json.Unmarshal(data, &actual); err != nil {
							return false
						}
						actual.PublishedAt = expected.PublishedAt
						t.Logf("expected: %+v", expected)
						t.Logf("actual:   %+v", actual)
						return reflect.DeepEqual(expected, actual)
					}),
					mock.Anything,
				).Return("messageId", nil).Once()
			},
			wantErr: false,
		},
		{
			name:    "failed CI execution with per-component due to pubsub failure per component",
			oamFile: "testdata/oam_test_success_per_component.yaml",
			setupMocks: func(cloudStorageMock *MockCloudStorage, pubSubMock *MockPubSubClient) {
				cloudStorageMock.On("ListPrefixes", mock.Anything, mock.Anything, mock.Anything, mock.Anything).
					Return([]string{"tenants/hammer-central/", "tenants/acme-corp/"}, nil)
				cloudStorageMock.On("EnsureBucketExists", mock.Anything, mock.Anything, mock.Anything, mock.Anything).
					Return(nil)
				cloudStorageMock.On("WriteObject", mock.Anything, mock.Anything, mock.Anything, mock.Anything, mock.Anything).
					Return(nil)
				tenantBytes, _ := os.ReadFile("testdata/tenantGood/tenant_test_regular.yaml")
				cloudStorageMock.On("GetObject", mock.Anything, mock.Anything, mock.Anything).
					Return(tenantBytes, nil)
				pubSubMock.On(
					"PublishMessage",
					mock.Anything,
					mock.Anything,
					mock.Anything,
					mock.MatchedBy(func(data []byte) bool {
						expectedBytes, _ := os.ReadFile("testdata/expected/expected_ci_result_reconcile_only.json")

						var expected, actual ci.CIPubSubMessage
						if err := json.Unmarshal(expectedBytes, &expected); err != nil {
							return false
						}
						if err := json.Unmarshal(data, &actual); err != nil {
							return false
						}
						actual.PublishedAt = expected.PublishedAt
						t.Logf("expected: %+v", expected)
						t.Logf("actual:   %+v", actual)
						return reflect.DeepEqual(expected, actual)
					}),
					mock.Anything,
				).Return("messageId", nil).Once()
				pubSubMock.On(
					"PublishMessage",
					mock.Anything,
					mock.Anything,
					mock.Anything,
					mock.MatchedBy(func(data []byte) bool {
						expectedBytes, _ := os.ReadFile("testdata/expected/expected_ci_result_per_component.json")

						var expected, actual ci.CIPubSubMessage
						if err := json.Unmarshal(expectedBytes, &expected); err != nil {
							return false
						}
						if err := json.Unmarshal(data, &actual); err != nil {
							return false
						}
						actual.PublishedAt = expected.PublishedAt
						t.Logf("expected: %+v", expected)
						t.Logf("actual:   %+v", actual)
						return reflect.DeepEqual(expected, actual)
					}),
					mock.Anything,
				).Return(nil, fmt.Errorf("some error")).Once()
			},
			wantErr: true,
		},
		{
			name:    "failed CI execution with per-component due to pubsub failure on reconcile",
			oamFile: "testdata/oam_test_success_per_component.yaml",
			setupMocks: func(cloudStorageMock *MockCloudStorage, pubSubMock *MockPubSubClient) {
				cloudStorageMock.On("ListPrefixes", mock.Anything, mock.Anything, mock.Anything, mock.Anything).
					Return([]string{"tenants/hammer-central/", "tenants/acme-corp/"}, nil)
				cloudStorageMock.On("EnsureBucketExists", mock.Anything, mock.Anything, mock.Anything, mock.Anything).
					Return(nil)
				cloudStorageMock.On("WriteObject", mock.Anything, mock.Anything, mock.Anything, mock.Anything, mock.Anything).
					Return(nil)
				tenantBytes, _ := os.ReadFile("testdata/tenantGood/tenant_test_regular.yaml")
				cloudStorageMock.On("GetObject", mock.Anything, mock.Anything, mock.Anything).
					Return(tenantBytes, nil)
				pubSubMock.On(
					"PublishMessage",
					mock.Anything,
					mock.Anything,
					mock.Anything,
					mock.MatchedBy(func(data []byte) bool {
						expectedBytes, _ := os.ReadFile("testdata/expected/expected_ci_result_reconcile_only.json")

						var expected, actual ci.CIPubSubMessage
						if err := json.Unmarshal(expectedBytes, &expected); err != nil {
							return false
						}
						if err := json.Unmarshal(data, &actual); err != nil {
							return false
						}
						actual.PublishedAt = expected.PublishedAt
						t.Logf("expected: %+v", expected)
						t.Logf("actual:   %+v", actual)
						return reflect.DeepEqual(expected, actual)
					}),
					mock.Anything,
				).Return("", fmt.Errorf("some error")).Once()
			},
			wantErr: true,
		},
		{
			name:    "failed CI execution with per-component fail to get envs",
			oamFile: "testdata/oam_test_success_per_component.yaml",
			setupMocks: func(cloudStorageMock *MockCloudStorage, pubSubMock *MockPubSubClient) {
				cloudStorageMock.On("ListPrefixes", mock.Anything, mock.Anything, mock.Anything, mock.Anything).
					Return([]string{"tenants/hammer-central/", "tenants/acme-corp/"}, nil)
				cloudStorageMock.On("EnsureBucketExists", mock.Anything, mock.Anything, mock.Anything, mock.Anything).
					Return(nil)
				cloudStorageMock.On("WriteObject", mock.Anything, mock.Anything, mock.Anything, mock.Anything, mock.Anything).
					Return(nil)
				cloudStorageMock.On("GetObject", mock.Anything, mock.Anything, mock.Anything).
					Return(nil, fmt.Errorf("some error")).Once()
			},
			wantErr: true,
		},
		{
			name:    "failed CI execution fail to write OAM file",
			oamFile: "testdata/oam_test_success_per_component.yaml",
			setupMocks: func(cloudStorageMock *MockCloudStorage, pubSubMock *MockPubSubClient) {
				cloudStorageMock.On("ListPrefixes", mock.Anything, mock.Anything, mock.Anything, mock.Anything).
					Return([]string{"tenants/hammer-central/", "tenants/acme-corp/"}, nil)
				cloudStorageMock.On("EnsureBucketExists", mock.Anything, mock.Anything, mock.Anything, mock.Anything).
					Return(nil)
				cloudStorageMock.On("WriteObject", mock.Anything, mock.Anything, mock.Anything, mock.Anything, mock.Anything).
					Return(fmt.Errorf("some error")).Once()
			},
			wantErr: true,
		},
		{
			name:    "successful CI execution no artifacts",
			oamFile: "testdata/oam_test_success_no_components.yaml",
			setupMocks: func(cloudStorageMock *MockCloudStorage, pubSubMock *MockPubSubClient) {
				cloudStorageMock.On("ListPrefixes", mock.Anything, mock.Anything, mock.Anything, mock.Anything).
					Return([]string{"tenants/hammer-central/", "tenants/acme-corp/"}, nil)
				cloudStorageMock.On("EnsureBucketExists", mock.Anything, mock.Anything, mock.Anything, mock.Anything).
					Return(nil)
			},
			wantErr: false,
		},
		{
			name:    "failed unparseable oam file",
			oamFile: "testdata/oam_unparseable.yaml",
			setupMocks: func(cloudStorageMock *MockCloudStorage, pubSubMock *MockPubSubClient) {
			},
			wantErr: true,
		},
		{
			name:    "failed ensure bucket fails",
			oamFile: "testdata/oam_test_success.yaml",
			setupMocks: func(cloudStorageMock *MockCloudStorage, pubSubMock *MockPubSubClient) {
				cloudStorageMock.On("ListPrefixes", mock.Anything, mock.Anything, mock.Anything, mock.Anything).
					Return([]string{"tenants/hammer-central/", "tenants/acme-corp/"}, nil)
				cloudStorageMock.On("EnsureBucketExists", mock.Anything, mock.Anything, mock.Anything, mock.Anything).
					Return(fmt.Errorf("some error"))
			},
			wantErr: true,
		},
		{
			name:    "failed bad component",
			oamFile: "testdata/oam_test_fail_bad_component.yaml",
			setupMocks: func(cloudStorageMock *MockCloudStorage, pubSubMock *MockPubSubClient) {
				cloudStorageMock.On("ListPrefixes", mock.Anything, mock.Anything, mock.Anything, mock.Anything).
					Return([]string{"tenants/hammer-central/", "tenants/test/"}, nil)
				cloudStorageMock.On("EnsureBucketExists", mock.Anything, mock.Anything, mock.Anything, mock.Anything).
					Return(nil)
			},
			wantErr: true,
		},
		{
			name:    "failed ci failure",
			oamFile: "testdata/oam_test_fail_ci_failure.yaml",
			setupMocks: func(cloudStorageMock *MockCloudStorage, pubSubMock *MockPubSubClient) {
				cloudStorageMock.On("ListPrefixes", mock.Anything, mock.Anything, mock.Anything, mock.Anything).
					Return([]string{"tenants/hammer-central/", "tenants/test/"}, nil)
				cloudStorageMock.On("EnsureBucketExists", mock.Anything, mock.Anything, mock.Anything, mock.Anything).
					Return(nil)
			},
			wantErr: true,
		},
		{
			name:    "failed ci pubsub error",
			oamFile: "testdata/oam_test_success.yaml",
			setupMocks: func(cloudStorageMock *MockCloudStorage, pubSubMock *MockPubSubClient) {
				cloudStorageMock.On("ListPrefixes", mock.Anything, mock.Anything, mock.Anything, mock.Anything).
					Return([]string{"tenants/hammer-central/", "tenants/acme-corp/"}, nil)
				cloudStorageMock.On("EnsureBucketExists", mock.Anything, mock.Anything, mock.Anything, mock.Anything).
					Return(nil)
				cloudStorageMock.On("WriteObject", mock.Anything, mock.Anything, mock.Anything, mock.Anything, mock.Anything).
					Return(nil)
				tenantBytes, _ := os.ReadFile("testdata/tenantGood/tenant_test_regular.yaml")
				cloudStorageMock.On("GetObject", mock.Anything, mock.Anything, mock.Anything).
					Return(tenantBytes, nil)
				pubSubMock.On(
					"PublishMessage",
					mock.Anything,
					mock.Anything,
					mock.Anything,
					mock.Anything,
					mock.Anything,
				).Return("", fmt.Errorf("some error"))
			},
			wantErr: true,
		},
		{
			name:    "failed ci while getting tenant state",
			oamFile: "testdata/oam_test_success.yaml",
			setupMocks: func(cloudStorageMock *MockCloudStorage, pubSubMock *MockPubSubClient) {
				cloudStorageMock.On("ListPrefixes", mock.Anything, mock.Anything, mock.Anything, mock.Anything).
					Return([]string{"tenants/hammer-central/", "tenants/acme-corp/"}, nil)
				cloudStorageMock.On("EnsureBucketExists", mock.Anything, mock.Anything, mock.Anything, mock.Anything).
					Return(nil)
				cloudStorageMock.On("WriteObject", mock.Anything, mock.Anything, mock.Anything, mock.Anything, mock.Anything).
					Return(nil)
				cloudStorageMock.On("GetObject", mock.Anything, mock.Anything, mock.Anything).
					Return(nil, fmt.Errorf("some error"))
			},
			wantErr: true,
		},
		{
			name:    "failed ci while u0ploading OAM file",
			oamFile: "testdata/oam_test_success.yaml",
			setupMocks: func(cloudStorageMock *MockCloudStorage, pubSubMock *MockPubSubClient) {
				cloudStorageMock.On("ListPrefixes", mock.Anything, mock.Anything, mock.Anything, mock.Anything).
					Return([]string{"tenants/hammer-central/", "tenants/acme-corp/"}, nil)
				cloudStorageMock.On("EnsureBucketExists", mock.Anything, mock.Anything, mock.Anything, mock.Anything).
					Return(nil)
				cloudStorageMock.On("WriteObject", mock.Anything, mock.Anything, mock.Anything, mock.Anything, mock.Anything).
					Return(fmt.Errorf("some error"))
			},
			wantErr: true,
		},
		{
			name:    "failed ci tenant not found",
			oamFile: "testdata/oam_test_success.yaml",
			setupMocks: func(cloudStorageMock *MockCloudStorage, pubSubMock *MockPubSubClient) {
				cloudStorageMock.On("ListPrefixes", mock.Anything, mock.Anything, mock.Anything, mock.Anything).
					Return([]string{"tenants/hammer-central/", "tenants/something/"}, nil)
			},
			wantErr: true,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			os.Setenv("BRANCH_NAME", "main")
			os.Setenv("COMMIT_SHA", "1234567")
			os.Setenv("BUILD_ID", "12341")
			cloudStorageMock := new(MockCloudStorage)
			pubSubMock := new(MockPubSubClient)

			tt.setupMocks(cloudStorageMock, pubSubMock)
			ciCmd := &CICmd{OAMFile: tt.oamFile, CloudStorage: cloudStorageMock, PubSub: pubSubMock}

			err := ciCmd.Execute(context.Background(), []string{})
			if tt.wantErr {
				require.Error(t, err)
			} else {
				require.NoError(t, err)
			}
			cloudStorageMock.AssertExpectations(t)
			pubSubMock.AssertExpectations(t)
		})
	}
}

func TestCIExecuteAnalyze(t *testing.T) {
	tests := []struct {
		name       string
		oamFile    string
		setupMocks func(cloudStorageMock *MockCloudStorage, pubSubMock *MockPubSubClient)
		wantErr    bool
	}{
		{
			name:    "successful CI execution",
			oamFile: "testdata/oam_test_success.yaml",
			setupMocks: func(cloudStorageMock *MockCloudStorage, pubSubMock *MockPubSubClient) {
				cloudStorageMock.On("ListPrefixes", mock.Anything, mock.Anything, mock.Anything, mock.Anything).
					Return([]string{"tenants/hammer-central/", "tenants/acme-corp/"}, nil)
			},
			wantErr: false,
		},
		{
			name:    "failed CI execution",
			oamFile: "testdata/oam_test_fail_ci_failure.yaml",
			setupMocks: func(cloudStorageMock *MockCloudStorage, pubSubMock *MockPubSubClient) {
				cloudStorageMock.On("ListPrefixes", mock.Anything, mock.Anything, mock.Anything, mock.Anything).
					Return([]string{"tenants/hammer-central/", "tenants/test/"}, nil)
			},
			wantErr: true,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			os.Setenv("BRANCH_NAME", "something")
			cloudStorageMock := new(MockCloudStorage)
			pubSubMock := new(MockPubSubClient)

			tt.setupMocks(cloudStorageMock, pubSubMock)
			ciCmd := &CICmd{OAMFile: tt.oamFile, CloudStorage: cloudStorageMock, PubSub: pubSubMock}

			err := ciCmd.Execute(context.Background(), []string{})
			if tt.wantErr {
				require.Error(t, err)
			} else {
				require.NoError(t, err)
			}
			cloudStorageMock.AssertExpectations(t)
			pubSubMock.AssertExpectations(t)
		})
	}
}
