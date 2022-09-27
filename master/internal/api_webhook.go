package internal

import (
	"context"
	webhookv1 "github.com/determined-ai/determined/proto/pkg/webhookv1"
	"github.com/determined-ai/determined/master/internal/db"
	"github.com/determined-ai/determined/proto/pkg/apiv1"
	"github.com/determined-ai/determined/master/pkg/actor"
)

var webhooksAddr = actor.Addr("webhooks")

func (a *apiServer) GetWebhooks(
	ctx context.Context, req *apiv1.GetWebhooksRequest,
) (*apiv1.GetWebhooksResponse, error) {
	resp := &apiv1.GetWebhooksResponse{Webhooks: []*webhookv1.Webhook{}}
	err := db.Bun().NewSelect().
		Model(&resp.Webhooks).Scan(ctx)
	if err != nil {
		return nil, err
	}

	return resp, nil
}


func (a *apiServer) PostWebhook(
	ctx context.Context, req *apiv1.PostWebhookRequest,
) (*apiv1.PostWebhookResponse, error) {

	w := &webhookv1.Webhook{}
	err := a.m.db.QueryProto("insert_webhook", w)

	return &apiv1.PostWebhookResponse{Webhook: w},
		err
}
