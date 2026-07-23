package project

import (
	"context"
	"fmt"
	"log/slog"

	billing "cloud.google.com/go/billing/apiv1"
	billingpb "cloud.google.com/go/billing/apiv1/billingpb"
	"github.com/googleapis/gax-go/v2"
	"google.golang.org/api/option"
)

type billingAPI interface {
	UpdateProjectBillingInfo(ctx context.Context, req *billingpb.UpdateProjectBillingInfoRequest, opts ...gax.CallOption) (*billingpb.ProjectBillingInfo, error)
	Close() error
}
type billingAdapter struct {
	client *billing.CloudBillingClient
}

func (a *billingAdapter) UpdateProjectBillingInfo(ctx context.Context, req *billingpb.UpdateProjectBillingInfoRequest, opts ...gax.CallOption) (*billingpb.ProjectBillingInfo, error) {
	return a.client.UpdateProjectBillingInfo(ctx, req, opts...)
}

func (a *billingAdapter) Close() error {
	return a.client.Close()
}

type BillingClient interface {
	LinkBillingAccount(ctx context.Context, projectID, billingAccount string) error
	Close() error
}

type BillingClientImpl struct {
	client billingAPI
}

func NewBillingClient(ctx context.Context, opts ...option.ClientOption) (*BillingClientImpl, error) {
	client, err := billing.NewCloudBillingClient(ctx, opts...)
	if err != nil {
		return nil, fmt.Errorf("creating billing client: %w", err)
	}
	return &BillingClientImpl{client: &billingAdapter{client: client}}, nil
}

func newBillingClientWithAPI(api billingAPI) *BillingClientImpl {
	return &BillingClientImpl{client: api}
}

func (c *BillingClientImpl) Close() error {
	return c.client.Close()
}

func (c *BillingClientImpl) LinkBillingAccount(ctx context.Context, projectID, billingAccount string) error {
	_, err := c.client.UpdateProjectBillingInfo(ctx, &billingpb.UpdateProjectBillingInfoRequest{
		Name: "projects/" + projectID,
		ProjectBillingInfo: &billingpb.ProjectBillingInfo{
			BillingAccountName: "billingAccounts/" + billingAccount,
		},
	})
	if err != nil {
		return fmt.Errorf("linking billing account to %s: %w", projectID, err)
	}
	slog.InfoContext(ctx, "billing account linked", "projectID", projectID, "billingAccount", billingAccount)
	return nil
}
