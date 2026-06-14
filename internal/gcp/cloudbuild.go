package gcp

import (
	"context"
	"fmt"
	"os"
	"time"

	cloudbuild "cloud.google.com/go/cloudbuild/apiv1"
	"cloud.google.com/go/cloudbuild/apiv1/v2/cloudbuildpb"
	"cloud.google.com/go/longrunning/autogen/longrunningpb"
	"gopkg.in/yaml.v3"
)

type CloudBuildClient interface {
	TestCloudBuild(ctx context.Context, projectID, location, cloudbuildPath, cloudBuildTestPath string) error
	CreateOrUpdateCloudBuildTrigger(ctx context.Context, projectId, projectNumber, location, cloudBuildPath, triggerName string) error
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
	Substitutions map[string]string `yaml:"substitutions"`
}

type cloudBuildTestConfig struct {
	Substitutions map[string]string `yaml:"substitutions"`
}

func parseCloudBuild(path string) (*cloudBuildConfig, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("reading cloudbuild.yaml: %w", err)
	}

	var cfg cloudBuildConfig
	if err := yaml.Unmarshal(data, &cfg); err != nil {
		return nil, fmt.Errorf("parsing cloudbuild.yaml: %w", err)
	}

	return &cfg, nil
}

func parseCloudBuildTest(path string) (*cloudBuildTestConfig, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("reading cloud_build_test.yaml: %w", err)
	}

	var cfg cloudBuildTestConfig
	if err := yaml.Unmarshal(data, &cfg); err != nil {
		return nil, fmt.Errorf("parsing cloud_build_test.yaml: %w", err)
	}

	return &cfg, nil
}

func buildSteps(cfg *cloudBuildConfig) []*cloudbuildpb.BuildStep {
	steps := make([]*cloudbuildpb.BuildStep, 0, len(cfg.Steps))

	for _, s := range cfg.Steps {
		steps = append(steps, &cloudbuildpb.BuildStep{
			Name:       s.Name,
			Id:         s.ID,
			Entrypoint: s.Entrypoint,
			Args:       s.Args,
			Dir:        s.Dir,
			Env:        s.Env,
		})
	}

	return steps
}

func createBuild(
	projectID string,
	substitutions map[string]string,
	steps []*cloudbuildpb.BuildStep,
) *cloudbuildpb.Build {
	return &cloudbuildpb.Build{
		ProjectId:     projectID,
		Steps:         steps,
		Substitutions: substitutions,
		Options: &cloudbuildpb.BuildOptions{
			SubstitutionOption: cloudbuildpb.BuildOptions_ALLOW_LOOSE,
			Logging:            cloudbuildpb.BuildOptions_CLOUD_LOGGING_ONLY,
		},
	}
}

func (c *CloudBuildClientImpl) TestCloudBuild(
	ctx context.Context,
	projectID,
	location,
	cloudBuildPath,
	cloudBuildTestPath string,
) error {
	buildConfig, err := parseCloudBuild(cloudBuildPath)
	if err != nil {
		return err
	}

	testConfig, err := parseCloudBuildTest(cloudBuildTestPath)
	if err != nil {
		return err
	}

	build := createBuild(
		projectID,
		testConfig.Substitutions,
		buildSteps(buildConfig),
	)

	op, err := c.client.CreateBuild(ctx, &cloudbuildpb.CreateBuildRequest{
		ProjectId: projectID,
		Parent:    fmt.Sprintf("projects/%s/locations/%s", projectID, location),
		Build:     build,
	})
	if err != nil {
		return fmt.Errorf("submitting build: %w", err)
	}

	return c.waitForBuild(ctx, projectID, location, op)
}

func (c *CloudBuildClientImpl) CreateOrUpdateCloudBuildTrigger(
	ctx context.Context,
	projectID,
	projectNumber,
	location,
	cloudBuildPath,
	triggerName string,
) error {
	buildConfig, err := parseCloudBuild(cloudBuildPath)
	if err != nil {
		return err
	}

	trigger, err := createBuildTrigger(
		projectID,
		projectNumber,
		triggerName,
		buildConfig,
	)
	if err != nil {
		return err
	}

	existingTrigger, err := c.findTrigger(ctx, projectID, triggerName)
	if err != nil {
		return err
	}

	if existingTrigger == nil {
		return c.createTrigger(
			ctx,
			projectID,
			location,
			trigger,
		)
	}

	return c.updateTrigger(
		ctx,
		projectID,
		existingTrigger.Id,
		trigger,
	)
}

func (c *CloudBuildClientImpl) waitForBuild(
	ctx context.Context,
	projectID,
	location string,
	op *longrunningpb.Operation,
) error {
	meta := &cloudbuildpb.BuildOperationMetadata{}
	if err := op.Metadata.UnmarshalTo(meta); err != nil {
		return fmt.Errorf("unmarshaling operation metadata: %w", err)
	}

	buildName := fmt.Sprintf(
		"projects/%s/locations/%s/builds/%s",
		projectID,
		location,
		meta.Build.Id,
	)

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

		case cloudbuildpb.Build_FAILURE,
			cloudbuildpb.Build_INTERNAL_ERROR,
			cloudbuildpb.Build_TIMEOUT,
			cloudbuildpb.Build_CANCELLED:
			return fmt.Errorf("build failed with status: %s", build.Status)
		}

		time.Sleep(10 * time.Second)
	}
}

func createBuildTrigger(
	projectID,
	projectNumber,
	triggerName string,
	cfg *cloudBuildConfig,
) (*cloudbuildpb.BuildTrigger, error) {
	secretResourceName := fmt.Sprintf(
		"projects/%s/secrets/%s/versions/latest",
		projectNumber,
		"cloudbuild-webhook-secret",
	)

	serviceAccountName := fmt.Sprintf(
		"projects/%s/serviceAccounts/%s",
		projectID,
		"sa-cloud-build@cloud-build-pipeline-396819.iam.gserviceaccount.com",
	)

	build := &cloudbuildpb.Build{
		Steps:          buildSteps(cfg),
		Substitutions:  cfg.Substitutions,
		ServiceAccount: serviceAccountName,
		Options: &cloudbuildpb.BuildOptions{
			SubstitutionOption: cloudbuildpb.BuildOptions_ALLOW_LOOSE,
			Logging:            cloudbuildpb.BuildOptions_CLOUD_LOGGING_ONLY,
		},
	}

	return makeTrigger(
		triggerName,
		"webhook",
		secretResourceName,
		"",
		build,
		cfg.Substitutions,
	)
}

func (c *CloudBuildClientImpl) findTrigger(
	ctx context.Context,
	projectID,
	triggerName string,
) (*cloudbuildpb.BuildTrigger, error) {
	triggers, err := c.client.ListBuildTriggers(
		ctx,
		&cloudbuildpb.ListBuildTriggersRequest{
			ProjectId: projectID,
		},
	)
	if err != nil {
		return nil, fmt.Errorf("listing triggers: %w", err)
	}

	for _, trigger := range triggers.Triggers {
		if trigger.Name == triggerName {
			return trigger, nil
		}
	}

	return nil, nil
}

func (c *CloudBuildClientImpl) createTrigger(
	ctx context.Context,
	projectID,
	location string,
	trigger *cloudbuildpb.BuildTrigger,
) error {
	_, err := c.client.CreateBuildTrigger(
		ctx,
		&cloudbuildpb.CreateBuildTriggerRequest{
			ProjectId: projectID,
			Parent:    fmt.Sprintf("projects/%s/locations/%s", projectID, location),
			Trigger:   trigger,
		},
	)

	if err != nil {
		return fmt.Errorf("creating trigger: %w", err)
	}

	return nil
}

func (c *CloudBuildClientImpl) updateTrigger(
	ctx context.Context,
	projectID,
	triggerID string,
	trigger *cloudbuildpb.BuildTrigger,
) error {
	_, err := c.client.UpdateBuildTrigger(
		ctx,
		&cloudbuildpb.UpdateBuildTriggerRequest{
			ProjectId: projectID,
			TriggerId: triggerID,
			Trigger:   trigger,
		},
	)

	if err != nil {
		return fmt.Errorf("updating trigger: %w", err)
	}

	return nil
}

func makeTrigger(
	name, pipelineType, webhookSecret, pubSubTopic string,
	build *cloudbuildpb.Build,
	subs map[string]string,
) (*cloudbuildpb.BuildTrigger, error) {
	t := &cloudbuildpb.BuildTrigger{
		Name:          name,
		Substitutions: subs,
		BuildTemplate: &cloudbuildpb.BuildTrigger_Build{Build: build},
	}

	switch pipelineType {
	case "webhook":
		t.WebhookConfig = &cloudbuildpb.WebhookConfig{
			AuthMethod: &cloudbuildpb.WebhookConfig_Secret{
				Secret: webhookSecret,
			},
		}
	case "pubsub":
		t.PubsubConfig = &cloudbuildpb.PubsubConfig{
			Topic: pubSubTopic,
		}
	case "manual":
		// No trigger source required.
	default:
		return nil, fmt.Errorf("unsupported pipeline type %q", pipelineType)
	}
	return t, nil
}
