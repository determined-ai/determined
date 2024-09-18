package configpolicy

import (
	"context"
	"fmt"

	"github.com/determined-ai/determined/master/internal/grpcutil"
	"github.com/determined-ai/determined/master/pkg/model"
	"github.com/determined-ai/determined/proto/pkg/workspacev1"
)

// ConfigPolicyAuthZBasic is classic OSS controls.
type ConfigPolicyAuthZBasic struct{}

// CanModifyWorkspaceConfigPolicies requires curUser to be an admin or workspace owner.
func (a *ConfigPolicyAuthZBasic) CanModifyWorkspaceConfigPolicies(
	ctx context.Context, curUser model.User, workspace *workspacev1.Workspace,
) error {
	if !curUser.Admin && curUser.ID != model.UserID(workspace.UserId) {
		return fmt.Errorf("only admins may set config policies for workspaces")
	}
	return nil
}

// CanViewWorkspaceConfigPolicies returns a nil error.
func (a *ConfigPolicyAuthZBasic) CanViewWorkspaceConfigPolicies(
	ctx context.Context, curUser model.User, workspace *workspacev1.Workspace,
) error {
	return nil
}

// CanModifyGlobalConfigPolicies requires curUser to be an admin.
func (a *ConfigPolicyAuthZBasic) CanModifyGlobalConfigPolicies(ctx context.Context, curUser *model.User,
) error {
	if !curUser.Admin {
		return grpcutil.ErrPermissionDenied
	}
	return nil
}

// CanViewGlobalConfigPolicies returns a nil error.
func (a *ConfigPolicyAuthZBasic) CanViewGlobalConfigPolicies(ctx context.Context, curUser *model.User,
) error {
	return nil
}

func init() {
	AuthZProvider.Register("basic", &ConfigPolicyAuthZBasic{})
}
