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
	"github.com/googleapis/gax-go/v2"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

type IAMClient interface {
	EnsureServiceAccountExists(ctx context.Context, projectID, name, displayName string) (string, error)
	BindProjectRoles(ctx context.Context, projectID, saEmail string, roles []string) error
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

type projectsAPI interface {
	GetIamPolicy(ctx context.Context, req *iampb.GetIamPolicyRequest, opts ...gax.CallOption) (*iampb.Policy, error)
	SetIamPolicy(ctx context.Context, req *iampb.SetIamPolicyRequest, opts ...gax.CallOption) (*iampb.Policy, error)
	GetProject(ctx context.Context, req *resourcemanagerpb.GetProjectRequest, opts ...gax.CallOption) (*resourcemanagerpb.Project, error)
	CreateProject(ctx context.Context, req *resourcemanagerpb.CreateProjectRequest, opts ...gax.CallOption) (*resourcemanager.CreateProjectOperation, error)
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

func (a *projectsAdapter) CreateProject(ctx context.Context, req *resourcemanagerpb.CreateProjectRequest, opts ...gax.CallOption) (*resourcemanager.CreateProjectOperation, error) {
	return a.client.CreateProject(ctx, req, opts...)
}

func (a *projectsAdapter) Close() error {
	return a.client.Close()
}

type IAMClientImpl struct {
	iam      iamAPI
	projects projectsAPI
}

func NewIAMClient(ctx context.Context) (*IAMClientImpl, error) {
	iamClient, err := iam.NewIamClient(ctx)
	if err != nil {
		return nil, fmt.Errorf("creating iam client: %w", err)
	}
	projectsClient, err := resourcemanager.NewProjectsClient(ctx)
	if err != nil {
		iamClient.Close()
		return nil, fmt.Errorf("creating projects client for iam: %w", err)
	}
	return &IAMClientImpl{iam: &iamAdapter{client: iamClient}, projects: projectsClient}, nil
}

func newIamClientWithAPI(api iamAPI, api2 projectsAPI) *IAMClientImpl {
	return &IAMClientImpl{iam: api, projects: api2}
}

func (c *IAMClientImpl) Close() error {
	if err := c.iam.Close(); err != nil {
		return err
	}
	return c.projects.Close()
}

// EnsureServiceAccountExists creates a SA if it doesn't exist, returns its email
func (c *IAMClientImpl) EnsureServiceAccountExists(ctx context.Context, projectID, name, displayName string) (string, error) {
	resource := fmt.Sprintf("projects/%s/serviceAccounts/%s@%s.iam.gserviceaccount.com", projectID, name, projectID)

	sa, err := c.iam.GetServiceAccount(ctx, &adminpb.GetServiceAccountRequest{Name: resource})
	if err == nil {
		slog.InfoContext(ctx, "service account already exists", "email", sa.Email)
		return sa.Email, nil
	}
	if status.Code(err) != codes.NotFound {
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
		return "", fmt.Errorf("creating service account %s: %w", name, err)
	}
	slog.InfoContext(ctx, "service account created", "email", sa.Email)
	return sa.Email, nil
}

func (c *IAMClientImpl) BindProjectRoles(ctx context.Context, projectID, saEmail string, roles []string) error {
	projectName := "projects/" + projectID
	member := "serviceAccount:" + saEmail

	policy, err := c.projects.GetIamPolicy(ctx, &iampb.GetIamPolicyRequest{
		Resource: projectName,
	})
	if err != nil {
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
		return fmt.Errorf("setting IAM policy for %s: %w", projectID, err)
	}
	slog.InfoContext(ctx, "roles bound", "projectID", projectID, "sa", saEmail, "roles", roles)
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
