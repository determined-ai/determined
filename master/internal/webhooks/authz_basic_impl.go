package webhooks

import (
	"context"
	"fmt"

	"github.com/determined-ai/determined/master/internal/db"
	"github.com/determined-ai/determined/master/pkg/model"
)

// WebhookAuthZBasic is basic OSS controls.
type WebhookAuthZBasic struct{}

// CanEditWebhooks always returns true and a nil error.
// workspace being nil means the webhook is globally scoped.
func (a *WebhookAuthZBasic) CanEditWebhooks(
	ctx context.Context, curUser *model.User, workspace *model.Workspace,
) (serverError error) {
	if workspace != nil && curUser.ID == workspace.UserID {
		return nil
	}
	if !curUser.Admin {
		return fmt.Errorf("non admin users can't edit webhooks")
	}
	return nil
}

// CanGetWebhooks returns a list of workspace that user can get webhooks from.
func (a *WebhookAuthZBasic) CanGetWebhooks(
	ctx context.Context, curUser *model.User,
) (workspaceIDsWithPermsFilter []int32, serverError error) {
	var workspaceIDs []int32
	q := db.Bun().NewSelect().Table("workspaces").Column("id")
	if !curUser.Admin {
		q.Where("user_id = ?", curUser.ID)
	}
	err := q.Scan(ctx, &workspaceIDs)
	return workspaceIDs, err
}

func init() {
	AuthZProvider.Register("basic", &WebhookAuthZBasic{})
}
