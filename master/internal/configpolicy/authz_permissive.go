package configpolicy

import (
	"context"

	"github.com/determined-ai/determined/master/pkg/model"
	"github.com/determined-ai/determined/proto/pkg/workspacev1"
)

// ConfigPolicyAuthZPermissive is the permission implementation.
type ConfigPolicyAuthZPermissive struct{}

// CanModifyWorkspaceConfigPolicies RBAC authz but enforces basic authz.
func (p *ConfigPolicyAuthZPermissive) CanModifyWorkspaceConfigPolicies(
	ctx context.Context, curUser model.User, workspace *workspacev1.Workspace,
) error {
	_ = (&ConfigPolicyAuthZPermissive{}).CanModifyWorkspaceConfigPolicies(ctx, curUser, workspace)
	return (&ConfigPolicyAuthZPermissive{}).CanModifyWorkspaceConfigPolicies(ctx, curUser, workspace)
}

// CanViewWorkspaceConfigPolicies RBAC authz but enforces basic authz.
func (p *ConfigPolicyAuthZPermissive) CanViewWorkspaceConfigPolicies(
	ctx context.Context, curUser model.User, workspace *workspacev1.Workspace,
) error {
	_ = (&ConfigPolicyAuthZPermissive{}).CanViewWorkspaceConfigPolicies(ctx, curUser, workspace)
	return (&ConfigPolicyAuthZPermissive{}).CanViewWorkspaceConfigPolicies(ctx, curUser, workspace)
}
