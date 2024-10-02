package user

import (
	"context"
	"fmt"

	"github.com/pkg/errors"
	log "github.com/sirupsen/logrus"

	"github.com/determined-ai/determined/master/internal/db"
	"github.com/determined-ai/determined/master/internal/rbac/audit"
	"github.com/determined-ai/determined/master/pkg/model"
	"github.com/determined-ai/determined/proto/pkg/rbacv1"
)

// UserAuthZRBAC is the RBAC implementation of user authorization.
type UserAuthZRBAC struct{}

func noPermissionRequired(ctx context.Context, curUserID, targetUserID model.UserID) {
	fields := audit.ExtractLogFields(ctx)
	fields["userID"] = curUserID
	fields["permissionRequired"] = []audit.PermissionWithSubject{
		{
			PermissionTypes: []rbacv1.PermissionType{},
			SubjectType:     "user",
			SubjectIDs:      []string{fmt.Sprint(targetUserID)},
		},
	}
	audit.Log(fields)
}

func logCanAdministrateUser(fields log.Fields, curUserID model.UserID) {
	fields["userID"] = curUserID
	fields["permissionRequired"] = []audit.PermissionWithSubject{
		{
			PermissionTypes: []rbacv1.PermissionType{
				rbacv1.PermissionType_PERMISSION_TYPE_ADMINISTRATE_USER,
			},
			SubjectType: "user",
		},
	}
}

func canAdministrateUser(ctx context.Context, curUserID model.UserID) (err error) {
	fields := audit.ExtractLogFields(ctx)
	logCanAdministrateUser(fields, curUserID)
	defer func() {
		audit.LogFromErr(fields, err)
	}()

	return db.DoesPermissionMatch(ctx, curUserID, nil,
		rbacv1.PermissionType_PERMISSION_TYPE_ADMINISTRATE_USER)
}

// CanGetUser always returns true.
func (a *UserAuthZRBAC) CanGetUser(
	ctx context.Context, curUser, targetUser model.User,
) error {
	noPermissionRequired(ctx, curUser.ID, targetUser.ID)
	return nil
}

// FilterUserList always returns the input user list and does not filtering.
func (a *UserAuthZRBAC) FilterUserList(
	ctx context.Context, curUser model.User, users []model.FullUser,
) ([]model.FullUser, error) {
	userIDs := make([]string, 0, len(users))
	for _, user := range users {
		userIDs = append(userIDs, fmt.Sprint(user.ID))
	}

	fields := audit.ExtractLogFields(ctx)
	fields["userID"] = curUser.ID
	fields["permissionRequired"] = []audit.PermissionWithSubject{
		{
			PermissionTypes: []rbacv1.PermissionType{},
			SubjectType:     "user",
			SubjectIDs:      userIDs,
		},
	}
	audit.Log(fields)

	return users, nil
}

// CanCreateUser returns an error if the user does not have admin permissions or
// does not have permission to update groups.
func (a *UserAuthZRBAC) CanCreateUser(
	ctx context.Context, curUser, userToAdd model.User, agentUserGroup *model.AgentUserGroup,
) error {
	return canAdministrateUser(ctx, curUser.ID)
}

// CanSetUsersPassword returns an error if the user is not the target user and does not have admin
// permissions when trying to set another user's password.
func (a *UserAuthZRBAC) CanSetUsersPassword(
	ctx context.Context, curUser, targetUser model.User,
) (err error) {
	fields := audit.ExtractLogFields(ctx)
	logCanAdministrateUser(fields, curUser.ID)
	defer func() {
		audit.LogFromErr(fields, err)
	}()

	err = db.DoesPermissionMatch(ctx, curUser.ID, nil,
		rbacv1.PermissionType_PERMISSION_TYPE_ADMINISTRATE_USER)
	if err != nil && curUser.ID != targetUser.ID {
		return errors.New("only admin privileged users can change other user's passwords")
	}
	return nil
}

// CanSetUsersActive returns an error if the user does not have admin permissions.
func (a *UserAuthZRBAC) CanSetUsersActive(
	ctx context.Context, curUser, targetUser model.User, toActiveVal bool,
) error {
	return canAdministrateUser(ctx, curUser.ID)
}

// CanSetUsersAdmin returns an error if the user does not have admin permissions.
func (a *UserAuthZRBAC) CanSetUsersAdmin(
	ctx context.Context, curUser, targetUser model.User, toAdminVal bool,
) error {
	return canAdministrateUser(ctx, curUser.ID)
}

// CanSetUsersAgentUserGroup returns an error if the user does not have admin permissions.
func (a *UserAuthZRBAC) CanSetUsersAgentUserGroup(
	ctx context.Context, curUser, targetUser model.User, agentUserGroup model.AgentUserGroup,
) error {
	return canAdministrateUser(ctx, curUser.ID)
}

// CanSetUsersUsername returns an error if the user does not have admin permissions.
func (a *UserAuthZRBAC) CanSetUsersUsername(
	ctx context.Context, curUser, targetUser model.User,
) error {
	return canAdministrateUser(ctx, curUser.ID)
}

// CanSetUsersDisplayName returns an error if the user is not an admin and does not have admin
// permissions when trying to set another user's display name.
func (a *UserAuthZRBAC) CanSetUsersDisplayName(
	ctx context.Context, curUser, targetUser model.User,
) (err error) {
	fields := audit.ExtractLogFields(ctx)
	logCanAdministrateUser(fields, curUser.ID)
	defer func() {
		audit.LogFromErr(fields, err)
	}()

	if curUser == targetUser {
		return nil
	}

	err = db.DoesPermissionMatch(ctx, curUser.ID, nil,
		rbacv1.PermissionType_PERMISSION_TYPE_ADMINISTRATE_USER)
	if err != nil && curUser.ID != targetUser.ID {
		return errors.New("only admin privileged users can set another user's display name")
	}
	return nil
}

// CanGetUsersImage always returns nil.
func (a *UserAuthZRBAC) CanGetUsersImage(
	ctx context.Context, curUser, targetUser model.User,
) error {
	noPermissionRequired(ctx, curUser.ID, targetUser.ID)

	return nil
}

// CanGetUsersOwnSettings always returns nil.
func (a *UserAuthZRBAC) CanGetUsersOwnSettings(ctx context.Context, curUser model.User) error {
	noPermissionRequired(ctx, curUser.ID, curUser.ID)

	return nil
}

// CanCreateUsersOwnSetting always returns nil.
func (a *UserAuthZRBAC) CanCreateUsersOwnSetting(
	ctx context.Context, curUser model.User, setting []*model.UserWebSetting,
) error {
	noPermissionRequired(ctx, curUser.ID, curUser.ID)

	return nil
}

// CanResetUsersOwnSettings always returns nil.
func (a *UserAuthZRBAC) CanResetUsersOwnSettings(ctx context.Context, curUser model.User) error {
	noPermissionRequired(ctx, curUser.ID, curUser.ID)

	return nil
}

// CanSetUsersRemote returns an error if the user does not have admin permissions.
func (a *UserAuthZRBAC) CanSetUsersRemote(ctx context.Context, curUser model.User) error {
	return canAdministrateUser(ctx, curUser.ID)
}

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
func (a *UserAuthZRBAC) CanCreateAccessToken(
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

// CanGetAccessTokens returns an error if the user does not have permission to view either
// their own token or another user's token based on the targetUser.
func (a *UserAuthZRBAC) CanGetAccessTokens(
	ctx context.Context, curUser, targetUser model.User,
) (err error) {
	fields := audit.ExtractLogFields(ctx)

	// TODO: improve logging around the case were a user is viewing their own token's description
	if curUser.ID == targetUser.ID {
		err = db.DoesPermissionMatch(ctx, curUser.ID, nil,
			rbacv1.PermissionType_PERMISSION_TYPE_VIEW_TOKEN)
		if err != nil {
			return errors.Wrap(err, "unable to get token due to insufficient permissions")
		}
		return nil
	}

	logCanAdministrateAccessTokenOnUser(fields, curUser.ID,
		rbacv1.PermissionType_PERMISSION_TYPE_VIEW_OTHER_TOKEN)
	defer func() {
		audit.LogFromErr(fields, err)
	}()

	err = db.DoesPermissionMatch(ctx, curUser.ID, nil,
		rbacv1.PermissionType_PERMISSION_TYPE_VIEW_OTHER_TOKEN)
	if err != nil && curUser.ID != targetUser.ID {
		return errors.New("unable to get token due to insufficient permissions")
	}
	return nil
}

// CanUpdateAccessToken returns an error if the user does not have permission to update either
// their own token or another user's token based on the targetTokenUserID.
func (a *UserAuthZRBAC) CanUpdateAccessToken(
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
	AuthZProvider.Register("rbac", &UserAuthZRBAC{})
}
