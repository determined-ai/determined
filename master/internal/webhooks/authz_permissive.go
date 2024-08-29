package webhooks

import (
	"context"

	"github.com/determined-ai/determined/master/pkg/model"
)

// WebhookAuthZPermissive is the permission implementation.
type WebhookAuthZPermissive struct{}

// CanEditWebhooks calls RBAC authz but enforces basic authz.
func (p *WebhookAuthZPermissive) CanEditWebhooks(
	ctx context.Context, curUser *model.User, workspace *model.Workspace,
) error {
	_ = (&WebhookAuthZRBAC{}).CanEditWebhooks(ctx, curUser, workspace)
	return (&WebhookAuthZBasic{}).CanEditWebhooks(ctx, curUser, workspace)
}

// WebhookAvailableWorkspaces calls RBAC authz but enforces basic authz.
func (p *WebhookAuthZPermissive) WebhookAvailableWorkspaces(
	ctx context.Context, curUser *model.User,
) (workspaceIDsWithPermsFilter []int32, serverError error) {
	_, _ = (&WebhookAuthZRBAC{}).WebhookAvailableWorkspaces(ctx, curUser)
	return (&WebhookAuthZBasic{}).WebhookAvailableWorkspaces(ctx, curUser)
}

func init() {
	AuthZProvider.Register("permissive", &WebhookAuthZPermissive{})
}
