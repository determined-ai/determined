package webhooks

import (
	"context"

	"github.com/determined-ai/determined/master/internal/db"
	"github.com/determined-ai/determined/master/internal/rbac"
	"github.com/determined-ai/determined/master/pkg/model"
	"github.com/determined-ai/determined/master/pkg/ptrs"
	"github.com/determined-ai/determined/proto/pkg/rbacv1"
)

// WebhookAuthZRBAC is RBAC webhook access controls.
type WebhookAuthZRBAC struct{}

// CanEditWebhooks checks if a user can edit webhooks.
// workspace being nil means the webhook is globally scoped.
func (a *WebhookAuthZRBAC) CanEditWebhooks(
	ctx context.Context, curUser *model.User, workspace *model.Workspace,
) error {
	var workspaceID *int32
	if workspace != nil {
		workspaceID = ptrs.Ptr(int32(workspace.ID))
	}
	return db.DoesPermissionMatch(ctx, curUser.ID, workspaceID,
		rbacv1.PermissionType_PERMISSION_TYPE_EDIT_WEBHOOKS)
}

// CanGetWebhooks returns a list of workspace that user can get webhooks from.
func (a *WebhookAuthZRBAC) CanGetWebhooks(
	ctx context.Context, curUser *model.User,
) (workspaceIDsWithPermsFilter []int32, serverError error) {
	var workspaceIDs []int32
	assignmentsMap, err := rbac.GetPermissionSummary(ctx, curUser.ID)
	if err != nil {
		return workspaceIDs, err
	}
	for role, roleAssignments := range assignmentsMap {
		for _, permission := range role.Permissions {
			if permission.ID == int(
				rbacv1.PermissionType_PERMISSION_TYPE_VIEW_WEBHOOKS) {
				for _, assignment := range roleAssignments {
					if !assignment.Scope.WorkspaceID.Valid {
						// if permission is global, return the entire list of workspaces.
						err := db.Bun().NewSelect().Table("workspaces").Column("id").Scan(ctx, &workspaceIDs)
						return workspaceIDs, err
					}
					workspaceIDs = append(workspaceIDs, assignment.Scope.WorkspaceID.Int32)
				}
			}
		}
	}
	return workspaceIDs, nil
}

func init() {
	AuthZProvider.Register("rbac", &WebhookAuthZRBAC{})
}
