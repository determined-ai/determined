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
	err := a.m.db.QueryProto("insert_webhook", w, req.Url)

	if err == nil {
		for _, trigger := range req.Triggers {
			var triggerType string
			t := &webhookv1.Trigger{}
			switch trigger.TriggerType {
			case webhookv1.TriggerType_TRIGGER_TYPE_METRIC_THRESHOLD_EXCEEDED:
				triggerType = "METRIC_THRESHOLD_EXCEEDED"
			case webhookv1.TriggerType_TRIGGER_TYPE_EXPERIMENT_STATE_CHANGE:
				triggerType = "EXPERIMENT_STATE_CHANGE"
			}
			err = a.m.db.QueryProto("insert_webhook_trigger", t,triggerType,  trigger.Condition, w.Id)
		}
	}

	return &apiv1.PostWebhookResponse{Webhook: w},
		err
}


func (a *apiServer) DeleteWebhook(
	ctx context.Context, req *apiv1.DeleteWebhookRequest,
) (*apiv1.DeleteWebhookResponse, error) {
	_, err := db.Bun().NewDelete().Model(&webhookv1.Webhook{Id: req.Id}).Where("id = ?", req.Id).Exec(context.TODO())
	return &apiv1.DeleteWebhookResponse{Completed: true}, err
}
