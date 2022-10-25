package webhooks

import (
	"context"
	"fmt"

	"github.com/determined-ai/determined/master/pkg/model"
)

// WebhookAuthZBasic is basic OSS controls.
type WebhookAuthZBasic struct{}

// CanEditWebhooks always returns true and a nil error.
func (a *WebhookAuthZBasic) CanEditWebhooks(
	ctx context.Context, curUser *model.User,
) (serverError error) {
	if !curUser.Admin {
		return fmt.Errorf("non admin users can't edit webhooks")
	}
	return nil
}

func init() {
	AuthZProvider.Register("basic", &WebhookAuthZBasic{})
}
