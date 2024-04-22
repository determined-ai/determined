package oauth

import (
	"context"

	"github.com/determined-ai/determined/master/pkg/model"
)

// OauthAuthZPermissive is the permission implementation.
type OauthAuthZPermissive struct{}

// CanAdministrateOauth calls RBAC authz but enforces basic authz.
func (a *OauthAuthZPermissive) CanAdministrateOauth(ctx context.Context,
	curUser model.User,
) error {
	_ = (&OauthAuthZRBAC{}).CanAdministrateOauth(ctx, curUser)
	return (&OauthAuthZBasic{}).CanAdministrateOauth(ctx, curUser)
}

func init() {
	AuthZProvider.Register("permissive", &OauthAuthZPermissive{})
}
