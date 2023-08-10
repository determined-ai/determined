package internal

import (
	"context"
	"fmt"
	"regexp"
	"strings"
	"time"

	"github.com/pkg/errors"
	"gopkg.in/guregu/null.v3"

	bun "github.com/uptrace/bun"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
	timestamppb "google.golang.org/protobuf/types/known/timestamppb"

	"github.com/determined-ai/determined/master/internal/api"
	"github.com/determined-ai/determined/master/internal/authz"
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
	errExternalSessions = status.Error(codes.PermissionDenied, "not enabled with external sessions")
	latinText           = regexp.MustCompile("[^[:graph:]\\s]")
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
		Remote:         user.Remote,
		AgentUserGroup: agentUserGroup,
		DisplayName:    displayNameString,
		ModifiedAt:     timestamppb.New(user.ModifiedAt),
	}
}

func getFullModelUserByUsername(username string) (*model.FullUser, error) {
	userModel, err := user.UserByUsername(username)
	if errors.Is(err, db.ErrNotFound) {
		return nil, api.NotFoundErrs("user", "", true)
	}
	fullUser, err := user.UserByID(userModel.ID)
	return fullUser, err
}

func getFullModelUser(userID model.UserID) (*model.FullUser, error) {
	userModel, err := user.UserByID(userID)
	if errors.Is(err, db.ErrNotFound) {
		return nil, api.NotFoundErrs("user", "", true)
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
		apiv1.GetUsersRequest_SORT_BY_NAME:          "name",
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
	nameFilterExpr := "%" + req.Name + "%"
	selectExpr := `
		SELECT
			u.id, u.display_name, u.username, u.admin, u.active, u.modified_at, u.remote,
			h.uid AS agent_uid, h.gid AS agent_gid, h.user_ AS agent_user, h.group_ AS agent_group, 
			COALESCE(u.display_name, u.username) AS name
		FROM users u
			LEFT OUTER JOIN agent_user_groups h ON (u.id = h.user_id)
		WHERE ((? = '') OR u.display_name ILIKE ? OR u.username ILIKE ?)
	`
	query := selectExpr + fmt.Sprintf(" ORDER BY %s", orderExpr)
	err := db.Bun().NewRaw(query,
		req.Name, nameFilterExpr, nameFilterExpr).Scan(context.Background(), &users)
	if err != nil {
		return nil, err
	}

	curUser, _, err := grpcutil.GetUser(ctx)
	if err != nil {
		return nil, err
	}
	if users, err = user.AuthZProvider.Get().FilterUserList(ctx, *curUser, users); err != nil {
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
	if err = user.AuthZProvider.Get().CanGetUser(
		ctx, *curUser, targetFullUser.ToUser()); err != nil {
		return nil, authz.SubIfUnauthorized(err, api.NotFoundErrs("user", "", true))
	}
	return &apiv1.GetUserResponse{User: toProtoUserFromFullUser(*targetFullUser)}, nil
}

func (a *apiServer) GetMe(
	ctx context.Context, req *apiv1.GetMeRequest,
) (*apiv1.GetMeResponse, error) {
	curUser, _, err := grpcutil.GetUser(ctx)
	if err != nil {
		return nil, err
	}
	curFullUser, err := getFullModelUser(curUser.ID)
	if err != nil {
		return nil, err
	}
	return &apiv1.GetMeResponse{User: toProtoUserFromFullUser(*curFullUser)}, err
}

func (a *apiServer) GetUserByUsername(
	ctx context.Context, req *apiv1.GetUserByUsernameRequest,
) (*apiv1.GetUserByUsernameResponse, error) {
	curUser, _, err := grpcutil.GetUser(ctx)
	if err != nil {
		return nil, err
	}
	targetFullUser, err := getFullModelUserByUsername(req.Username)
	if err != nil {
		return nil, err
	}

	if err = user.AuthZProvider.Get().CanGetUser(ctx, *curUser, targetFullUser.ToUser()); err != nil {
		return nil, authz.SubIfUnauthorized(err, api.NotFoundErrs("user", "", true))
	}
	return &apiv1.GetUserByUsernameResponse{User: toProtoUserFromFullUser(*targetFullUser)}, nil
}

func (a *apiServer) PostUser(
	ctx context.Context, req *apiv1.PostUserRequest,
) (*apiv1.PostUserResponse, error) {
	if a.m.config.InternalConfig.ExternalSessions.Enabled() {
		return nil, errExternalSessions
	}
	if req.User == nil {
		return nil, status.Error(codes.InvalidArgument, "must specify user to create")
	}
	if req.Password != "" && req.User.Remote {
		return nil, status.Error(codes.InvalidArgument, "cannot set password for remote user")
	}

	userToAdd := &model.User{
		Username: req.User.Username,
		Admin:    req.User.Admin,
		Active:   req.User.Active,
		Remote:   req.User.Remote,
	}
	clearedUsername, err := clearUsername(*userToAdd, userToAdd.Username, 2)
	if err != nil {
		return nil, err
	}
	userToAdd.Username = *clearedUsername

	if req.User.DisplayName != "" {
		clearedDisplayName, err := clearUsername(*userToAdd, req.User.DisplayName, 0)
		if err != nil {
			return nil, err
		}
		userToAdd.DisplayName = null.StringFrom(*clearedDisplayName)
	}

	var agentUserGroup *model.AgentUserGroup
	if req.User.AgentUserGroup != nil {
		aug := req.User.AgentUserGroup
		if agentUserGroup, err = model.AgentUserGroupFromProto(aug); err != nil {
			return nil, status.Error(
				codes.InvalidArgument,
				err.Error(),
			)
		}
	}

	curUser, _, err := grpcutil.GetUser(ctx)
	if err != nil {
		return nil, err
	}
	if err = user.AuthZProvider.Get().
		CanCreateUser(ctx, *curUser, *userToAdd, agentUserGroup); err != nil {
		return nil, status.Error(codes.PermissionDenied, err.Error())
	}

	if err = grpcutil.ValidateRequest(
		func() (bool, string) { return req.User != nil, "no user specified" },
		func() (bool, string) { return req.User.Username != "", "no username specified" },
	); err != nil {
		return nil, err
	}

	if req.User.Remote {
		userToAdd.PasswordHash = model.NoPasswordLogin
	} else {
		var hashedPassword string
		if req.IsHashed {
			hashedPassword = req.Password
		} else {
			hashedPassword = replicateClientSideSaltAndHash(req.Password)
		}

		if err = userToAdd.UpdatePasswordHash(hashedPassword); err != nil {
			return nil, err
		}
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
	if a.m.config.InternalConfig.ExternalSessions.Enabled() {
		return nil, errExternalSessions
	}
	curUser, _, err := grpcutil.GetUser(ctx)
	if err != nil {
		return nil, err
	}

	targetFullUser, err := getFullModelUser(model.UserID(req.UserId))
	if err != nil {
		return nil, err
	}
	targetUser := targetFullUser.ToUser()
	if err = user.AuthZProvider.Get().CanSetUsersPassword(ctx, *curUser, targetUser); err != nil {
		if canGetErr := user.AuthZProvider.
			Get().CanGetUser(ctx, *curUser, targetFullUser.ToUser()); canGetErr != nil {
			return nil, authz.SubIfUnauthorized(canGetErr, api.NotFoundErrs("user", "", true))
		}
		return nil, status.Error(codes.PermissionDenied, err.Error())
	}

	if err = targetUser.UpdatePasswordHash(replicateClientSideSaltAndHash(req.Password)); err != nil {
		return nil, err
	}
	switch err = a.m.db.UpdateUser(&targetUser, []string{"password_hash"}, nil); {
	case err == db.ErrNotFound:
		return nil, api.NotFoundErrs("user", "", true)
	case err != nil:
		return nil, err
	}
	fullUser, err := getUser(a.m.db, model.UserID(req.UserId))
	return &apiv1.SetUserPasswordResponse{User: fullUser}, err
}

func (a *apiServer) PatchUser(
	ctx context.Context, req *apiv1.PatchUserRequest,
) (*apiv1.PatchUserResponse, error) {
	if a.m.config.InternalConfig.ExternalSessions.Enabled() {
		return nil, errExternalSessions
	}
	if req.User == nil {
		return nil, status.Error(codes.InvalidArgument, "must provide user")
	}

	curUser, _, err := grpcutil.GetUser(ctx)
	if err != nil {
		return nil, err
	}

	targetFullUser, err := getFullModelUser(model.UserID(req.UserId))
	if err != nil {
		return nil, err
	}
	targetUser := targetFullUser.ToUser()
	if err = user.AuthZProvider.Get().CanGetUser(ctx, *curUser, targetUser); err != nil {
		return nil, authz.SubIfUnauthorized(err, api.NotFoundErrs("user", "", true))
	}

	updatedUser := &model.User{ID: targetUser.ID}
	var insertColumns []string
	if req.User.Admin != nil {
		if err = user.AuthZProvider.Get().
			CanSetUsersAdmin(ctx, *curUser, targetUser, req.User.Admin.Value); err != nil {
			return nil, status.Error(codes.PermissionDenied, err.Error())
		}

		updatedUser.Admin = req.User.Admin.Value
		insertColumns = append(insertColumns, "admin")
	}

	if req.User.Remote != nil {
		if err = user.AuthZProvider.Get().
			CanSetUsersRemote(ctx, *curUser); err != nil {
			return nil, status.Error(codes.PermissionDenied, err.Error())
		}

		updatedUser.Remote = *req.User.Remote
		insertColumns = append(insertColumns, "remote")
	}

	if req.User.Active != nil {
		if err = user.AuthZProvider.Get().
			CanSetUsersActive(ctx, *curUser, targetUser, req.User.Active.Value); err != nil {
			return nil, status.Error(codes.PermissionDenied, err.Error())
		}

		updatedUser.Active = req.User.Active.Value
		insertColumns = append(insertColumns, "active")
	}

	if req.User.Username != nil && *req.User.Username != targetUser.Username {
		if err = user.AuthZProvider.Get().CanSetUsersUsername(ctx, *curUser, targetUser); err != nil {
			return nil, status.Error(codes.PermissionDenied, err.Error())
		}

		if targetUser.Remote {
			return nil, status.Error(codes.InvalidArgument, "Cannot set username for remote users")
		}

		username, err := clearUsername(targetUser, *req.User.Username, 2)
		if err != nil {
			return nil, err
		}

		updatedUser.Username = *username
		insertColumns = append(insertColumns, "username")
	}

	if req.User.DisplayName != nil && *req.User.DisplayName != targetUser.DisplayName.ValueOrZero() {
		if err = user.AuthZProvider.Get().
			CanSetUsersDisplayName(ctx, *curUser, targetUser); err != nil {
			return nil, status.Error(codes.PermissionDenied, err.Error())
		}

		displayName, err := clearUsername(targetUser, *req.User.DisplayName, 0)
		if err != nil {
			return nil, err
		}

		if *displayName != "" {
			lowerDisplayName := strings.ToLower(*displayName)
			if ok, err := db.Bun().NewSelect().Model(&model.User{}).
				WhereGroup(" AND ", func(q *bun.SelectQuery) *bun.SelectQuery {
					return q.WhereOr("LOWER(username) = ?", lowerDisplayName).
						WhereOr("LOWER(display_name) = ?", lowerDisplayName)
				}).Where("id != ?", targetUser.ID).
				Exists(ctx); err != nil {
				return nil, errors.Wrap(err, "error finding similar display names")
			} else if ok {
				return nil, status.Errorf(codes.InvalidArgument, "can not change display name "+
					"to %s found a similar username or display name", *displayName)
			}
			updatedUser.DisplayName = null.StringFromPtr(displayName)
		}
		insertColumns = append(insertColumns, "display_name")
	}

	if req.User.Password != nil {
		if err = user.AuthZProvider.Get().
			CanSetUsersPassword(ctx, *curUser, targetUser); err != nil {
			return nil, status.Error(codes.PermissionDenied, err.Error())
		}

		if targetUser.Remote {
			return nil, status.Error(codes.InvalidArgument, "Cannot set password for remote users")
		}

		hashedPassword := *req.User.Password
		if !req.User.IsHashed {
			hashedPassword = replicateClientSideSaltAndHash(hashedPassword)
		}
		if err := updatedUser.UpdatePasswordHash(hashedPassword); err != nil {
			return nil, errors.Wrap(err, "error hashing password")
		}
		insertColumns = append(insertColumns, "password_hash")
	}

	var ug *model.AgentUserGroup
	if aug := req.User.AgentUserGroup; aug != nil {
		if ug, err = model.AgentUserGroupFromProto(aug); err != nil {
			return nil, status.Error(
				codes.InvalidArgument,
				err.Error(),
			)
		}

		if err = user.AuthZProvider.Get().
			CanSetUsersAgentUserGroup(ctx, *curUser, targetUser, *ug); err != nil {
			return nil, status.Error(codes.PermissionDenied, err.Error())
		}
	}

	if err := a.m.db.UpdateUser(updatedUser, insertColumns, ug); err != nil {
		return nil, err
	}

	fullUser, err := getUser(a.m.db, model.UserID(req.UserId))
	return &apiv1.PatchUserResponse{User: fullUser}, err
}

func (a *apiServer) GetUserSetting(
	ctx context.Context, req *apiv1.GetUserSettingRequest,
) (*apiv1.GetUserSettingResponse, error) {
	curUser, _, err := grpcutil.GetUser(ctx)
	if err != nil {
		return nil, err
	}
	if err = user.AuthZProvider.Get().CanGetUsersOwnSettings(ctx, *curUser); err != nil {
		return nil, status.Error(codes.PermissionDenied, err.Error())
	}

	settings, err := db.GetUserSetting(curUser.ID)
	return &apiv1.GetUserSettingResponse{Settings: settings}, err
}

func (a *apiServer) PostUserSetting(
	ctx context.Context, req *apiv1.PostUserSettingRequest,
) (*apiv1.PostUserSettingResponse, error) {
	if a.m.config.InternalConfig.ExternalSessions.Enabled() {
		return nil, errExternalSessions
	}
	if req.Settings == nil {
		req.Settings = make([]*userv1.UserWebSetting, 0)
	}

	curUser, _, err := grpcutil.GetUser(ctx)
	if err != nil {
		return nil, err
	}
	settingsModel := make([]*model.UserWebSetting, 0, len(req.Settings))
	for _, setting := range req.Settings {
		settingsModel = append(settingsModel, &model.UserWebSetting{
			UserID:      curUser.ID,
			Key:         setting.Key,
			Value:       setting.Value,
			StoragePath: setting.StoragePath,
		})
	}
	if err = user.AuthZProvider.Get().CanCreateUsersOwnSetting(
		ctx, *curUser, settingsModel); err != nil {
		return nil, status.Error(codes.PermissionDenied, err.Error())
	}

	err = db.UpdateUserSetting(settingsModel)
	return &apiv1.PostUserSettingResponse{}, err
}

func (a *apiServer) ResetUserSetting(
	ctx context.Context, req *apiv1.ResetUserSettingRequest,
) (*apiv1.ResetUserSettingResponse, error) {
	curUser, _, err := grpcutil.GetUser(ctx)
	if err != nil {
		return nil, err
	}
	if err = user.AuthZProvider.Get().CanResetUsersOwnSettings(ctx, *curUser); err != nil {
		return nil, status.Error(codes.PermissionDenied, err.Error())
	}

	err = db.ResetUserSetting(curUser.ID)
	return &apiv1.ResetUserSettingResponse{}, err
}

func (a *apiServer) PostUserActivity(
	ctx context.Context, req *apiv1.PostUserActivityRequest,
) (*apiv1.PostUserActivityResponse, error) {
	curUser, _, err := grpcutil.GetUser(ctx)
	if err != nil {
		return nil, err
	}

	timestamp := time.Now()
	if _, err := db.Bun().NewInsert().Model(model.UserActivityFromProto(
		req.ActivityType, req.EntityType, req.EntityId, int32(curUser.ID), timestamp,
	)).On("CONFLICT (user_id, activity_type, entity_type, entity_id) DO UPDATE").
		Set("activity_time = ?", timestamp).
		Exec(ctx); err != nil {
		return nil, err
	}
	return &apiv1.PostUserActivityResponse{}, err
}
