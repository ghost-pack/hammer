package project

import (
	"context"
	"fmt"
	"log/slog"

	iam "cloud.google.com/go/iam/admin/apiv1"
	adminpb "cloud.google.com/go/iam/admin/apiv1/adminpb"
	iampb "cloud.google.com/go/iam/apiv1/iampb"
	resourcemanager "cloud.google.com/go/resourcemanager/apiv3"
	"cloud.google.com/go/resourcemanager/apiv3/resourcemanagerpb"
	"github.com/ghost-pack/hammer/internal/observability/tracing"
	"github.com/googleapis/gax-go/v2"
	"go.opentelemetry.io/otel/attribute"
	otelCodes "go.opentelemetry.io/otel/codes"
	"go.opentelemetry.io/otel/trace"
	"google.golang.org/api/option"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

type IAMClient interface {
	EnsureServiceAccountExists(ctx context.Context, projectID, name, displayName string) (string, error)
	BindProjectRoles(ctx context.Context, projectID, saEmail string, roles []string) error
	UnbindProjectRoles(ctx context.Context, projectID, saEmail string, roles []string) error
	BindOrgRoles(ctx context.Context, orgID, saEmail string, roles []string) error
	UnbindOrgRoles(ctx context.Context, orgID, saEmail string, roles []string) error
	Close() error
}

type iamAPI interface {
	GetServiceAccount(ctx context.Context, req *adminpb.GetServiceAccountRequest, opts ...gax.CallOption) (*adminpb.ServiceAccount, error)
	CreateServiceAccount(ctx context.Context, req *adminpb.CreateServiceAccountRequest, opts ...gax.CallOption) (*adminpb.ServiceAccount, error)
	Close() error
}

type iamAdapter struct {
	client *iam.IamClient
}

func (a *iamAdapter) GetServiceAccount(ctx context.Context, req *adminpb.GetServiceAccountRequest, opts ...gax.CallOption) (*adminpb.ServiceAccount, error) {
	return a.client.GetServiceAccount(ctx, req, opts...)
}

func (a *iamAdapter) CreateServiceAccount(ctx context.Context, req *adminpb.CreateServiceAccountRequest, opts ...gax.CallOption) (*adminpb.ServiceAccount, error) {
	return a.client.CreateServiceAccount(ctx, req, opts...)
}

func (a *iamAdapter) Close() error {
	return a.client.Close()
}

type createProjectOperation interface {
	Wait(ctx context.Context, opts ...gax.CallOption) (*resourcemanagerpb.Project, error)
}

type projectsAPI interface {
	GetIamPolicy(ctx context.Context, req *iampb.GetIamPolicyRequest, opts ...gax.CallOption) (*iampb.Policy, error)
	SetIamPolicy(ctx context.Context, req *iampb.SetIamPolicyRequest, opts ...gax.CallOption) (*iampb.Policy, error)
	GetProject(ctx context.Context, req *resourcemanagerpb.GetProjectRequest, opts ...gax.CallOption) (*resourcemanagerpb.Project, error)
	CreateProject(ctx context.Context, req *resourcemanagerpb.CreateProjectRequest, opts ...gax.CallOption) (createProjectOperation, error) // ← changed
	Close() error
}

type projectsAdapter struct {
	client *resourcemanager.ProjectsClient
}

func (a *projectsAdapter) GetIamPolicy(ctx context.Context, req *iampb.GetIamPolicyRequest, opts ...gax.CallOption) (*iampb.Policy, error) {
	return a.client.GetIamPolicy(ctx, req, opts...)
}

func (a *projectsAdapter) SetIamPolicy(ctx context.Context, req *iampb.SetIamPolicyRequest, opts ...gax.CallOption) (*iampb.Policy, error) {
	return a.client.SetIamPolicy(ctx, req, opts...)
}

func (a *projectsAdapter) GetProject(ctx context.Context, req *resourcemanagerpb.GetProjectRequest, opts ...gax.CallOption) (*resourcemanagerpb.Project, error) {
	return a.client.GetProject(ctx, req, opts...)
}

func (a *projectsAdapter) CreateProject(ctx context.Context, req *resourcemanagerpb.CreateProjectRequest, opts ...gax.CallOption) (createProjectOperation, error) {
	return a.client.CreateProject(ctx, req, opts...)
}

func (a *projectsAdapter) Close() error {
	return a.client.Close()
}

type organizationAPI interface {
	GetIamPolicy(ctx context.Context, req *iampb.GetIamPolicyRequest, opts ...gax.CallOption) (*iampb.Policy, error)
	SetIamPolicy(ctx context.Context, req *iampb.SetIamPolicyRequest, opts ...gax.CallOption) (*iampb.Policy, error)
	Close() error
}

type organizationAdapter struct {
	client *resourcemanager.OrganizationsClient
}

func (a *organizationAdapter) GetIamPolicy(ctx context.Context, req *iampb.GetIamPolicyRequest, opts ...gax.CallOption) (*iampb.Policy, error) {
	return a.client.GetIamPolicy(ctx, req, opts...)
}

func (a *organizationAdapter) SetIamPolicy(ctx context.Context, req *iampb.SetIamPolicyRequest, opts ...gax.CallOption) (*iampb.Policy, error) {
	return a.client.SetIamPolicy(ctx, req, opts...)
}

func (a *organizationAdapter) Close() error {
	return a.client.Close()
}

type IAMClientImpl struct {
	iam          iamAPI
	projects     projectsAPI
	organization organizationAPI
}

func NewIAMClient(ctx context.Context, opts ...option.ClientOption) (*IAMClientImpl, error) {
	iamClient, err := iam.NewIamClient(ctx, opts...)
	if err != nil {
		return nil, fmt.Errorf("creating iam client: %w", err)
	}
	projectsClient, err := resourcemanager.NewProjectsClient(ctx, opts...)
	if err != nil {
		iamClient.Close()
		return nil, fmt.Errorf("creating projects client for iam: %w", err)
	}
	orgsClient, err := resourcemanager.NewOrganizationsClient(ctx, opts...)
	if err != nil {
		iamClient.Close()
		projectsClient.Close()
		return nil, fmt.Errorf("creating organizations client for iam: %w", err)
	}
	return &IAMClientImpl{
		iam:          &iamAdapter{client: iamClient},
		projects:     &projectsAdapter{client: projectsClient},
		organization: &organizationAdapter{client: orgsClient},
	}, nil
}

func newIamClientWithAPI(api iamAPI, projects projectsAPI, org organizationAPI) *IAMClientImpl {
	return &IAMClientImpl{iam: api, projects: projects, organization: org}
}

func (c *IAMClientImpl) Close() error {
	if err := c.iam.Close(); err != nil {
		return err
	}
	if err := c.projects.Close(); err != nil {
		return err
	}
	return c.organization.Close()
}

// EnsureServiceAccountExists creates a SA if it doesn't exist, returns its email
func (c *IAMClientImpl) EnsureServiceAccountExists(ctx context.Context, projectID, name, displayName string) (string, error) {
	ctx, span := tracing.Tracer("ensure service account exists").Start(ctx, "ensure service account",
		trace.WithAttributes(
			attribute.String("project", projectID),
			attribute.String("serviceAccountDisplayName", displayName)))
	defer span.End()

	resource := fmt.Sprintf("projects/%s/serviceAccounts/%s@%s.iam.gserviceaccount.com", projectID, name, projectID)

	sa, err := c.iam.GetServiceAccount(ctx, &adminpb.GetServiceAccountRequest{Name: resource})
	if err == nil {
		slog.InfoContext(ctx, "service account already exists", "email", sa.Email)
		span.SetStatus(otelCodes.Ok, "service account already exists")
		return sa.Email, nil
	}
	if status.Code(err) != codes.NotFound {
		span.RecordError(err)
		span.SetStatus(otelCodes.Error, err.Error())
		return "", fmt.Errorf("checking service account %s: %w", name, err)
	}

	sa, err = c.iam.CreateServiceAccount(ctx, &adminpb.CreateServiceAccountRequest{
		Name:      "projects/" + projectID,
		AccountId: name,
		ServiceAccount: &adminpb.ServiceAccount{
			DisplayName: displayName,
		},
	})
	if err != nil {
		span.RecordError(err)
		span.SetStatus(otelCodes.Error, err.Error())
		return "", fmt.Errorf("creating service account %s: %w", name, err)
	}
	slog.InfoContext(ctx, "service account created", "email", sa.Email)
	span.SetStatus(otelCodes.Ok, "service account created")
	return sa.Email, nil
}

func (c *IAMClientImpl) BindProjectRoles(ctx context.Context, projectID, saEmail string, roles []string) error {
	ctx, span := tracing.Tracer("bind service account roles").Start(ctx, "bind service account roles",
		trace.WithAttributes(
			attribute.String("project", projectID),
			attribute.String("serviceAccountEmail", saEmail)))
	defer span.End()

	projectName := "projects/" + projectID
	member := "serviceAccount:" + saEmail

	policy, err := c.projects.GetIamPolicy(ctx, &iampb.GetIamPolicyRequest{
		Resource: projectName,
	})
	if err != nil {
		span.RecordError(err)
		span.SetStatus(otelCodes.Error, err.Error())
		return fmt.Errorf("getting IAM policy for %s: %w", projectID, err)
	}

	for _, role := range roles {
		addBinding(policy, role, member)
	}

	_, err = c.projects.SetIamPolicy(ctx, &iampb.SetIamPolicyRequest{
		Resource: projectName,
		Policy:   policy,
	})
	if err != nil {
		span.RecordError(err)
		span.SetStatus(otelCodes.Error, err.Error())
		return fmt.Errorf("setting IAM policy for %s: %w", projectID, err)
	}
	slog.InfoContext(ctx, "roles bound", "projectID", projectID, "sa", saEmail, "roles", roles)
	span.SetStatus(otelCodes.Ok, "roles bound")
	return nil
}

func (c *IAMClientImpl) BindOrgRoles(ctx context.Context, orgID, saEmail string, roles []string) error {
	ctx, span := tracing.Tracer("bind org roles").Start(ctx, "bind org roles",
		trace.WithAttributes(
			attribute.String("org", orgID),
			attribute.String("serviceAccountEmail", saEmail)))
	defer span.End()

	resource := "organizations/" + orgID
	member := "serviceAccount:" + saEmail

	policy, err := c.organization.GetIamPolicy(ctx, &iampb.GetIamPolicyRequest{Resource: resource})
	if err != nil {
		span.RecordError(err)
		span.SetStatus(otelCodes.Error, err.Error())
		return fmt.Errorf("getting org IAM policy for %s: %w", orgID, err)
	}

	for _, role := range roles {
		addBinding(policy, role, member)
	}

	_, err = c.organization.SetIamPolicy(ctx, &iampb.SetIamPolicyRequest{
		Resource: resource,
		Policy:   policy,
	})
	if err != nil {
		span.RecordError(err)
		span.SetStatus(otelCodes.Error, err.Error())
		return fmt.Errorf("setting org IAM policy for %s: %w", orgID, err)
	}

	slog.InfoContext(ctx, "org roles bound", "orgID", orgID, "sa", saEmail, "roles", roles)
	span.SetStatus(otelCodes.Ok, "org roles bound")
	return nil
}

func addBinding(policy *iampb.Policy, role, member string) {
	for _, binding := range policy.Bindings {
		if binding.Role == role {
			for _, m := range binding.Members {
				if m == member {
					return
				}
			}
			binding.Members = append(binding.Members, member)
			return
		}
	}
	policy.Bindings = append(policy.Bindings, &iampb.Binding{
		Role:    role,
		Members: []string{member},
	})
}

func (c *IAMClientImpl) UnbindProjectRoles(ctx context.Context, projectID, saEmail string, roles []string) error {
	ctx, span := tracing.Tracer("unbind service account roles").Start(ctx, "unbind service account roles",
		trace.WithAttributes(
			attribute.String("project", projectID),
			attribute.String("serviceAccountEmail", saEmail)))
	defer span.End()

	projectName := "projects/" + projectID
	member := "serviceAccount:" + saEmail

	policy, err := c.projects.GetIamPolicy(ctx, &iampb.GetIamPolicyRequest{
		Resource: projectName,
	})
	if err != nil {
		span.RecordError(err)
		span.SetStatus(otelCodes.Error, err.Error())
		return fmt.Errorf("getting IAM policy for %s: %w", projectID, err)
	}

	for _, role := range roles {
		removeBinding(policy, role, member)
	}

	_, err = c.projects.SetIamPolicy(ctx, &iampb.SetIamPolicyRequest{
		Resource: projectName,
		Policy:   policy,
	})
	if err != nil {
		span.RecordError(err)
		span.SetStatus(otelCodes.Error, err.Error())
		return fmt.Errorf("removing IAM roles for %s: %w", projectID, err)
	}
	slog.InfoContext(ctx, "roles unbound", "projectID", projectID, "sa", saEmail, "roles", roles)
	span.SetStatus(otelCodes.Ok, "roles unbound")
	return nil
}

func (c *IAMClientImpl) UnbindOrgRoles(ctx context.Context, orgID, saEmail string, roles []string) error {
	ctx, span := tracing.Tracer("unbind org roles").Start(ctx, "unbind org roles",
		trace.WithAttributes(
			attribute.String("org", orgID),
			attribute.String("serviceAccountEmail", saEmail)))
	defer span.End()

	resource := "organizations/" + orgID
	member := "serviceAccount:" + saEmail

	policy, err := c.organization.GetIamPolicy(ctx, &iampb.GetIamPolicyRequest{Resource: resource})
	if err != nil {
		span.RecordError(err)
		span.SetStatus(otelCodes.Error, err.Error())
		return fmt.Errorf("getting org IAM policy for %s: %w", orgID, err)
	}

	for _, role := range roles {
		removeBinding(policy, role, member)
	}

	_, err = c.organization.SetIamPolicy(ctx, &iampb.SetIamPolicyRequest{
		Resource: resource,
		Policy:   policy,
	})
	if err != nil {
		span.RecordError(err)
		span.SetStatus(otelCodes.Error, err.Error())
		return fmt.Errorf("removing org IAM roles for %s: %w", orgID, err)
	}

	slog.InfoContext(ctx, "org roles unbound", "orgID", orgID, "sa", saEmail, "roles", roles)
	span.SetStatus(otelCodes.Ok, "org roles unbound")
	return nil
}

func removeBinding(policy *iampb.Policy, role, member string) {
	for i, binding := range policy.Bindings {
		if binding.Role != role {
			continue
		}
		// remove the member from this binding
		members := make([]string, 0, len(binding.Members))
		for _, m := range binding.Members {
			if m != member {
				members = append(members, m)
			}
		}
		if len(members) == 0 {
			// no members left — remove the binding entirely
			policy.Bindings = append(policy.Bindings[:i], policy.Bindings[i+1:]...)
		} else {
			binding.Members = members
		}
		return
	}
}
