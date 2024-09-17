package configpolicy

import (
	"context"

	"github.com/determined-ai/determined/master/pkg/model"
	"github.com/determined-ai/determined/proto/pkg/workspacev1"
)

// ConfigPolicyAuthZPermissive is the permission implementation.
type ConfigPolicyAuthZPermissive struct{}

// CanModifyWorkspaceConfigPolicies calls RBAC authz but enforces basic authz.
func (p *ConfigPolicyAuthZPermissive) CanModifyWorkspaceConfigPolicies(
	ctx context.Context, curUser model.User, workspace *workspacev1.Workspace,
) error {
	_ = (&ConfigPolicyAuthZRBAC{}).CanModifyWorkspaceConfigPolicies(ctx, curUser, workspace)
	return (&ConfigPolicyAuthZBasic{}).CanModifyWorkspaceConfigPolicies(ctx, curUser, workspace)
}

// CanViewWorkspaceConfigPolicies calls RBAC authz but enforces basic authz.
func (p *ConfigPolicyAuthZPermissive) CanViewWorkspaceConfigPolicies(
	ctx context.Context, curUser model.User, workspace *workspacev1.Workspace,
) error {
	_ = (&ConfigPolicyAuthZRBAC{}).CanViewWorkspaceConfigPolicies(ctx, curUser, workspace)
	return (&ConfigPolicyAuthZBasic{}).CanViewWorkspaceConfigPolicies(ctx, curUser, workspace)
}
