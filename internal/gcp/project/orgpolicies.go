package project

import (
	"context"
	"fmt"
	"log/slog"

	orgpolicy "cloud.google.com/go/orgpolicy/apiv2"
	orgpolicypb "cloud.google.com/go/orgpolicy/apiv2/orgpolicypb"
	"github.com/googleapis/gax-go/v2"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

type orgPolicyAPI interface {
	GetPolicy(ctx context.Context, req *orgpolicypb.GetPolicyRequest, opts ...gax.CallOption) (*orgpolicypb.Policy, error)
	CreatePolicy(ctx context.Context, req *orgpolicypb.CreatePolicyRequest, opts ...gax.CallOption) (*orgpolicypb.Policy, error)
	UpdatePolicy(ctx context.Context, req *orgpolicypb.UpdatePolicyRequest, opts ...gax.CallOption) (*orgpolicypb.Policy, error)
	Close() error
}
type orgPolicyAdapter struct {
	client *orgpolicy.Client
}

func (a *orgPolicyAdapter) GetPolicy(ctx context.Context, req *orgpolicypb.GetPolicyRequest, opts ...gax.CallOption) (*orgpolicypb.Policy, error) {
	return a.client.GetPolicy(ctx, req, opts...)
}

func (a *orgPolicyAdapter) CreatePolicy(ctx context.Context, req *orgpolicypb.CreatePolicyRequest, opts ...gax.CallOption) (*orgpolicypb.Policy, error) {
	return a.client.CreatePolicy(ctx, req, opts...)
}

func (a *orgPolicyAdapter) UpdatePolicy(ctx context.Context, req *orgpolicypb.UpdatePolicyRequest, opts ...gax.CallOption) (*orgpolicypb.Policy, error) {
	return a.client.UpdatePolicy(ctx, req, opts...)
}

func (a *orgPolicyAdapter) Close() error {
	return a.client.Close()
}

type OrgPolicyClient interface {
	EnforcePolicy(ctx context.Context, resource, constraint string) error
	Close() error
}

type OrgPolicyClientImpl struct {
	client orgPolicyAPI
}

func NewOrgPolicyClient(ctx context.Context) (*OrgPolicyClientImpl, error) {
	client, err := orgpolicy.NewClient(ctx)
	if err != nil {
		return nil, fmt.Errorf("creating org policy client: %w", err)
	}
	return &OrgPolicyClientImpl{client: &orgPolicyAdapter{client: client}}, nil
}

func newOrgPolicyClientWithAPI(api orgPolicyAPI) *OrgPolicyClientImpl {
	return &OrgPolicyClientImpl{client: api}
}

func (c *OrgPolicyClientImpl) Close() error {
	return c.client.Close()
}

// EnforcePolicy enforces a boolean constraint on a resource.
// resource is e.g. "projects/my-project-id" or "folders/123456"
// constraint is e.g. "constraints/iam.disableServiceAccountKeyCreation"
func (c *OrgPolicyClientImpl) EnforcePolicy(ctx context.Context, resource, constraint string) error {
	name := fmt.Sprintf("%s/policies/%s", resource, constraint)

	existing, err := c.client.GetPolicy(ctx, &orgpolicypb.GetPolicyRequest{Name: name})
	if err != nil && status.Code(err) != codes.NotFound {
		return fmt.Errorf("checking policy %s: %w", constraint, err)
	}

	policy := &orgpolicypb.Policy{
		Name: name,
		Spec: &orgpolicypb.PolicySpec{
			Rules: []*orgpolicypb.PolicySpec_PolicyRule{
				{
					Kind: &orgpolicypb.PolicySpec_PolicyRule_Enforce{
						Enforce: true,
					},
				},
			},
		},
	}

	if status.Code(err) == codes.NotFound {
		_, err = c.client.CreatePolicy(ctx, &orgpolicypb.CreatePolicyRequest{
			Parent: resource,
			Policy: policy,
		})
	} else {
		// etag must be passed back to avoid overwriting concurrent changes
		policy.Etag = existing.Etag
		_, err = c.client.UpdatePolicy(ctx, &orgpolicypb.UpdatePolicyRequest{
			Policy: policy,
		})
	}
	if err != nil {
		return fmt.Errorf("enforcing policy %s on %s: %w", constraint, resource, err)
	}

	slog.InfoContext(ctx, "org policy enforced", "resource", resource, "constraint", constraint)
	return nil
}
