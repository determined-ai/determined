package configpolicy

import (
	"context"
	"strconv"

	log "github.com/sirupsen/logrus"

	"github.com/determined-ai/determined/master/internal/db"
	"github.com/determined-ai/determined/master/internal/rbac/audit"
	"github.com/determined-ai/determined/master/pkg/model"
	"github.com/determined-ai/determined/proto/pkg/rbacv1"
	"github.com/determined-ai/determined/proto/pkg/workspacev1"
)

func init() {
	AuthZProvider.Register("rbac", &ConfigPolicyAuthZRBAC{})
}

// ConfigPolicyAuthZRBAC is RBAC authorization for config policies.
type ConfigPolicyAuthZRBAC struct{}

// CanModifyWorkspaceConfigPolicies determines whether a user can modify
// workspace task config policies.
func (r *ConfigPolicyAuthZRBAC) CanModifyWorkspaceConfigPolicies(
	ctx context.Context, curUser model.User, workspace *workspacev1.Workspace,
) (err error) {
	fields := audit.ExtractLogFields(ctx)
	addConfigPolicyInfo(curUser, workspace, fields, []rbacv1.PermissionType{
		rbacv1.PermissionType_PERMISSION_TYPE_MODIFY_WORKSPACE_CONFIG_POLICIES,
	})
	defer func() {
		audit.LogFromErr(fields, err)
	}()

	return db.DoesPermissionMatch(ctx, curUser.ID, nil,
		rbacv1.PermissionType_PERMISSION_TYPE_MODIFY_WORKSPACE_CONFIG_POLICIES)
}

// CanViewWorkspaceConfigPolicies determines whether a user can view
// workspace task config policies.
func (r *ConfigPolicyAuthZRBAC) CanViewWorkspaceConfigPolicies(
	ctx context.Context, curUser model.User, workspace *workspacev1.Workspace,
) (err error) {
	fields := audit.ExtractLogFields(ctx)
	addConfigPolicyInfo(curUser, workspace, fields, []rbacv1.PermissionType{
		rbacv1.PermissionType_PERMISSION_TYPE_MODIFY_WORKSPACE_CONFIG_POLICIES,
	})
	defer func() {
		audit.LogFromErr(fields, err)
	}()

	return db.DoesPermissionMatch(ctx, curUser.ID, nil,
		rbacv1.PermissionType_PERMISSION_TYPE_VIEW_WORKSPACE_CONFIG_POLICIES)
}

func addConfigPolicyInfo(curUser model.User,
	workspace *workspacev1.Workspace,
	logFields log.Fields,
	permissions []rbacv1.PermissionType,
) {
	logFields["userID"] = curUser.ID
	logFields["permissionsRequired"] = []audit.PermissionWithSubject{
		{
			PermissionTypes: permissions,
			SubjectType:     "config policy",
			SubjectIDs:      []string{strconv.Itoa(int(workspace.Id))},
		},
	}
}
