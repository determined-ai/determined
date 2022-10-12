package webhooks

import (
	"context"
	"net/url"

	"github.com/determined-ai/determined/proto/pkg/apiv1"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

// APIServer is an embedded api server struct.
type APIServer struct{}

// GetWebhooks returns all Webhooks.
func (a *APIServer) GetWebhooks(
	ctx context.Context, req *apiv1.GetWebhooksRequest,
) (*apiv1.GetWebhooksResponse, error) {
	webhooks, err := GetWebhooks(ctx)
	if err != nil {
		return nil, err
	}
	return &apiv1.GetWebhooksResponse{Webhooks: webhooks.Proto()}, nil
}

// PostWebhook creates a new Webhook.
func (a *APIServer) PostWebhook(
	ctx context.Context, req *apiv1.PostWebhookRequest,
) (*apiv1.PostWebhookResponse, error) {
	if len(req.Webhook.Triggers) == 0 {
		return nil, status.Errorf(
			codes.InvalidArgument,
			"at least one trigger required",
		)
	}
	if _, err := url.ParseRequestURI(req.Webhook.Url); err != nil {
		return nil, status.Errorf(
			codes.InvalidArgument,
			"valid url required",
		)
	}
	w := WebhookFromProto(req.Webhook)
	if err := AddWebhook(ctx, &w); err != nil {
		return nil, err
	}
	return &apiv1.PostWebhookResponse{Webhook: w.Proto()}, nil
}

// DeleteWebhook deletes a Webhook.
func (a *APIServer) DeleteWebhook(
	ctx context.Context, req *apiv1.DeleteWebhookRequest,
) (*apiv1.DeleteWebhookResponse, error) {
	if err := DeleteWebhook(ctx, WebhookID(req.Id)); err != nil {
		return nil, err
	}
	return &apiv1.DeleteWebhookResponse{}, nil
}

// TestWebhook sends a test event for a Webhook.
func (a *APIServer) TestWebhook(
	ctx context.Context, req *apiv1.TestWebhookRequest,
) (*apiv1.TestWebhookResponse, error) {
	panic("unimplemented")
}
