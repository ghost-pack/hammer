package gcp

import (
	"fmt"
	"testing"

	"cloud.google.com/go/cloudbuild/apiv1/v2/cloudbuildpb"
	"github.com/stretchr/testify/require"
)

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
							"sa-cloud-build@cloud-build-pipeline-396819.iam.gserviceaccount.com",
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
			got, _ := createBuildTrigger(tt.args.projectID, tt.args.projectNumber, tt.args.triggerName, tt.args.cfg)
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
						"sa-cloud-build@cloud-build-pipeline-396819.iam.gserviceaccount.com",
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
							"sa-cloud-build@cloud-build-pipeline-396819.iam.gserviceaccount.com",
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
						"sa-cloud-build@cloud-build-pipeline-396819.iam.gserviceaccount.com",
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
							"sa-cloud-build@cloud-build-pipeline-396819.iam.gserviceaccount.com",
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
						"sa-cloud-build@cloud-build-pipeline-396819.iam.gserviceaccount.com",
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
