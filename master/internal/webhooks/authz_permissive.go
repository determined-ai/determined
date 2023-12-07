package webhooks

import (
	"context"

	"github.com/determined-ai/determined/master/pkg/model"
)

// WebhookAuthZPermissive is the permission implementation.
type WebhookAuthZPermissive struct{}

// CanEditWebhooks calls RBAC authz but enforces basic authz.
func (p *WebhookAuthZPermissive) CanEditWebhooks(
	ctx context.Context, curUser *model.User,
) error {
	_ = (&WebhookAuthZRBAC{}).CanEditWebhooks(ctx, curUser)
	return (&WebhookAuthZBasic{}).CanEditWebhooks(ctx, curUser)
}

func init() {
	AuthZProvider.Register("permissive", &WebhookAuthZPermissive{})
}
