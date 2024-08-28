package webhooks

import (
	"context"

	"github.com/determined-ai/determined/master/internal/authz"
	"github.com/determined-ai/determined/master/pkg/model"
)

// WebhookAuthZ describes authz methods for experiments.
type WebhookAuthZ interface {
	// GET /api/v1/webhooks
	WebhookAvailableWorkspaces(
		ctx context.Context, curUser *model.User) (workspaceIDsWithPermsFilter []int32, serverError error)
	// POST /api/v1/webhooks
	// DELETE /api/v1/webhooks/:webhook_id
	// POST /api/v1/webhooks/test/:webhook_id
	CanEditWebhooks(ctx context.Context, curUser *model.User, workspace *model.Workspace) (serverError error)
}

// AuthZProvider is the authz registry for experiments.
var AuthZProvider authz.AuthZProviderType[WebhookAuthZ]
