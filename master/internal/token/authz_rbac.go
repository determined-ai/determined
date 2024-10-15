package token

import (
	"context"

	"github.com/pkg/errors"
	log "github.com/sirupsen/logrus"
	"github.com/uptrace/bun"

	"github.com/determined-ai/determined/master/internal/db"
	"github.com/determined-ai/determined/master/internal/rbac/audit"
	"github.com/determined-ai/determined/master/pkg/model"
	"github.com/determined-ai/determined/proto/pkg/rbacv1"
)

// TokenAuthZRBAC is the RBAC implementation of user authorization.
type TokenAuthZRBAC struct{}

func logCanAdministrateAccessTokenOnUser(fields log.Fields, curUserID model.UserID,
	permissionID rbacv1.PermissionType,
) {
	fields["userID"] = curUserID
	fields["permissionRequired"] = []audit.PermissionWithSubject{
		{
			PermissionTypes: []rbacv1.PermissionType{
				permissionID,
			},
			SubjectType: "user",
		},
	}
}

// CanCreateAccessToken returns an error if the user does not have permission to create either
// their own token or another user's token based on the targetUser.
func (a *TokenAuthZRBAC) CanCreateAccessToken(
	ctx context.Context, curUser, targetUser model.User,
) (err error) {
	fields := audit.ExtractLogFields(ctx)

	// TODO: improve logging around the case were a user is creating their own token
	if curUser.ID == targetUser.ID {
		err = db.DoesPermissionMatch(ctx, curUser.ID, nil,
			rbacv1.PermissionType_PERMISSION_TYPE_CREATE_TOKEN)
		if err != nil {
			return errors.Wrap(err, "unable to create token due to insufficient permissions")
		}
		return nil
	}

	logCanAdministrateAccessTokenOnUser(fields, curUser.ID,
		rbacv1.PermissionType_PERMISSION_TYPE_CREATE_OTHER_TOKEN)
	defer func() {
		audit.LogFromErr(fields, err)
	}()

	err = db.DoesPermissionMatch(ctx, curUser.ID, nil,
		rbacv1.PermissionType_PERMISSION_TYPE_CREATE_OTHER_TOKEN)
	if err != nil && curUser.ID != targetUser.ID {
		return errors.New("only admin privileged users can create other user's token")
	}
	return nil
}

// CanGetAccessTokens returns an error if the user does not have permission to view own or
// another user's token based on own role permissions.
func (a *TokenAuthZRBAC) CanGetAccessTokens(
	ctx context.Context, curUser model.User, query *bun.SelectQuery, targetUserID *model.UserID,
) (selectQuery *bun.SelectQuery, err error) {
	err = canGetOthersAccessTokens(ctx, curUser)
	if err != nil {
		if targetUserID != nil && *targetUserID != curUser.ID {
			return nil, err
		}
		err = canGetOwnAccessTokens(ctx, curUser)
		if err != nil {
			return nil, err
		}
		query = query.Where("us.user_id = ?", curUser.ID)
	}
	return query, nil
}

// CanGetOthersAccessTokens returns an error if the user does not have permission to view
// another user's token based on own role.
func canGetOthersAccessTokens(
	ctx context.Context, curUser model.User,
) (err error) {
	fields := audit.ExtractLogFields(ctx)
	logCanAdministrateAccessTokenOnUser(fields, curUser.ID,
		rbacv1.PermissionType_PERMISSION_TYPE_VIEW_OTHER_TOKEN)

	defer func(err *error) {
		audit.LogFromErr(fields, *err)
	}(&err)

	// Check if the user has permission to view other users' tokens
	err = db.DoesPermissionMatch(ctx, curUser.ID, nil,
		rbacv1.PermissionType_PERMISSION_TYPE_VIEW_OTHER_TOKEN)
	if err != nil {
		return errors.New("unable to get token due to insufficient permissions")
	}
	return nil
}

// CanGetOwnAccessTokens returns an error if the user does not have permission to view
// their own token based on the targetUser.
func canGetOwnAccessTokens(
	ctx context.Context, curUser model.User,
) (err error) {
	err = db.DoesPermissionMatch(ctx, curUser.ID, nil,
		rbacv1.PermissionType_PERMISSION_TYPE_VIEW_TOKEN)
	if err != nil {
		return errors.Wrap(err, "unable to get token due to insufficient permissions")
	}
	return nil
}

// CanUpdateAccessToken returns an error if the user does not have permission to update either
// their own token or another user's token based on the targetTokenUserID.
func (a *TokenAuthZRBAC) CanUpdateAccessToken(
	ctx context.Context,
	curUser model.User,
	targetTokenUserID model.UserID,
) (err error) {
	fields := audit.ExtractLogFields(ctx)
	// TODO: improve logging around the case were a user is updating their own token's description
	if curUser.ID == targetTokenUserID {
		err = db.DoesPermissionMatch(ctx, curUser.ID, nil,
			rbacv1.PermissionType_PERMISSION_TYPE_UPDATE_TOKEN)
		if err != nil {
			return errors.Wrap(err, "unable to update token due to insufficient permissions")
		}
		return nil
	}

	logCanAdministrateAccessTokenOnUser(fields, curUser.ID,
		rbacv1.PermissionType_PERMISSION_TYPE_ADMINISTRATE_TOKEN)
	defer func() {
		audit.LogFromErr(fields, err)
	}()

	err = db.DoesPermissionMatch(ctx, curUser.ID, nil,
		rbacv1.PermissionType_PERMISSION_TYPE_ADMINISTRATE_TOKEN)
	if err != nil && curUser.ID != targetTokenUserID {
		return errors.Wrap(err, "unable to update token due to insufficient permissions")
	}
	return nil
}

func init() {
	AuthZProvider.Register("rbac", &TokenAuthZRBAC{})
}
