package gcp

import (
	"context"
	"fmt"
	"strings"
	"testing"

	"cloud.google.com/go/cloudbuild/apiv1/v2/cloudbuildpb"
	"cloud.google.com/go/longrunning/autogen/longrunningpb"
	"github.com/googleapis/gax-go/v2"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
	"google.golang.org/api/option"
	"google.golang.org/protobuf/types/known/anypb"
)

func makeTestOperation(t *testing.T, buildID string) *longrunningpb.Operation {
	t.Helper()
	meta := &cloudbuildpb.BuildOperationMetadata{
		Build: &cloudbuildpb.Build{
			Id: buildID,
		},
	}
	anyMeta, err := anypb.New(meta)
	require.NoError(t, err)
	return &longrunningpb.Operation{
		Metadata: anyMeta,
	}
}

type MockCloudBuildApi struct {
	mock.Mock
}

func (m *MockCloudBuildApi) CreateBuild(ctx context.Context, req *cloudbuildpb.CreateBuildRequest, opts ...gax.CallOption) (*longrunningpb.Operation, error) {
	args := m.Called(ctx, req)
	op, _ := args.Get(0).(*longrunningpb.Operation)
	return op, args.Error(1)
}

func (m *MockCloudBuildApi) GetBuild(ctx context.Context, req *cloudbuildpb.GetBuildRequest, opts ...gax.CallOption) (*cloudbuildpb.Build, error) {
	args := m.Called(ctx, req)
	op, _ := args.Get(0).(*cloudbuildpb.Build)
	return op, args.Error(1)
}

func (m *MockCloudBuildApi) ListBuildTriggers(ctx context.Context, req *cloudbuildpb.ListBuildTriggersRequest, opts ...gax.CallOption) (*cloudbuildpb.ListBuildTriggersResponse, error) {
	args := m.Called(ctx, req)
	op, _ := args.Get(0).(*cloudbuildpb.ListBuildTriggersResponse)
	return op, args.Error(1)
}

func (m *MockCloudBuildApi) CreateBuildTrigger(ctx context.Context, req *cloudbuildpb.CreateBuildTriggerRequest, opts ...gax.CallOption) (*cloudbuildpb.BuildTrigger, error) {
	args := m.Called(ctx, req)
	op, _ := args.Get(0).(*cloudbuildpb.BuildTrigger)
	return op, args.Error(1)
}

func (m *MockCloudBuildApi) UpdateBuildTrigger(ctx context.Context, req *cloudbuildpb.UpdateBuildTriggerRequest, opts ...gax.CallOption) (*cloudbuildpb.BuildTrigger, error) {
	args := m.Called(ctx, req)
	op, _ := args.Get(0).(*cloudbuildpb.BuildTrigger)
	return op, args.Error(1)
}

func (m *MockCloudBuildApi) Close() error {
	args := m.Called()
	return args.Error(0)
}

func TestTestCloudBuild(t *testing.T) {
	t.Run("succeeds when build completes successfully", func(t *testing.T) {
		mockAPI := &MockCloudBuildApi{}
		mockAPI.On("CreateBuild", mock.Anything, mock.Anything).
			Return(makeTestOperation(t, "build-123"), nil)
		mockAPI.On("GetBuild", mock.Anything, mock.MatchedBy(func(req *cloudbuildpb.GetBuildRequest) bool {
			return strings.Contains(req.Name, "build-123")
		})).Return(&cloudbuildpb.Build{
			Status: cloudbuildpb.Build_SUCCESS,
		}, nil)

		client := newCloudBuildClientWithAPI(mockAPI)
		err := client.TestCloudBuild(context.Background(), "my-project", "us-central1", "testdata/cloudbuild.yaml", "testdata/cloudbuild_test.yaml")

		require.NoError(t, err)
		mockAPI.AssertExpectations(t)
	})

	t.Run("fails when unable to parse cloud build yaml", func(t *testing.T) {
		mockAPI := &MockCloudBuildApi{}
		client := newCloudBuildClientWithAPI(mockAPI)
		err := client.TestCloudBuild(context.Background(), "my-project", "us-central1", "testdata/cloudbuild_bad_yaml.yaml", "testdata/cloudbuild_test.yaml")

		require.Error(t, err)
		mockAPI.AssertExpectations(t)
	})

	t.Run("fails when unable to parse cloud build test yaml", func(t *testing.T) {
		mockAPI := &MockCloudBuildApi{}
		client := newCloudBuildClientWithAPI(mockAPI)
		err := client.TestCloudBuild(context.Background(), "my-project", "us-central1", "testdata/cloudbuild.yaml", "testdata/cloudbuild_test_bad_yaml.yaml")

		require.Error(t, err)
		mockAPI.AssertExpectations(t)
	})

	t.Run("fails when create build fials", func(t *testing.T) {
		mockAPI := &MockCloudBuildApi{}
		mockAPI.On("CreateBuild", mock.Anything, mock.Anything).
			Return(nil, fmt.Errorf("some error"))

		client := newCloudBuildClientWithAPI(mockAPI)
		err := client.TestCloudBuild(context.Background(), "my-project", "us-central1", "testdata/cloudbuild.yaml", "testdata/cloudbuild_test.yaml")

		require.Error(t, err)
		mockAPI.AssertExpectations(t)
	})

	t.Run("fails when unmarshal fails", func(t *testing.T) {
		mockAPI := &MockCloudBuildApi{}
		mockAPI.On("CreateBuild", mock.Anything, mock.Anything).
			Return(&longrunningpb.Operation{}, nil)

		client := newCloudBuildClientWithAPI(mockAPI)
		err := client.TestCloudBuild(context.Background(), "my-project", "us-central1", "testdata/cloudbuild.yaml", "testdata/cloudbuild_test.yaml")

		require.Error(t, err)
		mockAPI.AssertExpectations(t)
	})

	t.Run("fails when can't get build status", func(t *testing.T) {
		mockAPI := &MockCloudBuildApi{}
		mockAPI.On("CreateBuild", mock.Anything, mock.Anything).
			Return(makeTestOperation(t, "build-123"), nil)
		mockAPI.On("GetBuild", mock.Anything, mock.MatchedBy(func(req *cloudbuildpb.GetBuildRequest) bool {
			return strings.Contains(req.Name, "build-123")
		})).Return(nil, fmt.Errorf("some error"))

		client := newCloudBuildClientWithAPI(mockAPI)
		err := client.TestCloudBuild(context.Background(), "my-project", "us-central1", "testdata/cloudbuild.yaml", "testdata/cloudbuild_test.yaml")

		require.Error(t, err)
		mockAPI.AssertExpectations(t)
	})

	t.Run("fails when build status is bad", func(t *testing.T) {
		mockAPI := &MockCloudBuildApi{}
		mockAPI.On("CreateBuild", mock.Anything, mock.Anything).
			Return(makeTestOperation(t, "build-123"), nil)
		mockAPI.On("GetBuild", mock.Anything, mock.MatchedBy(func(req *cloudbuildpb.GetBuildRequest) bool {
			return strings.Contains(req.Name, "build-123")
		})).Return(&cloudbuildpb.Build{
			Status: cloudbuildpb.Build_FAILURE,
		}, nil)

		client := newCloudBuildClientWithAPI(mockAPI)
		err := client.TestCloudBuild(context.Background(), "my-project", "us-central1", "testdata/cloudbuild.yaml", "testdata/cloudbuild_test.yaml")

		require.Error(t, err)
		mockAPI.AssertExpectations(t)
	})

	t.Run("succeeds when checking build twice", func(t *testing.T) {
		mockAPI := &MockCloudBuildApi{}
		mockAPI.On("CreateBuild", mock.Anything, mock.Anything).
			Return(makeTestOperation(t, "build-123"), nil)
		mockAPI.On("GetBuild", mock.Anything, mock.MatchedBy(func(req *cloudbuildpb.GetBuildRequest) bool {
			return strings.Contains(req.Name, "build-123")
		})).Return(&cloudbuildpb.Build{
			Status: cloudbuildpb.Build_WORKING,
		}, nil).Once()

		mockAPI.On("GetBuild", mock.Anything, mock.MatchedBy(func(req *cloudbuildpb.GetBuildRequest) bool {
			return strings.Contains(req.Name, "build-123")
		})).Return(&cloudbuildpb.Build{
			Status: cloudbuildpb.Build_SUCCESS,
		}, nil)

		client := newCloudBuildClientWithAPI(mockAPI)
		err := client.TestCloudBuild(context.Background(), "my-project", "us-central1", "testdata/cloudbuild.yaml", "testdata/cloudbuild_test.yaml")

		require.NoError(t, err)
		mockAPI.AssertExpectations(t)
	})

}

func TestCreateOrUpdateCloudBuildTrigger(t *testing.T) {
	t.Run("fails to parse cloud build yaml", func(t *testing.T) {
		mockAPI := &MockCloudBuildApi{}

		client := newCloudBuildClientWithAPI(mockAPI)
		err := client.CreateOrUpdateCloudBuildTrigger(context.Background(), "my-project", "12345", "us-central1", "testdata/cloudbuild_bad_yaml.yaml", "webhook", "my_trigger")

		require.Error(t, err)
		mockAPI.AssertExpectations(t)
	})

	t.Run("fails to create trigger due to bad trigger type", func(t *testing.T) {
		mockAPI := &MockCloudBuildApi{}

		client := newCloudBuildClientWithAPI(mockAPI)
		err := client.CreateOrUpdateCloudBuildTrigger(context.Background(), "my-project", "12345", "us-central1", "testdata/cloudbuild.yaml", "asdfoydfs", "my_trigger")

		require.Error(t, err)
		mockAPI.AssertExpectations(t)
	})

	t.Run("fails to find trigger", func(t *testing.T) {
		mockAPI := &MockCloudBuildApi{}

		mockAPI.On("ListBuildTriggers", mock.Anything, mock.Anything).
			Return(nil, fmt.Errorf("some error"))

		client := newCloudBuildClientWithAPI(mockAPI)
		err := client.CreateOrUpdateCloudBuildTrigger(context.Background(), "my-project", "12345", "us-central1", "testdata/cloudbuild.yaml", "webhook", "my_trigger")

		require.Error(t, err)
		mockAPI.AssertExpectations(t)
	})

	t.Run("fails to update existing trigger", func(t *testing.T) {
		mockAPI := &MockCloudBuildApi{}

		mockAPI.On("ListBuildTriggers", mock.Anything, mock.Anything).
			Return(&cloudbuildpb.ListBuildTriggersResponse{
				Triggers: []*cloudbuildpb.BuildTrigger{
					{Name: "my_trigger"},
				},
			}, nil)

		mockAPI.On("UpdateBuildTrigger", mock.Anything, mock.Anything).
			Return(nil, fmt.Errorf("some error"))

		client := newCloudBuildClientWithAPI(mockAPI)
		err := client.CreateOrUpdateCloudBuildTrigger(context.Background(), "my-project", "12345", "us-central1", "testdata/cloudbuild.yaml", "webhook", "my_trigger")

		require.Error(t, err)
		mockAPI.AssertExpectations(t)
	})

	t.Run("successfully update existing trigger", func(t *testing.T) {
		mockAPI := &MockCloudBuildApi{}

		mockAPI.On("ListBuildTriggers", mock.Anything, mock.Anything).
			Return(&cloudbuildpb.ListBuildTriggersResponse{
				Triggers: []*cloudbuildpb.BuildTrigger{
					{Name: "my_trigger"},
				},
			}, nil)

		mockAPI.On("UpdateBuildTrigger", mock.Anything, mock.Anything).
			Return(nil, nil)

		client := newCloudBuildClientWithAPI(mockAPI)
		err := client.CreateOrUpdateCloudBuildTrigger(context.Background(), "my-project", "12345", "us-central1", "testdata/cloudbuild.yaml", "webhook", "my_trigger")

		require.NoError(t, err)
		mockAPI.AssertExpectations(t)
	})

	t.Run("fails to create new trigger", func(t *testing.T) {
		mockAPI := &MockCloudBuildApi{}

		mockAPI.On("ListBuildTriggers", mock.Anything, mock.Anything).
			Return(&cloudbuildpb.ListBuildTriggersResponse{
				Triggers: []*cloudbuildpb.BuildTrigger{},
			}, nil)

		mockAPI.On("CreateBuildTrigger", mock.Anything, mock.Anything).
			Return(nil, fmt.Errorf("some error"))

		client := newCloudBuildClientWithAPI(mockAPI)
		err := client.CreateOrUpdateCloudBuildTrigger(context.Background(), "my-project", "12345", "us-central1", "testdata/cloudbuild.yaml", "webhook", "my_trigger")

		require.Error(t, err)
		mockAPI.AssertExpectations(t)
	})

	t.Run("successfully create new trigger", func(t *testing.T) {
		mockAPI := &MockCloudBuildApi{}

		mockAPI.On("ListBuildTriggers", mock.Anything, mock.Anything).
			Return(&cloudbuildpb.ListBuildTriggersResponse{
				Triggers: []*cloudbuildpb.BuildTrigger{},
			}, nil)

		mockAPI.On("CreateBuildTrigger", mock.Anything, mock.Anything).
			Return(nil, nil)

		client := newCloudBuildClientWithAPI(mockAPI)
		err := client.CreateOrUpdateCloudBuildTrigger(context.Background(), "my-project", "12345", "us-central1", "testdata/cloudbuild.yaml", "webhook", "my_trigger")

		require.NoError(t, err)
		mockAPI.AssertExpectations(t)
	})
}

func Test_buildSteps(t *testing.T) {
	type args struct {
		cfg *cloudBuildConfig
	}
	tests := []struct {
		name string
		args args
		want []*cloudbuildpb.BuildStep
	}{
		{
			name: "success",
			args: args{
				cfg: &cloudBuildConfig{
					Steps: []struct {
						Name       string   `yaml:"name"`
						ID         string   `yaml:"id"`
						Entrypoint string   `yaml:"entrypoint"`
						Args       []string `yaml:"args"`
						Dir        string   `yaml:"dir"`
						Env        []string `yaml:"env"`
					}{
						{
							Name:       "golang",
							ID:         "test",
							Entrypoint: "bash",
							Args:       []string{"go", "test", "./..."},
							Dir:        ".",
							Env:        []string{"FOO=bar"},
						},
					},
					Substitutions: map[string]string{
						"_FOO": "bar",
					},
				},
			},
			want: []*cloudbuildpb.BuildStep{
				{
					Name:       "golang",
					Id:         "test",
					Entrypoint: "bash",
					Args:       []string{"go", "test", "./..."},
					Dir:        ".",
					Env:        []string{"FOO=bar"},
				},
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := buildSteps(tt.args.cfg)

			require.Equal(t, tt.want, got)
		})
	}
}

func Test_createBuild(t *testing.T) {
	type args struct {
		projectID     string
		substitutions map[string]string
		steps         []*cloudbuildpb.BuildStep
	}
	tests := []struct {
		name string
		args args
		want *cloudbuildpb.Build
	}{
		{
			name: "success",
			args: args{
				projectID: "project",
				substitutions: map[string]string{
					"_ENV":     "dev",
					"_PROJECT": "my-project",
				},
				steps: []*cloudbuildpb.BuildStep{
					{
						Name:       "golang",
						Id:         "test",
						Entrypoint: "bash",
						Args:       []string{"go", "test", "./..."},
						Dir:        ".",
						Env:        []string{"FOO=bar"},
					},
				},
			},
			want: &cloudbuildpb.Build{
				ProjectId: "project",
				Substitutions: map[string]string{
					"_ENV":     "dev",
					"_PROJECT": "my-project",
				},
				Steps: []*cloudbuildpb.BuildStep{
					{
						Name:       "golang",
						Id:         "test",
						Entrypoint: "bash",
						Args:       []string{"go", "test", "./..."},
						Dir:        ".",
						Env:        []string{"FOO=bar"},
					},
				},
				Options: &cloudbuildpb.BuildOptions{
					SubstitutionOption: cloudbuildpb.BuildOptions_ALLOW_LOOSE,
					Logging:            cloudbuildpb.BuildOptions_CLOUD_LOGGING_ONLY,
				},
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := createBuild(tt.args.projectID, tt.args.substitutions, tt.args.steps)
			require.Equal(t, tt.want, got)
		})
	}
}

func Test_createBuildTrigger(t *testing.T) {
	type args struct {
		projectID     string
		projectNumber string
		triggerName   string
		cfg           *cloudBuildConfig
	}
	tests := []struct {
		name    string
		args    args
		want    *cloudbuildpb.BuildTrigger
		wantErr bool
	}{
		{
			name: "success",
			args: args{
				projectID:     "project",
				projectNumber: "123456789",
				triggerName:   "trigger",
				cfg: &cloudBuildConfig{
					Steps: []struct {
						Name       string   `yaml:"name"`
						ID         string   `yaml:"id"`
						Entrypoint string   `yaml:"entrypoint"`
						Args       []string `yaml:"args"`
						Dir        string   `yaml:"dir"`
						Env        []string `yaml:"env"`
					}{
						{
							Name:       "golang",
							ID:         "test",
							Entrypoint: "bash",
							Args:       []string{"go", "test", "./..."},
							Dir:        ".",
							Env:        []string{"FOO=bar"},
						},
					},
					Substitutions: map[string]string{
						"_ENV":     "dev",
						"_PROJECT": "my-project",
					},
				},
			},
			want: &cloudbuildpb.BuildTrigger{
				Name: "trigger",
				Substitutions: map[string]string{
					"_ENV":     "dev",
					"_PROJECT": "my-project",
				},
				WebhookConfig: &cloudbuildpb.WebhookConfig{
					AuthMethod: &cloudbuildpb.WebhookConfig_Secret{
						Secret: fmt.Sprintf(
							"projects/%s/secrets/%s/versions/latest",
							"123456789",
							"cloudbuild-webhook-secret",
						),
					},
				},
				BuildTemplate: &cloudbuildpb.BuildTrigger_Build{
					Build: &cloudbuildpb.Build{
						ServiceAccount: fmt.Sprintf(
							"projects/%s/serviceAccounts/%s",
							"project",
							"sa-pipeline@hammer-central-prod.iam.gserviceaccount.com",
						),
						Substitutions: map[string]string{
							"_ENV":     "dev",
							"_PROJECT": "my-project",
						},
						Steps: []*cloudbuildpb.BuildStep{
							{
								Name:       "golang",
								Id:         "test",
								Entrypoint: "bash",
								Args:       []string{"go", "test", "./..."},
								Dir:        ".",
								Env:        []string{"FOO=bar"},
							},
						},
						Options: &cloudbuildpb.BuildOptions{
							SubstitutionOption: cloudbuildpb.BuildOptions_ALLOW_LOOSE,
							Logging:            cloudbuildpb.BuildOptions_CLOUD_LOGGING_ONLY,
						},
					},
				},
			},
			wantErr: false,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, _ := createBuildTrigger(tt.args.projectID, tt.args.projectNumber, tt.args.triggerName, "webhook", tt.args.cfg)
			require.Equal(t, tt.want, got)
		})
	}
}

func Test_makeTrigger(t *testing.T) {
	type args struct {
		name          string
		pipelineType  string
		webhookSecret string
		pubSubTopic   string
		build         *cloudbuildpb.Build
		subs          map[string]string
	}
	tests := []struct {
		name    string
		args    args
		want    *cloudbuildpb.BuildTrigger
		wantErr bool
	}{
		{
			name: "PubSub",
			args: args{
				name:          "golang",
				pipelineType:  "pubsub",
				webhookSecret: "",
				pubSubTopic:   "my-topic",
				build: &cloudbuildpb.Build{
					ServiceAccount: fmt.Sprintf(
						"projects/%s/serviceAccounts/%s",
						"project",
						"sa-pipeline@hammer-central-prod.iam.gserviceaccount.com",
					),
					Substitutions: map[string]string{
						"_ENV":     "dev",
						"_PROJECT": "my-project",
					},
					Steps: []*cloudbuildpb.BuildStep{
						{
							Name:       "golang",
							Id:         "test",
							Entrypoint: "bash",
							Args:       []string{"go", "test", "./..."},
							Dir:        ".",
							Env:        []string{"FOO=bar"},
						},
					},
					Options: &cloudbuildpb.BuildOptions{
						SubstitutionOption: cloudbuildpb.BuildOptions_ALLOW_LOOSE,
						Logging:            cloudbuildpb.BuildOptions_CLOUD_LOGGING_ONLY,
					},
				},
				subs: map[string]string{
					"_ENV":     "dev",
					"_PROJECT": "my-project",
				},
			},
			want: &cloudbuildpb.BuildTrigger{
				Name: "golang",
				Substitutions: map[string]string{
					"_ENV":     "dev",
					"_PROJECT": "my-project",
				},
				PubsubConfig: &cloudbuildpb.PubsubConfig{
					Topic: "my-topic",
				},
				BuildTemplate: &cloudbuildpb.BuildTrigger_Build{
					Build: &cloudbuildpb.Build{
						ServiceAccount: fmt.Sprintf(
							"projects/%s/serviceAccounts/%s",
							"project",
							"sa-pipeline@hammer-central-prod.iam.gserviceaccount.com",
						),
						Substitutions: map[string]string{
							"_ENV":     "dev",
							"_PROJECT": "my-project",
						},
						Steps: []*cloudbuildpb.BuildStep{
							{
								Name:       "golang",
								Id:         "test",
								Entrypoint: "bash",
								Args:       []string{"go", "test", "./..."},
								Dir:        ".",
								Env:        []string{"FOO=bar"},
							},
						},
						Options: &cloudbuildpb.BuildOptions{
							SubstitutionOption: cloudbuildpb.BuildOptions_ALLOW_LOOSE,
							Logging:            cloudbuildpb.BuildOptions_CLOUD_LOGGING_ONLY,
						},
					},
				},
			},
			wantErr: false,
		},
		{
			name: "Manaul",
			args: args{
				name:          "golang",
				pipelineType:  "manual",
				webhookSecret: "",
				pubSubTopic:   "",
				build: &cloudbuildpb.Build{
					ServiceAccount: fmt.Sprintf(
						"projects/%s/serviceAccounts/%s",
						"project",
						"sa-pipeline@hammer-central-prod.iam.gserviceaccount.com",
					),
					Substitutions: map[string]string{
						"_ENV":     "dev",
						"_PROJECT": "my-project",
					},
					Steps: []*cloudbuildpb.BuildStep{
						{
							Name:       "golang",
							Id:         "test",
							Entrypoint: "bash",
							Args:       []string{"go", "test", "./..."},
							Dir:        ".",
							Env:        []string{"FOO=bar"},
						},
					},
					Options: &cloudbuildpb.BuildOptions{
						SubstitutionOption: cloudbuildpb.BuildOptions_ALLOW_LOOSE,
						Logging:            cloudbuildpb.BuildOptions_CLOUD_LOGGING_ONLY,
					},
				},
				subs: map[string]string{
					"_ENV":     "dev",
					"_PROJECT": "my-project",
				},
			},
			want: &cloudbuildpb.BuildTrigger{
				Name: "golang",
				Substitutions: map[string]string{
					"_ENV":     "dev",
					"_PROJECT": "my-project",
				},
				BuildTemplate: &cloudbuildpb.BuildTrigger_Build{
					Build: &cloudbuildpb.Build{
						ServiceAccount: fmt.Sprintf(
							"projects/%s/serviceAccounts/%s",
							"project",
							"sa-pipeline@hammer-central-prod.iam.gserviceaccount.com",
						),
						Substitutions: map[string]string{
							"_ENV":     "dev",
							"_PROJECT": "my-project",
						},
						Steps: []*cloudbuildpb.BuildStep{
							{
								Name:       "golang",
								Id:         "test",
								Entrypoint: "bash",
								Args:       []string{"go", "test", "./..."},
								Dir:        ".",
								Env:        []string{"FOO=bar"},
							},
						},
						Options: &cloudbuildpb.BuildOptions{
							SubstitutionOption: cloudbuildpb.BuildOptions_ALLOW_LOOSE,
							Logging:            cloudbuildpb.BuildOptions_CLOUD_LOGGING_ONLY,
						},
					},
				},
			},
			wantErr: false,
		},
		{
			name: "Failure_some_other_trigger",
			args: args{
				name:          "golang",
				pipelineType:  "asdfadsf",
				webhookSecret: "",
				pubSubTopic:   "my-topic",
				build: &cloudbuildpb.Build{
					ServiceAccount: fmt.Sprintf(
						"projects/%s/serviceAccounts/%s",
						"project",
						"sa-pipeline@hammer-central-prod.iam.gserviceaccount.com",
					),
					Substitutions: map[string]string{
						"_ENV":     "dev",
						"_PROJECT": "my-project",
					},
					Steps: []*cloudbuildpb.BuildStep{
						{
							Name:       "golang",
							Id:         "test",
							Entrypoint: "bash",
							Args:       []string{"go", "test", "./..."},
							Dir:        ".",
							Env:        []string{"FOO=bar"},
						},
					},
					Options: &cloudbuildpb.BuildOptions{
						SubstitutionOption: cloudbuildpb.BuildOptions_ALLOW_LOOSE,
						Logging:            cloudbuildpb.BuildOptions_CLOUD_LOGGING_ONLY,
					},
				},
				subs: map[string]string{
					"_ENV":     "dev",
					"_PROJECT": "my-project",
				},
			},
			wantErr: true,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := makeTrigger(tt.args.name, tt.args.pipelineType, tt.args.webhookSecret, tt.args.pubSubTopic, tt.args.build, tt.args.subs)
			if err != nil {
				if !tt.wantErr {
					require.Fail(t, err.Error())
				} else {
					require.Error(t, err)
				}
			}
			require.Equal(t, tt.want, got)
		})
	}
}

func Test_parseCloudBuild(t *testing.T) {
	type args struct {
		path string
	}
	tests := []struct {
		name    string
		args    args
		want    *cloudBuildConfig
		wantErr bool
	}{
		{
			name: "Success",
			args: args{
				path: "./testdata/cloudbuild.yaml",
			},
			want: &cloudBuildConfig{
				Steps: []struct {
					Name       string   `yaml:"name"`
					ID         string   `yaml:"id"`
					Entrypoint string   `yaml:"entrypoint"`
					Args       []string `yaml:"args"`
					Dir        string   `yaml:"dir"`
					Env        []string `yaml:"env"`
				}{
					{
						Name:       "ubuntu",
						ID:         "test",
						Entrypoint: "bash",
						Args:       []string{"-c", "echo \"$_GIT_CLONE_URL\""},
						Dir:        "",
						Env:        nil,
					},
				},
				Substitutions: map[string]string{
					"_GIT_REPOSITORY_NAME": "$(body.repository.name)",
					"_GIT_CLONE_URL":       "$(body.repository.clone_url)",
					"_GIT_REF":             "$(body.ref)",
					"_GIT_HEAD_SHA":        "$(body.head_commit.id)",
				},
			},
			wantErr: false,
		},
		{
			name: "Failed_bad_yaml",
			args: args{
				path: "./testdata/cloudbuild_bad_yaml.yaml",
			},
			want:    nil,
			wantErr: true,
		},
		{
			name: "Failed_bad_path",
			args: args{
				path: "./testdata/nothinghere.yaml",
			},
			want:    nil,
			wantErr: true,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := parseCloudBuild(tt.args.path)
			if tt.wantErr {
				require.Error(t, err)
			}
			require.Equal(t, tt.want, got)
		})
	}
}

func Test_parseCloudBuildTest(t *testing.T) {
	type args struct {
		path string
	}
	tests := []struct {
		name    string
		args    args
		want    *cloudBuildTestConfig
		wantErr bool
	}{
		{
			name: "Success",
			args: args{
				path: "./testdata/cloudbuild_test.yaml",
			},
			want: &cloudBuildTestConfig{
				Substitutions: map[string]string{
					"_GIT_REPOSITORY_NAME": "test-cloudbuild-failure",
					"_GIT_CLONE_URL":       "https://github.com/brianpipeline/test-cloudbuild-failure.git",
					"_GIT_REF":             "refs/heads/main",
					"_GIT_HEAD_SHA":        "b3fa043e677500882e689fc1a978d96056d6702d",
				},
			},
			wantErr: false,
		},
		{
			name: "Failed_bad_yaml",
			args: args{
				path: "./testdata/cloudbuild_test_bad_yaml.yaml",
			},
			want:    nil,
			wantErr: true,
		},
		{
			name: "Failed_bad_path",
			args: args{
				path: "./testdata/nothinghere.yaml",
			},
			want:    nil,
			wantErr: true,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := parseCloudBuildTest(tt.args.path)
			if tt.wantErr {
				require.Error(t, err)
			}
			require.Equal(t, tt.want, got)
		})
	}
}

func TestNewCloudBuildClient(t *testing.T) {
	tests := []struct {
		name                  string
		setupCloudBuildClient func(ctx context.Context, opts ...option.ClientOption) (CloudBuildClient, error)
		wantErr               bool
	}{
		{
			name: "failed client creation",
			setupCloudBuildClient: func(ctx context.Context, opts ...option.ClientOption) (CloudBuildClient, error) {
				client, err := NewCloudBuildClient(ctx, opts...)
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
			_, creationErr := tt.setupCloudBuildClient(context.Background(), option.WithCredentialsFile("/nonexistent/credentials.json"))
			if creationErr != nil && tt.wantErr {
				require.Error(t, creationErr)
			} else {
				require.NoError(t, creationErr)
			}
		})
	}
}
