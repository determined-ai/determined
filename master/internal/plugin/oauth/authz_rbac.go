package oauth

import (
	"context"

	"github.com/determined-ai/determined/master/internal/config"
	"github.com/determined-ai/determined/master/internal/db"
	"github.com/determined-ai/determined/master/pkg/model"
	"github.com/determined-ai/determined/proto/pkg/rbacv1"
)

// OauthAuthZRBAC is the RBAC implementation of the OauthAuthZ interface.
type OauthAuthZRBAC struct{}

// CanAdministrateOauth checks if the user has permission to view
// and modify oauth clients and settings.
func (a *OauthAuthZRBAC) CanAdministrateOauth(
	ctx context.Context, curUser model.User,
) error {
	return db.DoesPermissionMatch(ctx, curUser.ID, nil,
		rbacv1.PermissionType_PERMISSION_TYPE_ADMINISTRATE_OAUTH)
}

func init() {
	AuthZProvider.Register(config.RBACAuthZType, &OauthAuthZRBAC{})
}
