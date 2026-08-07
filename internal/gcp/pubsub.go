package gcp

import (
	"context"
	"fmt"
	"log/slog"

	"cloud.google.com/go/iam/apiv1/iampb"
	pubsubadmin "cloud.google.com/go/pubsub/apiv1"
	"cloud.google.com/go/pubsub/v2/apiv1/pubsubpb"
	"github.com/ghost-pack/hammer/internal/observability/tracing"
	"github.com/googleapis/gax-go/v2"
	"go.opentelemetry.io/otel/attribute"
	otelCodes "go.opentelemetry.io/otel/codes"
	"go.opentelemetry.io/otel/trace"
	"google.golang.org/api/option"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

type PubsubClient interface {
	EnsureTopic(ctx context.Context, projectID, topicID, publisherSA string) error
	DeleteTopic(ctx context.Context, projectID, topicID string) error
	PublishMessage(ctx context.Context, projectID, topicID string, data []byte, attributes map[string]string) (string, error)
	Close() error
}

type pubsubAPI interface {
	GetTopic(ctx context.Context, req *pubsubpb.GetTopicRequest, opts ...gax.CallOption) (*pubsubpb.Topic, error)
	CreateTopic(ctx context.Context, req *pubsubpb.Topic, opts ...gax.CallOption) (*pubsubpb.Topic, error)
	DeleteTopic(ctx context.Context, req *pubsubpb.DeleteTopicRequest, opts ...gax.CallOption) error
	Publish(ctx context.Context, req *pubsubpb.PublishRequest, opts ...gax.CallOption) (*pubsubpb.PublishResponse, error)
	GetIamPolicy(ctx context.Context, req *iampb.GetIamPolicyRequest, opts ...gax.CallOption) (*iampb.Policy, error)
	SetIamPolicy(ctx context.Context, req *iampb.SetIamPolicyRequest, opts ...gax.CallOption) (*iampb.Policy, error)
	Close() error
}

type pubsubAdapter struct {
	client *pubsubadmin.PublisherClient
}

func (a *pubsubAdapter) GetTopic(ctx context.Context, req *pubsubpb.GetTopicRequest, opts ...gax.CallOption) (*pubsubpb.Topic, error) {
	return a.client.GetTopic(ctx, req, opts...)
}

func (a *pubsubAdapter) CreateTopic(ctx context.Context, req *pubsubpb.Topic, opts ...gax.CallOption) (*pubsubpb.Topic, error) {
	return a.client.CreateTopic(ctx, req, opts...)
}

func (a *pubsubAdapter) DeleteTopic(ctx context.Context, req *pubsubpb.DeleteTopicRequest, opts ...gax.CallOption) error {
	return a.client.DeleteTopic(ctx, req, opts...)
}

func (a *pubsubAdapter) Publish(ctx context.Context, req *pubsubpb.PublishRequest, opts ...gax.CallOption) (*pubsubpb.PublishResponse, error) {
	return a.client.Publish(ctx, req, opts...)
}

func (a *pubsubAdapter) GetIamPolicy(ctx context.Context, req *iampb.GetIamPolicyRequest, opts ...gax.CallOption) (*iampb.Policy, error) {
	return a.client.GetIamPolicy(ctx, req, opts...)
}

func (a *pubsubAdapter) SetIamPolicy(ctx context.Context, req *iampb.SetIamPolicyRequest, opts ...gax.CallOption) (*iampb.Policy, error) {
	return a.client.SetIamPolicy(ctx, req, opts...)
}

func (a *pubsubAdapter) Close() error {
	return a.client.Close()
}

type PubsubClientImpl struct {
	client pubsubAPI
}

func NewPubsubClient(ctx context.Context, opts ...option.ClientOption) (*PubsubClientImpl, error) {
	client, err := pubsubadmin.NewPublisherClient(ctx, opts...)
	if err != nil {
		return nil, fmt.Errorf("creating pubsub client: %w", err)
	}
	return &PubsubClientImpl{client: &pubsubAdapter{client: client}}, nil
}

func newPubsubClientWithAPI(api pubsubAPI) *PubsubClientImpl {
	return &PubsubClientImpl{client: api}
}

func (p *PubsubClientImpl) Close() error {
	return p.client.Close()
}

func (p *PubsubClientImpl) EnsureTopic(ctx context.Context, projectID, topicID, publisherSA string) error {
	ctx, span := tracing.Tracer("ensure pubsub topic").Start(ctx, "ensure pubsub topic",
		trace.WithAttributes(
			attribute.String("projectID", projectID),
			attribute.String("topicID", topicID)))
	defer span.End()

	topicName := fmt.Sprintf("projects/%s/topics/%s", projectID, topicID)

	_, err := p.client.GetTopic(ctx, &pubsubpb.GetTopicRequest{Topic: topicName})
	if err != nil && status.Code(err) != codes.NotFound && status.Code(err) != codes.AlreadyExists {
		span.RecordError(err)
		span.SetStatus(otelCodes.Error, err.Error())
		return fmt.Errorf("checking topic %q: %w", topicName, err)
	}

	if status.Code(err) == codes.NotFound {
		_, err = p.client.CreateTopic(ctx, &pubsubpb.Topic{Name: topicName})
		if err != nil && status.Code(err) != codes.AlreadyExists {
			span.RecordError(err)
			span.SetStatus(otelCodes.Error, err.Error())
			return fmt.Errorf("creating topic %q: %w", topicName, err)
		}
		slog.InfoContext(ctx, "topic created", "topic", topicName)
	} else {
		slog.InfoContext(ctx, "topic already exists", "topic", topicName)
	}

	policy, err := p.client.GetIamPolicy(ctx, &iampb.GetIamPolicyRequest{Resource: topicName})
	if err != nil {
		span.RecordError(err)
		span.SetStatus(otelCodes.Error, err.Error())
		return fmt.Errorf("getting IAM policy for topic %q: %w", topicName, err)
	}

	policy.Bindings = []*iampb.Binding{
		{
			Role:    "roles/pubsub.publisher",
			Members: []string{"serviceAccount:" + publisherSA},
		},
	}

	_, err = p.client.SetIamPolicy(ctx, &iampb.SetIamPolicyRequest{
		Resource: topicName,
		Policy:   policy,
	})
	if err != nil {
		span.RecordError(err)
		span.SetStatus(otelCodes.Error, err.Error())
		return fmt.Errorf("setting IAM policy on topic %q: %w", topicName, err)
	}

	slog.InfoContext(ctx, "topic IAM set", "topic", topicName, "publisher", publisherSA)
	span.SetStatus(otelCodes.Ok, "topic ensured")
	return nil
}

func (p *PubsubClientImpl) DeleteTopic(ctx context.Context, projectID, topicID string) error {
	ctx, span := tracing.Tracer("delete pubsub topic").Start(ctx, "delete pubsub topic",
		trace.WithAttributes(
			attribute.String("projectID", projectID),
			attribute.String("topicID", topicID)))
	defer span.End()

	topicName := fmt.Sprintf("projects/%s/topics/%s", projectID, topicID)

	err := p.client.DeleteTopic(ctx, &pubsubpb.DeleteTopicRequest{Topic: topicName})
	if err != nil {
		if status.Code(err) == codes.NotFound {
			slog.InfoContext(ctx, "topic already deleted", "topic", topicName)
			span.SetStatus(otelCodes.Ok, "topic not found")
			return nil
		}
		span.RecordError(err)
		span.SetStatus(otelCodes.Error, err.Error())
		return fmt.Errorf("deleting topic %q: %w", topicName, err)
	}

	slog.InfoContext(ctx, "topic deleted", "topic", topicName)
	span.SetStatus(otelCodes.Ok, "topic deleted")
	return nil
}

// PublishMessage publishes a message to the given topic and returns the server-assigned message ID.
func (p *PubsubClientImpl) PublishMessage(ctx context.Context, projectID, topicID string, data []byte, attributes map[string]string) (string, error) {
	ctx, span := tracing.Tracer("publish pubsub message").Start(ctx, "publish pubsub message",
		trace.WithAttributes(
			attribute.String("projectID", projectID),
			attribute.String("topicID", topicID)))
	defer span.End()

	topicName := fmt.Sprintf("projects/%s/topics/%s", projectID, topicID)

	resp, err := p.client.Publish(ctx, &pubsubpb.PublishRequest{
		Topic: topicName,
		Messages: []*pubsubpb.PubsubMessage{
			{
				Data:       data,
				Attributes: attributes,
			},
		},
	})
	if err != nil {
		span.RecordError(err)
		span.SetStatus(otelCodes.Error, err.Error())
		return "", fmt.Errorf("publishing message to topic %q: %w", topicName, err)
	}

	messageID := resp.MessageIds[0]
	slog.InfoContext(ctx, "message published", "topic", topicName, "messageID", messageID)
	span.SetStatus(otelCodes.Ok, "message published")
	return messageID, nil
}
