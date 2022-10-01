package webhooks

import (
	"context"

	"github.com/determined-ai/determined/proto/pkg/apiv1"
)

// WebhooksAPIServer is an embedded api server struct.
type WebhooksAPIServer struct{}

func (a *WebhooksAPIServer) GetWebhooks(
	ctx context.Context, req *apiv1.GetWebhooksRequest,
) (*apiv1.GetWebhooksResponse, error) {
	webhooks, err := GetWebhooks(ctx)
	if err != nil {
		return nil, err
	}
	return &apiv1.GetWebhooksResponse{Webhooks: webhooks.Proto()}, nil
}

func (a *WebhooksAPIServer) PostWebhook(
	ctx context.Context, req *apiv1.PostWebhookRequest,
) (*apiv1.PostWebhookResponse, error) {
	w := WebhookFromProto(req.Webhook)
	if err := AddWebhook(ctx, w); err != nil {
		return nil, err
	}
	return &apiv1.PostWebhookResponse{Webhook: w.Proto()}, nil
}

func (a *WebhooksAPIServer) DeleteWebhook(
	ctx context.Context, req *apiv1.DeleteWebhookRequest,
) (*apiv1.DeleteWebhookResponse, error) {
	if err := DeleteWebhook(ctx, WebhookID(req.Id)); err != nil {
		return nil, err
	}
	return &apiv1.DeleteWebhookResponse{}, nil
}

func (a *WebhooksAPIServer) TestWebhook(
	ctx context.Context, req *apiv1.TestWebhookRequest,
) (*apiv1.TestWebhookResponse, error) {
	panic("unimplemented")
}
