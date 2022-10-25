package internal

import (
	"context"
	"fmt"
	"regexp"
	"strings"

	"github.com/pkg/errors"
	"gopkg.in/guregu/null.v3"

	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
	timestamppb "google.golang.org/protobuf/types/known/timestamppb"

	"github.com/determined-ai/determined/master/internal/db"
	"github.com/determined-ai/determined/master/internal/grpcutil"
	"github.com/determined-ai/determined/master/internal/user"
	"github.com/determined-ai/determined/master/pkg/model"
	"github.com/determined-ai/determined/master/pkg/ptrs"
	"github.com/determined-ai/determined/proto/pkg/apiv1"
	"github.com/determined-ai/determined/proto/pkg/userv1"
)

const determinedName = "determined"

var (
	errUserNotFound = status.Error(codes.NotFound, "user not found")
	latinText       = regexp.MustCompile("[^\\p{Latin}\\p{N}\\s]")
)

func clearUsername(targetUser model.User, name string, minLength int) (*string, error) {
	clearName := strings.TrimSpace(name)
	// Reject non-ASCII chars to avoid hidden whitespace, confusable letters, etc.
	if latinText.ReplaceAllLiteralString(clearName, "") != clearName {
		return nil, status.Error(codes.InvalidArgument,
			"Display name and username cannot contain non-ASCII characters.")
	}
	if len(clearName) < minLength {
		return nil, status.Error(codes.InvalidArgument,
			"Display name or username value has minimum length.")
	}
	// Restrict 'admin' and 'determined'.
	if !targetUser.Admin && (strings.ToLower(clearName) == "admin") {
		return nil, status.Error(codes.InvalidArgument, "Non-admin user cannot be renamed 'admin'")
	}
	if targetUser.Username != determinedName && (strings.ToLower(clearName) == determinedName) {
		return nil, status.Error(codes.InvalidArgument, "User cannot be renamed 'determined'")
	}
	return &clearName, nil
}

func validateProtoAgentUserGroup(aug *userv1.AgentUserGroup) error {
	if aug.AgentUid == nil || aug.AgentGid == nil || aug.AgentUser == nil || aug.AgentGroup == nil {
		return status.Error(
			codes.InvalidArgument,
			"AgentUid, AgentGid, AgentUser and AgentGroup cannot be empty",
		)
	}
	if *aug.AgentUser == "" || *aug.AgentGroup == "" {
		return status.Error(
			codes.InvalidArgument,
			"AgentUser and AgentGroup names cannot be empty",
		)
	}

	return nil
}

// TODO(ilia): We need null.Int32.
func i64Ptr2i32(v *int64) *int32 {
	if v == nil {
		return nil
	}

	return ptrs.Ptr(int32(*v))
}

func toProtoUserFromFullUser(user model.FullUser) *userv1.User {
	var agentUserGroup *userv1.AgentUserGroup
	if user.AgentUID.Valid || user.AgentGID.Valid || user.AgentUser.Valid || user.AgentGroup.Valid {
		agentUserGroup = &userv1.AgentUserGroup{
			AgentUid:   i64Ptr2i32(user.AgentUID.Ptr()),
			AgentGid:   i64Ptr2i32(user.AgentGID.Ptr()),
			AgentUser:  user.AgentUser.Ptr(),
			AgentGroup: user.AgentGroup.Ptr(),
		}
	}
	displayNameString := user.DisplayName.ValueOrZero()
	return &userv1.User{
		Id:             int32(user.ID),
		Username:       user.Username,
		Admin:          user.Admin,
		Active:         user.Active,
		AgentUserGroup: agentUserGroup,
		DisplayName:    displayNameString,
		ModifiedAt:     timestamppb.New(user.ModifiedAt),
	}
}

func getFullModelUser(userID model.UserID) (*model.FullUser, error) {
	userModel, err := user.UserByID(userID)
	if errors.Is(err, db.ErrNotFound) {
		return nil, errUserNotFound
	}
	return userModel, err
}

func getUser(d *db.PgDB, userID model.UserID) (*userv1.User, error) {
	user, err := getFullModelUser(userID)
	if err != nil {
		return nil, err
	}
	return toProtoUserFromFullUser(*user), nil
}

// TODO remove this eventually since authz replaces this
// We can't yet since we use it else where.
func userShouldBeAdmin(ctx context.Context, a *apiServer) error {
	u, _, err := grpcutil.GetUser(ctx)
	if err != nil {
		return err
	}
	if !u.Admin {
		return grpcutil.ErrPermissionDenied
	}
	return nil
}

func (a *apiServer) GetUsers(
	ctx context.Context, req *apiv1.GetUsersRequest,
) (*apiv1.GetUsersResponse, error) {
	sortColMap := map[apiv1.GetUsersRequest_SortBy]string{
		apiv1.GetUsersRequest_SORT_BY_UNSPECIFIED:   "id",
		apiv1.GetUsersRequest_SORT_BY_DISPLAY_NAME:  "display_name",
		apiv1.GetUsersRequest_SORT_BY_USER_NAME:     "username",
		apiv1.GetUsersRequest_SORT_BY_ADMIN:         "admin",
		apiv1.GetUsersRequest_SORT_BY_ACTIVE:        "active",
		apiv1.GetUsersRequest_SORT_BY_MODIFIED_TIME: "modified_at",
	}
	orderByMap := map[apiv1.OrderBy]string{
		apiv1.OrderBy_ORDER_BY_UNSPECIFIED: "ASC",
		apiv1.OrderBy_ORDER_BY_ASC:         "ASC",
		apiv1.OrderBy_ORDER_BY_DESC:        "DESC",
	}

	orderExpr := ""
	switch _, ok := sortColMap[req.SortBy]; {
	case !ok:
		return nil, fmt.Errorf("unsupported sort by %s", req.SortBy)
	case sortColMap[req.SortBy] != "id":
		orderExpr = fmt.Sprintf(
			"%s %s, id %s",
			sortColMap[req.SortBy], orderByMap[req.OrderBy], orderByMap[req.OrderBy],
		)
	default:
		orderExpr = fmt.Sprintf("id %s", orderByMap[req.OrderBy])
	}
	users := []model.FullUser{}
	err := a.m.db.QueryF(
		"get_users",
		[]interface{}{orderExpr},
		&users,
	)
	if err != nil {
		return nil, err
	}

	curUser, _, err := grpcutil.GetUser(ctx)
	if err != nil {
		return nil, err
	}
	if users, err = user.AuthZProvider.Get().FilterUserList(*curUser, users); err != nil {
		return nil, err
	}

	resp := &apiv1.GetUsersResponse{}
	for _, user := range users {
		resp.Users = append(resp.Users, toProtoUserFromFullUser(user))
	}

	return resp, a.paginate(&resp.Pagination, &resp.Users, req.Offset, req.Limit)
}

func (a *apiServer) GetUser(
	ctx context.Context, req *apiv1.GetUserRequest,
) (*apiv1.GetUserResponse, error) {
	curUser, _, err := grpcutil.GetUser(ctx)
	if err != nil {
		return nil, err
	}
	targetFullUser, err := getFullModelUser(model.UserID(req.UserId))
	if err != nil {
		return nil, err
	}

	var ok bool
	if ok, err = user.AuthZProvider.Get().CanGetUser(*curUser, targetFullUser.ToUser()); err != nil {
		return nil, err
	} else if !ok {
		return nil, errUserNotFound
	}
	return &apiv1.GetUserResponse{User: toProtoUserFromFullUser(*targetFullUser)}, err
}

func (a *apiServer) PostUser(
	ctx context.Context, req *apiv1.PostUserRequest,
) (*apiv1.PostUserResponse, error) {
	if req.User == nil {
		return nil, status.Error(codes.InvalidArgument, "must specify user to create")
	}
	userToAdd := &model.User{
		Username: req.User.Username,
		Admin:    req.User.Admin,
		Active:   req.User.Active,
	}
	if req.User.DisplayName != "" {
		userToAdd.DisplayName = null.StringFrom(req.User.DisplayName)
	}

	var agentUserGroup *model.AgentUserGroup
	if req.User.AgentUserGroup != nil {
		aug := req.User.AgentUserGroup
		if err := validateProtoAgentUserGroup(aug); err != nil {
			return nil, err
		}

		agentUserGroup = &model.AgentUserGroup{
			UID:   int(*req.User.AgentUserGroup.AgentUid),
			GID:   int(*req.User.AgentUserGroup.AgentGid),
			User:  *req.User.AgentUserGroup.AgentUser,
			Group: *req.User.AgentUserGroup.AgentGroup,
		}
		if agentUserGroup.User == "" || agentUserGroup.Group == "" {
			return nil, status.Error(
				codes.InvalidArgument,
				"AgentUser and AgentGroup names cannot be empty",
			)
		}
	}

	curUser, _, err := grpcutil.GetUser(ctx)
	if err != nil {
		return nil, err
	}
	if err = user.AuthZProvider.Get().
		CanCreateUser(*curUser, *userToAdd, agentUserGroup); err != nil {
		return nil, status.Error(codes.PermissionDenied, err.Error())
	}

	if err = grpcutil.ValidateRequest(
		func() (bool, string) { return req.User != nil, "no user specified" },
		func() (bool, string) { return req.User.Username != "", "no username specified" },
	); err != nil {
		return nil, err
	}
	if err = userToAdd.UpdatePasswordHash(replicateClientSideSaltAndHash(req.Password)); err != nil {
		return nil, err
	}

	userID, err := a.m.db.AddUser(userToAdd, agentUserGroup)
	switch {
	case err == db.ErrDuplicateRecord:
		return nil, status.Error(codes.InvalidArgument, "user already exists")
	case err != nil:
		return nil, err
	}
	fullUser, err := getUser(a.m.db, userID)
	return &apiv1.PostUserResponse{User: fullUser}, err
}

func (a *apiServer) SetUserPassword(
	ctx context.Context, req *apiv1.SetUserPasswordRequest,
) (*apiv1.SetUserPasswordResponse, error) {
	// TODO if ExternalSessions is there, don't even allow this
	curUser, _, err := grpcutil.GetUser(ctx)
	if err != nil {
		return nil, err
	}

	targetFullUser, err := getFullModelUser(model.UserID(req.UserId))
	if err != nil {
		return nil, err
	}
	targetUser := targetFullUser.ToUser()
	if err = user.AuthZProvider.Get().CanSetUsersPassword(*curUser, targetUser); err != nil {
		if ok, canGetErr := user.AuthZProvider.
			Get().CanGetUser(*curUser, targetFullUser.ToUser()); canGetErr != nil {
			return nil, canGetErr
		} else if !ok {
			return nil, errUserNotFound
		}
		return nil, status.Error(codes.PermissionDenied, err.Error())
	}

	if err = targetUser.UpdatePasswordHash(replicateClientSideSaltAndHash(req.Password)); err != nil {
		return nil, err
	}
	switch err = a.m.db.UpdateUser(&targetUser, []string{"password_hash"}, nil); {
	case err == db.ErrNotFound:
		return nil, errUserNotFound
	case err != nil:
		return nil, err
	}
	fullUser, err := getUser(a.m.db, model.UserID(req.UserId))
	return &apiv1.SetUserPasswordResponse{User: fullUser}, err
}

func (a *apiServer) PatchUser(
	ctx context.Context, req *apiv1.PatchUserRequest,
) (*apiv1.PatchUserResponse, error) {
	curUser, _, err := grpcutil.GetUser(ctx)
	if err != nil {
		return nil, err
	}

	uid := model.UserID(req.UserId)
	targetFullUser, err := getFullModelUser(uid)
	if err != nil {
		return nil, err
	}
	targetUser := targetFullUser.ToUser()
	if err = user.AuthZProvider.Get().CanSetUsersDisplayName(*curUser, targetUser); err != nil {
		if ok, canGetErr := user.AuthZProvider.Get().
			CanGetUser(*curUser, targetFullUser.ToUser()); canGetErr != nil {
			return nil, canGetErr
		} else if !ok {
			return nil, errUserNotFound
		}
		return nil, status.Error(codes.PermissionDenied, err.Error())
	}

	if req.User.DisplayName != nil {
		if err = user.AuthZProvider.Get().CanSetUsersDisplayName(*curUser, targetUser); err != nil {
			if ok, canGetErr := user.AuthZProvider.Get().
				CanGetUser(*curUser, targetFullUser.ToUser()); canGetErr != nil {
				return nil, canGetErr
			} else if !ok {
				return nil, errUserNotFound
			}
			return nil, status.Error(codes.PermissionDenied, err.Error())
		}

		u := &userv1.User{}

		displayName, err2 := clearUsername(targetUser, *req.User.DisplayName, 0)
		if err2 != nil {
			return nil, err2
		}

		if displayName == nil || *displayName == "" {
			// Set empty display name to null.
			err = a.m.db.QueryProto("set_user_display_name", u, req.UserId, nil)
		} else {
			err = a.m.db.QueryProto("set_user_display_name", u, req.UserId, displayName)
		}
		if err == db.ErrNotFound {
			return nil, errUserNotFound
		} else if err != nil {
			return nil, err
		}
	}

	if req.User.Username != nil {
		if err = user.AuthZProvider.Get().CanSetUsersUsername(*curUser, targetUser); err != nil {
			if ok, findErr := user.AuthZProvider.Get().CanGetUser(*curUser, targetUser); err != nil {
				return nil, findErr
			} else if !ok {
				return nil, errUserNotFound
			}
			return nil, err
		}

		// Reject SSO and SCIM
		if user.IsSSOUser(targetUser) {
			return nil, status.Error(codes.InvalidArgument,
				"Cannot change username of SSO/SCIM user through this API.")
		}

		username, err3 := clearUsername(targetUser, *req.User.Username, 2)
		if err3 != nil {
			return nil, err3
		}

		err = a.m.db.UpdateUsername(&targetUser.ID, *username)
		if err == db.ErrNotFound {
			return nil, errUserNotFound
		} else if err != nil {
			return nil, err
		}
	}

	var toUpdate []string
	if req.User.Active != nil {
		if err = user.AuthZProvider.Get().
			CanSetUsersActive(*curUser, targetUser, req.User.Active.Value); err != nil {
			if ok, canGetErr := user.AuthZProvider.Get().
				CanGetUser(*curUser, targetFullUser.ToUser()); canGetErr != nil {
				return nil, canGetErr
			} else if !ok {
				return nil, errUserNotFound
			}
			return nil, status.Error(codes.PermissionDenied, err.Error())
		}
		targetUser.Active = req.User.Active.Value
		toUpdate = append(toUpdate, "active")
	}

	if req.User.Admin != nil {
		if err = user.AuthZProvider.Get().CanSetUsersAdmin(*curUser, targetUser,
			req.User.Admin.Value); err != nil {
			if ok, canGetErr := user.AuthZProvider.Get().
				CanGetUser(*curUser, targetFullUser.ToUser()); canGetErr != nil {
				return nil, canGetErr
			} else if !ok {
				return nil, errUserNotFound
			}
			return nil, status.Error(codes.PermissionDenied, err.Error())
		}
		targetUser.Admin = req.User.Admin.Value
		toUpdate = append(toUpdate, "admin")
	}

	var ug *model.AgentUserGroup
	if aug := req.User.AgentUserGroup; aug != nil {
		if err = validateProtoAgentUserGroup(aug); err != nil {
			return nil, err
		}
		ug = &model.AgentUserGroup{
			UID:   int(*req.User.AgentUserGroup.AgentUid),
			GID:   int(*req.User.AgentUserGroup.AgentGid),
			User:  *req.User.AgentUserGroup.AgentUser,
			Group: *req.User.AgentUserGroup.AgentGroup,
		}
		if err = user.AuthZProvider.Get().
			CanSetUsersAgentUserGroup(*curUser, targetUser, *ug); err != nil {
			if ok, canGetErr := user.AuthZProvider.Get().
				CanGetUser(*curUser, targetFullUser.ToUser()); canGetErr != nil {
				return nil, canGetErr
			} else if !ok {
				return nil, errUserNotFound
			}
			return nil, status.Error(codes.PermissionDenied, err.Error())
		}
	}

	switch err = a.m.db.UpdateUser(&targetUser, toUpdate, ug); {
	case err == db.ErrNotFound:
		return nil, errUserNotFound
	case err != nil:
		return nil, err
	}

	fullUser, err := getUser(a.m.db, uid)
	return &apiv1.PatchUserResponse{User: fullUser}, err
}

func (a *apiServer) GetUserSetting(
	ctx context.Context, req *apiv1.GetUserSettingRequest,
) (*apiv1.GetUserSettingResponse, error) {
	curUser, _, err := grpcutil.GetUser(ctx)
	if err != nil {
		return nil, err
	}
	if err = user.AuthZProvider.Get().CanGetUsersOwnSettings(*curUser); err != nil {
		return nil, status.Error(codes.PermissionDenied, err.Error())
	}

	settings, err := db.GetUserSetting(curUser.ID)
	return &apiv1.GetUserSettingResponse{Settings: settings}, err
}

func (a *apiServer) PostUserSetting(
	ctx context.Context, req *apiv1.PostUserSettingRequest,
) (*apiv1.PostUserSettingResponse, error) {
	if req.Setting == nil {
		return nil, status.Error(codes.InvalidArgument, "must specify setting")
	}

	curUser, _, err := grpcutil.GetUser(ctx)
	if err != nil {
		return nil, err
	}
	settingModel := model.UserWebSetting{
		UserID:      curUser.ID,
		Key:         req.Setting.Key,
		Value:       req.Setting.Value,
		StoragePath: req.StoragePath,
	}
	if err = user.AuthZProvider.Get().CanCreateUsersOwnSetting(*curUser, settingModel); err != nil {
		return nil, status.Error(codes.PermissionDenied, err.Error())
	}

	err = db.UpdateUserSetting(&settingModel)
	return &apiv1.PostUserSettingResponse{}, err
}

func (a *apiServer) ResetUserSetting(
	ctx context.Context, req *apiv1.ResetUserSettingRequest,
) (*apiv1.ResetUserSettingResponse, error) {
	curUser, _, err := grpcutil.GetUser(ctx)
	if err != nil {
		return nil, err
	}
	if err = user.AuthZProvider.Get().CanResetUsersOwnSettings(*curUser); err != nil {
		return nil, status.Error(codes.PermissionDenied, err.Error())
	}

	err = db.ResetUserSetting(curUser.ID)
	return &apiv1.ResetUserSettingResponse{}, err
}
