package webhooks

import (
	"context"

	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"

	"github.com/determined-ai/determined/master/internal/grpcutil"

	"github.com/determined-ai/determined/proto/pkg/apiv1"
)

// WebhooksAPIServer is an embedded api server struct.
type WebhooksAPIServer struct{}

// AuthorizeRequest checks if the user has CanEditWebhooks permissions.
func AuthorizeRequest(ctx context.Context) error {
	curUser, _, err := grpcutil.GetUser(ctx)
	if err != nil {
		return status.Errorf(codes.Internal, "failed to get the user: %s", err)
	}
	authErr := AuthZProvider.Get().
		CanEditWebhooks(ctx, curUser)
	if authErr != nil {
		return status.Error(codes.PermissionDenied, authErr.Error())
	}
	return nil
}

// GetWebhooks returns all Webhooks.
func (a *WebhooksAPIServer) GetWebhooks(
	ctx context.Context, req *apiv1.GetWebhooksRequest,
) (*apiv1.GetWebhooksResponse, error) {
	err := AuthorizeRequest(ctx)
	if err != nil {
		return nil, err
	}
	webhooks, err := GetWebhooks(ctx)
	if err != nil {
		return nil, err
	}
	return &apiv1.GetWebhooksResponse{Webhooks: webhooks.Proto()}, nil
}

// PostWebhook creates a new Webhook.
func (a *WebhooksAPIServer) PostWebhook(
	ctx context.Context, req *apiv1.PostWebhookRequest,
) (*apiv1.PostWebhookResponse, error) {
	err := AuthorizeRequest(ctx)
	if err != nil {
		return nil, err
	}
	w := WebhookFromProto(req.Webhook)
	if err := AddWebhook(ctx, &w); err != nil {
		return nil, err
	}
	return &apiv1.PostWebhookResponse{Webhook: w.Proto()}, nil
}

// DeleteWebhook deletes a Webhook.
func (a *WebhooksAPIServer) DeleteWebhook(
	ctx context.Context, req *apiv1.DeleteWebhookRequest,
) (*apiv1.DeleteWebhookResponse, error) {
	err := AuthorizeRequest(ctx)
	if err != nil {
		return nil, err
	}
	if err := DeleteWebhook(ctx, WebhookID(req.Id)); err != nil {
		return nil, err
	}
	return &apiv1.DeleteWebhookResponse{}, nil
}

// TestWebhook sends a test event for a Webhook.
func (a *WebhooksAPIServer) TestWebhook(
	ctx context.Context, req *apiv1.TestWebhookRequest,
) (*apiv1.TestWebhookResponse, error) {
	err := AuthorizeRequest(ctx)
	if err != nil {
		return nil, err
	}
	panic("unimplemented")
}
