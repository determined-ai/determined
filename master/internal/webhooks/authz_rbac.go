package webhooks

import (
	"context"

	"github.com/determined-ai/determined/master/internal/db"
	"github.com/determined-ai/determined/master/pkg/model"
	"github.com/determined-ai/determined/proto/pkg/rbacv1"
)

// WebhookAuthZRBAC is RBAC webhook access controls.
type WebhookAuthZRBAC struct{}

// CanEditWebhooks checks if a user can edit webhooks.
func (a *WebhookAuthZRBAC) CanEditWebhooks(ctx context.Context, curUser *model.User) error {
	return db.DoesPermissionMatch(ctx, curUser.ID, nil,
		rbacv1.PermissionType_PERMISSION_TYPE_EDIT_WEBHOOKS)
}

func init() {
	AuthZProvider.Register("rbac", &WebhookAuthZRBAC{})
}
