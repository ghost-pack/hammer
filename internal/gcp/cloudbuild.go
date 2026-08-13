package gcp

import (
	"context"
	"fmt"
	"os"
	"time"

	cloudbuild "cloud.google.com/go/cloudbuild/apiv1"
	"cloud.google.com/go/cloudbuild/apiv1/v2/cloudbuildpb"
	"cloud.google.com/go/longrunning/autogen/longrunningpb"
	"github.com/ghost-pack/hammer/internal/observability/tracing"
	"github.com/googleapis/gax-go/v2"
	"go.opentelemetry.io/otel/attribute"
	otelCodes "go.opentelemetry.io/otel/codes"
	"go.opentelemetry.io/otel/trace"
	"google.golang.org/api/option"
	"gopkg.in/yaml.v3"
)

type CloudBuildClient interface {
	TestCloudBuild(ctx context.Context, projectID, location, cloudbuildPath, cloudBuildTestPath string) error
	CreateOrUpdateCloudBuildTrigger(ctx context.Context, projectId, projectNumber, location, cloudBuildPath, triggerType, triggerName, pubsubTopic, serviceAccount string, manuallyApproved bool) error
	Close() error
}

type CloudBuildClientImpl struct {
	client cloudBuildApi
}

type cloudBuildApi interface {
	CreateBuild(ctx context.Context, req *cloudbuildpb.CreateBuildRequest, opts ...gax.CallOption) (*longrunningpb.Operation, error)
	GetBuild(ctx context.Context, req *cloudbuildpb.GetBuildRequest, opts ...gax.CallOption) (*cloudbuildpb.Build, error)
	ListBuildTriggers(ctx context.Context, req *cloudbuildpb.ListBuildTriggersRequest, opts ...gax.CallOption) (*cloudbuildpb.ListBuildTriggersResponse, error)
	CreateBuildTrigger(ctx context.Context, req *cloudbuildpb.CreateBuildTriggerRequest, opts ...gax.CallOption) (*cloudbuildpb.BuildTrigger, error)
	UpdateBuildTrigger(ctx context.Context, req *cloudbuildpb.UpdateBuildTriggerRequest, opts ...gax.CallOption) (*cloudbuildpb.BuildTrigger, error)
	// TODO: Probably get delete trigger too, for reconciliation purposes.
	Close() error
}

type cloudBuildAdapter struct {
	client *cloudbuild.Client
}

func (a *cloudBuildAdapter) CreateBuild(ctx context.Context, req *cloudbuildpb.CreateBuildRequest, opts ...gax.CallOption) (*longrunningpb.Operation, error) {
	return a.client.CreateBuild(ctx, req, opts...)
}

func (a *cloudBuildAdapter) GetBuild(ctx context.Context, req *cloudbuildpb.GetBuildRequest, opts ...gax.CallOption) (*cloudbuildpb.Build, error) {
	return a.client.GetBuild(ctx, req, opts...)
}

func (a *cloudBuildAdapter) ListBuildTriggers(ctx context.Context, req *cloudbuildpb.ListBuildTriggersRequest, opts ...gax.CallOption) (*cloudbuildpb.ListBuildTriggersResponse, error) {
	return a.client.ListBuildTriggers(ctx, req, opts...)
}

func (a *cloudBuildAdapter) CreateBuildTrigger(ctx context.Context, req *cloudbuildpb.CreateBuildTriggerRequest, opts ...gax.CallOption) (*cloudbuildpb.BuildTrigger, error) {
	return a.client.CreateBuildTrigger(ctx, req, opts...)
}

func (a *cloudBuildAdapter) UpdateBuildTrigger(ctx context.Context, req *cloudbuildpb.UpdateBuildTriggerRequest, opts ...gax.CallOption) (*cloudbuildpb.BuildTrigger, error) {
	return a.client.UpdateBuildTrigger(ctx, req, opts...)
}

func (a *cloudBuildAdapter) Close() error {
	return a.client.Close()
}

func NewCloudBuildClient(ctx context.Context, opts ...option.ClientOption) (*CloudBuildClientImpl, error) {
	client, err := cloudbuild.NewClient(ctx, opts...)
	if err != nil {
		return nil, fmt.Errorf("creating cloud build client: %w", err)
	}
	return &CloudBuildClientImpl{
		client: &cloudBuildAdapter{client: client},
	}, nil
}

func newCloudBuildClientWithAPI(api cloudBuildApi) *CloudBuildClientImpl {
	return &CloudBuildClientImpl{client: api}
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
		SecretEnv  []string `yaml:"secretEnv,omitempty"`
	} `yaml:"steps"`
	Substitutions    map[string]string `yaml:"substitutions,omitempty"`
	AvailableSecrets *secrets          `yaml:"availableSecrets,omitempty"`
}

type secrets struct {
	SecretManager []secretManagerSecret `yaml:"secretManager"`
}

type secretManagerSecret struct {
	VersionName string `yaml:"versionName"`
	Env         string `yaml:"env"`
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
			SecretEnv:  s.SecretEnv,
		})
	}

	return steps
}

func availableSecrets(cfg *cloudBuildConfig) *cloudbuildpb.Secrets {
	secretReferences := &cloudbuildpb.Secrets{}

	if cfg.AvailableSecrets == nil {
		return secretReferences
	}
	for _, s := range cfg.AvailableSecrets.SecretManager {
		secretReferences.SecretManager = append(secretReferences.SecretManager, &cloudbuildpb.SecretManagerSecret{
			VersionName: s.VersionName,
			Env:         s.Env,
		})
	}

	return secretReferences
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
	ctx, span := tracing.Tracer("gcloud builds submit test").Start(ctx, "gcloud builds submit test",
		trace.WithAttributes(
			attribute.String("cmd", "gcloud"),
			attribute.StringSlice("args", []string{"builds", "submit"})))
	defer span.End()

	buildConfig, err := parseCloudBuild(cloudBuildPath)
	if err != nil {
		span.RecordError(err)
		span.SetStatus(otelCodes.Error, err.Error())
		return err
	}

	testConfig, err := parseCloudBuildTest(cloudBuildTestPath)
	if err != nil {
		span.RecordError(err)
		span.SetStatus(otelCodes.Error, err.Error())
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
		buildSubmissionError := fmt.Errorf("submitting build: %w", err)
		span.RecordError(buildSubmissionError)
		span.SetStatus(otelCodes.Error, buildSubmissionError.Error())
		return buildSubmissionError
	}

	err = c.waitForBuild(ctx, projectID, location, op)
	if err != nil {
		span.RecordError(err)
		span.SetStatus(otelCodes.Error, err.Error())
		return err
	}
	span.SetStatus(otelCodes.Ok, "")
	return nil
}

func (c *CloudBuildClientImpl) CreateOrUpdateCloudBuildTrigger(
	ctx context.Context,
	projectID,
	projectNumber,
	location,
	cloudBuildPath,
	triggerType,
	triggerName,
	pubsubTopic,
	serviceAccount string,
	manuallyApproved bool,
) error {
	ctx, span := tracing.Tracer("gcloud builds triggers").Start(ctx, "gcloud builds triggers",
		trace.WithAttributes(
			attribute.String("cmd", "gcloud"),
			attribute.StringSlice("args", []string{"builds", "triggers"})))
	defer span.End()
	buildConfig, err := parseCloudBuild(cloudBuildPath)
	if err != nil {
		span.RecordError(err)
		span.SetStatus(otelCodes.Error, err.Error())
		return err
	}

	trigger, err := createBuildTrigger(
		projectID,
		projectNumber,
		triggerName,
		triggerType,
		pubsubTopic,
		serviceAccount,
		manuallyApproved,
		buildConfig,
	)
	if err != nil {
		span.RecordError(err)
		span.SetStatus(otelCodes.Error, err.Error())
		return err
	}

	existingTrigger, err := c.findTrigger(ctx, projectID, triggerName)
	if err != nil {
		span.RecordError(err)
		span.SetStatus(otelCodes.Error, err.Error())
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

	err = c.updateTrigger(
		ctx,
		projectID,
		existingTrigger.Id,
		trigger,
	)
	if err != nil {
		span.RecordError(err)
		span.SetStatus(otelCodes.Error, err.Error())
		return err
	}
	span.SetStatus(otelCodes.Ok, "")
	return nil
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
	triggerName,
	triggerType,
	pubsubTopic,
	serviceAccount string,
	manuallyApproved bool,
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
		serviceAccount,
	)

	build := &cloudbuildpb.Build{
		Steps:          buildSteps(cfg),
		Substitutions:  cfg.Substitutions,
		ServiceAccount: serviceAccountName,
		Approval: &cloudbuildpb.BuildApproval{
			Config: &cloudbuildpb.ApprovalConfig{
				ApprovalRequired: manuallyApproved,
			},
		},
		Options: &cloudbuildpb.BuildOptions{
			SubstitutionOption: cloudbuildpb.BuildOptions_ALLOW_LOOSE,
			Logging:            cloudbuildpb.BuildOptions_CLOUD_LOGGING_ONLY,
		},
		AvailableSecrets: availableSecrets(cfg),
	}

	return makeTrigger(
		triggerName,
		triggerType,
		secretResourceName,
		pubsubTopic,
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
	name, triggerType, webhookSecret, pubSubTopic string,
	build *cloudbuildpb.Build,
	subs map[string]string,
) (*cloudbuildpb.BuildTrigger, error) {
	t := &cloudbuildpb.BuildTrigger{
		Name:          name,
		Substitutions: subs,
		BuildTemplate: &cloudbuildpb.BuildTrigger_Build{Build: build},
	}

	switch triggerType {
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
		return nil, fmt.Errorf("unsupported ci type %q", triggerType)
	}
	return t, nil
}
