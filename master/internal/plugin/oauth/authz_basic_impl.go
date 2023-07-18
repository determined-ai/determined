package oauth

import (
	"context"

	"github.com/determined-ai/determined/master/internal/authz"
	"github.com/determined-ai/determined/master/pkg/model"
)

// OauthAuthZBasic is basic OSS controls.
type OauthAuthZBasic struct{}

// CanAdministrateOauth returns an error if the current user is not an admin.
func (a *OauthAuthZBasic) CanAdministrateOauth(
	_ context.Context, curUser model.User,
) error {
	if !curUser.Admin {
		return authz.PermissionDeniedError{}.WithPrefix(
			"non-admin users may not view or modify oauth clients or settings",
		)
	}
	return nil
}

func init() {
	AuthZProvider.Register("basic", &OauthAuthZBasic{})
}
