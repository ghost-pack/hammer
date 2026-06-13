package gcp

import (
	"context"
	"fmt"
	"os"
	"time"

	cloudbuild "cloud.google.com/go/cloudbuild/apiv1"
	"cloud.google.com/go/cloudbuild/apiv1/v2/cloudbuildpb"
	"gopkg.in/yaml.v3"
)

type CloudBuildClient interface {
	TestCloudBuild(ctx context.Context, projectID, location, cloudbuildPath, cloudBuildTestPath string) error
	Close() error
}

type CloudBuildClientImpl struct {
	client *cloudbuild.Client
}

func NewCloudBuildClient(ctx context.Context) (*CloudBuildClientImpl, error) {
	client, err := cloudbuild.NewClient(ctx)
	if err != nil {
		return nil, fmt.Errorf("creating cloud build client: %w", err)
	}
	return &CloudBuildClientImpl{client: client}, nil
}

func (c *CloudBuildClientImpl) Close() error {
	return c.client.Close()
}

type cloudBuildConfig struct {
	Steps []struct {
		Name       string   `yaml:"name"`
		ID         string   `yaml:"id"`
		Entrypoint string   `yaml:"entrypoint"`
		Args       []string `yaml:"args"`
		Dir        string   `yaml:"dir"`
		Env        []string `yaml:"env"`
	} `yaml:"steps"`
}

type cloudBuildTestConfig struct {
	Substitutions map[string]string `yaml:"substitutions"`
}

func (c *CloudBuildClientImpl) TestCloudBuild(ctx context.Context, projectID, location, cloudBuildPath, cloudBuildTestPath string) error {
	buildData, err := os.ReadFile(cloudBuildPath)
	if err != nil {
		return fmt.Errorf("reading cloudbuild.yaml: %w", err)
	}
	var buildConfig cloudBuildConfig
	if err := yaml.Unmarshal(buildData, &buildConfig); err != nil {
		return fmt.Errorf("parsing cloudbuild.yaml: %w", err)
	}

	testData, err := os.ReadFile(cloudBuildTestPath)
	if err != nil {
		return fmt.Errorf("reading cloud_build_test.yaml: %w", err)
	}
	var testConfig cloudBuildTestConfig
	if err := yaml.Unmarshal(testData, &testConfig); err != nil {
		return fmt.Errorf("parsing cloud_build_test.yaml: %w", err)
	}

	var steps []*cloudbuildpb.BuildStep
	for _, s := range buildConfig.Steps {
		steps = append(steps, &cloudbuildpb.BuildStep{
			Name:       s.Name,
			Id:         s.ID,
			Entrypoint: s.Entrypoint,
			Args:       s.Args,
			Dir:        s.Dir,
			Env:        s.Env,
		})
	}
	op, err := c.client.CreateBuild(ctx, &cloudbuildpb.CreateBuildRequest{
		ProjectId: projectID,
		Parent:    fmt.Sprintf("projects/%s/locations/%s", projectID, location),
		Build: &cloudbuildpb.Build{
			Steps:         steps,
			Substitutions: testConfig.Substitutions,
			Options: &cloudbuildpb.BuildOptions{
				SubstitutionOption: cloudbuildpb.BuildOptions_ALLOW_LOOSE,
				Logging:            cloudbuildpb.BuildOptions_CLOUD_LOGGING_ONLY,
			},
		},
	})
	if err != nil {
		return fmt.Errorf("submitting build: %w", err)
	}

	meta := &cloudbuildpb.BuildOperationMetadata{}
	if err := op.Metadata.UnmarshalTo(meta); err != nil {
		return fmt.Errorf("unmarshaling operation metadata: %w", err)
	}

	buildName := fmt.Sprintf("projects/%s/locations/%s/builds/%s", projectID, location, meta.Build.Id)

	for {
		build, err := c.client.GetBuild(ctx, &cloudbuildpb.GetBuildRequest{
			Name: buildName,
		})
		if err != nil {
			return fmt.Errorf("getting build status: %w", err)
		}
		switch build.Status {
		case cloudbuildpb.Build_SUCCESS:
			return nil
		case cloudbuildpb.Build_FAILURE, cloudbuildpb.Build_INTERNAL_ERROR,
			cloudbuildpb.Build_TIMEOUT, cloudbuildpb.Build_CANCELLED:
			return fmt.Errorf("build failed with status: %s", build.Status)
		}
		time.Sleep(10 * time.Second)
	}
}
