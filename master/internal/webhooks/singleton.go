package webhooks

import (
	"context"

	log "github.com/sirupsen/logrus"

	"github.com/determined-ai/determined/master/pkg/model"
	"github.com/determined-ai/determined/proto/pkg/webhookv1"
)

var defaultManager *WebhookManager

// SetDefault sets the default webhook manager singleton.
func SetDefault(w *WebhookManager) {
	defaultManager = w
}

// ScanLogs sends webhooks for task logs. This should be called wherever we add task logs.
func ScanLogs(ctx context.Context, logs []*model.TaskLog, workspaceID model.AccessScopeID, expID *int) error {
	if defaultManager == nil {
		log.Error("webhook manager is uninitialized")
		return nil
	}

	return defaultManager.scanLogs(ctx, logs, workspaceID, expID)
}

// AddWebhook adds a Webhook and its Triggers to the DB.
func AddWebhook(ctx context.Context, w *Webhook) error {
	if defaultManager == nil {
		log.Error("webhook manager is uninitialized")
		return nil
	}

	return defaultManager.addWebhook(ctx, w)
}

// DeleteWebhook deletes a Webhook and its Triggers from the DB.
func DeleteWebhook(ctx context.Context, id WebhookID) error {
	if defaultManager == nil {
		log.Error("webhook manager is uninitialized")
		return nil
	}

	return defaultManager.deleteWebhook(ctx, id)
}

// UpdateWebhook updates a Webhook in the DB.
func UpdateWebhook(ctx context.Context, id int32, p *webhookv1.PatchWebhook) error {
	if defaultManager == nil {
		log.Error("webhook manager is uninitialized")
		return nil
	}

	return defaultManager.updateWebhook(ctx, id, p)
}
