package webhooks

import (
	"github.com/determined-ai/determined/master/pkg/model"
)

// WebhookAuthZBasic is basic OSS controls.
type WebhookAuthZBasic struct{}

// CanEditWebhooks always returns true and a nil error.
func (a *WebhookAuthZBasic) CanEditWebhooks(
	curUser *model.User,
) (canEditWeb bool, serverError error) {
	return true, nil
}

func init() {
	AuthZProvider.Register("basic", &WebhookAuthZBasic{})
}
