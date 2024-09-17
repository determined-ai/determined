package configpolicy

import (
	"context"

	"github.com/determined-ai/determined/master/internal/authz"
	"github.com/determined-ai/determined/master/pkg/model"
	"github.com/determined-ai/determined/proto/pkg/workspacev1"
)

// ConfigPolicyAuthZ describes authz methods for config policies.
type ConfigPolicyAuthZ interface {
	// PUT /api/v1/config-policies/workspaces/:workspace-id/:type
	CanModifyWorkspaceConfigPolicies(ctx context.Context, curUser model.User, workspace *workspacev1.Workspace,
	) error
	// GET /api/v1/config-policies/workspaces/:workspace-id/:type
	CanViewWorkspaceConfigPolicies(ctx context.Context, curUser model.User, workspace *workspacev1.Workspace,
	) error
}

// AuthZProvider providers WorkspaceAuthZ implementations.
var AuthZProvider authz.AuthZProviderType[ConfigPolicyAuthZ]
