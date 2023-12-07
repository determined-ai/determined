package oauth

import (
	"context"

	"github.com/determined-ai/determined/master/internal/authz"
	"github.com/determined-ai/determined/master/internal/config"
	"github.com/determined-ai/determined/master/internal/db"
	"github.com/determined-ai/determined/master/internal/rbac/audit"
	"github.com/determined-ai/determined/master/pkg/model"
	"github.com/determined-ai/determined/proto/pkg/rbacv1"
)

// OauthAuthZRBAC is the RBAC implementation of the OauthAuthZ interface.
type OauthAuthZRBAC struct{}

// CanAdministrateOauth checks if the user has permission to view
// and modify oauth clients and settings.
func (a *OauthAuthZRBAC) CanAdministrateOauth(
	ctx context.Context, curUser model.User,
) (err error) {
	fields := audit.ExtractLogFields(ctx)
	defer func() {
		if err == nil || authz.IsPermissionDenied(err) {
			fields["permissionRequired"] = []audit.PermissionWithSubject{
				{
					PermissionTypes: []rbacv1.PermissionType{
						rbacv1.PermissionType_PERMISSION_TYPE_ADMINISTRATE_OAUTH,
					},
					SubjectType: "oauth",
				},
			}
			fields["permissionGranted"] = !authz.IsPermissionDenied(err)
			audit.Log(fields)
		}
	}()
	err = db.DoesPermissionMatch(ctx, curUser.ID, nil,
		rbacv1.PermissionType_PERMISSION_TYPE_ADMINISTRATE_OAUTH)
	return err
}

func init() {
	AuthZProvider.Register(config.RBACAuthZType, &OauthAuthZRBAC{})
}
