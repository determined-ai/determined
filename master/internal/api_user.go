package internal

import (
	"context"
	"fmt"
	"regexp"
	"strings"

	"gopkg.in/guregu/null.v3"

	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
	timestamppb "google.golang.org/protobuf/types/known/timestamppb"

	"github.com/determined-ai/determined/master/internal/db"
	"github.com/determined-ai/determined/master/internal/grpcutil"
	"github.com/determined-ai/determined/master/pkg/model"
	"github.com/determined-ai/determined/proto/pkg/apiv1"
	"github.com/determined-ai/determined/proto/pkg/userv1"
)

var errUserNotFound = status.Error(codes.NotFound, "user not found")

func toProtoUserFromFullUser(user model.FullUser) *userv1.User {
	var agentUserGroup *userv1.AgentUserGroup
	if user.AgentUID.Valid || user.AgentGID.Valid {
		agentUserGroup = &userv1.AgentUserGroup{
			AgentUid: int32(user.AgentUID.ValueOrZero()),
			AgentGid: int32(user.AgentGID.ValueOrZero()),
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

func getUser(d *db.PgDB, userID model.UserID) (*userv1.User, error) {
	user, err := d.UserByID(userID)
	switch {
	case err == db.ErrNotFound:
		return nil, errUserNotFound
	case err != nil:
		return nil, err
	}
	var protoAug *userv1.AgentUserGroup
	agentUserGroup, err := d.AgentUserGroup(user.ID)
	if agentUserGroup != nil {
		protoAug = &userv1.AgentUserGroup{
			AgentUid: int32(agentUserGroup.UID),
			AgentGid: int32(agentUserGroup.GID),
		}
	}
	displayNameString := user.DisplayName.ValueOrZero()
	return &userv1.User{
		Id:             int32(user.ID),
		Username:       user.Username,
		Admin:          user.Admin,
		Active:         user.Active,
		AgentUserGroup: protoAug,
		DisplayName:    displayNameString,
	}, err
}

func userShouldBeAdmin(ctx context.Context, a *apiServer) error {
	u, _, err := grpcutil.GetUser(ctx, a.m.db, &a.m.config.InternalConfig.ExternalSessions)
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
	resp := &apiv1.GetUsersResponse{}
	for _, user := range users {
		resp.Users = append(resp.Users, toProtoUserFromFullUser(user))
	}

	return resp, a.paginate(&resp.Pagination, &resp.Users, req.Offset, req.Limit)
}

func (a *apiServer) GetUser(
	_ context.Context, req *apiv1.GetUserRequest,
) (*apiv1.GetUserResponse, error) {
	fullUser, err := getUser(a.m.db, model.UserID(req.UserId))
	return &apiv1.GetUserResponse{User: fullUser}, err
}

func (a *apiServer) PostUser(
	ctx context.Context, req *apiv1.PostUserRequest,
) (*apiv1.PostUserResponse, error) {
	if err := userShouldBeAdmin(ctx, a); err != nil {
		return nil, err
	}

	if err := grpcutil.ValidateRequest(
		func() (bool, string) { return req.User != nil, "no user specified" },
		func() (bool, string) { return req.User.Username != "", "no username specified" },
	); err != nil {
		return nil, err
	}
	user := &model.User{
		Username: req.User.Username,
		Admin:    req.User.Admin,
		Active:   req.User.Active,
	}
	if req.User.DisplayName != "" {
		user.DisplayName = null.StringFrom(req.User.DisplayName)
	}
	if err := user.UpdatePasswordHash(replicateClientSideSaltAndHash(req.Password)); err != nil {
		return nil, err
	}
	var agentUserGroup *model.AgentUserGroup
	if req.User.AgentUserGroup != nil {
		agentUserGroup = &model.AgentUserGroup{
			UID: int(req.User.AgentUserGroup.AgentUid),
			GID: int(req.User.AgentUserGroup.AgentGid),
		}
	}

	userID, err := a.m.db.AddUser(user, agentUserGroup)
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
	curUser, _, err := grpcutil.GetUser(ctx, a.m.db, &a.m.config.InternalConfig.ExternalSessions)
	if err != nil {
		return nil, err
	}
	if !curUser.Admin && curUser.ID != model.UserID(req.UserId) {
		return nil, grpcutil.ErrPermissionDenied
	}
	user := &model.User{ID: model.UserID(req.UserId)}
	if err = user.UpdatePasswordHash(replicateClientSideSaltAndHash(req.Password)); err != nil {
		return nil, err
	}
	switch err = a.m.db.UpdateUser(user, []string{"password_hash"}, nil); {
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
	curUser, _, err := grpcutil.GetUser(ctx, a.m.db, &a.m.config.InternalConfig.ExternalSessions)
	if err != nil {
		return nil, err
	}
	uid := model.UserID(req.UserId)
	if !curUser.Admin && curUser.ID != uid {
		return nil, grpcutil.ErrPermissionDenied
	}
	// TODO: handle any field name:
	if req.User.DisplayName != nil {
		u := &userv1.User{}
		if req.User.DisplayName.Value == "" {
			// Disallow empty diaplay name for sorting purpose.
			err = a.m.db.QueryProto("set_user_display_name", u, req.UserId, nil)
		} else {
			// Remove non-ASCII chars to avoid hidden whitespace, confusable letters, etc.
			re := regexp.MustCompile("[^\\p{Latin}\\p{N}\\s]")
			displayName := re.ReplaceAllLiteralString(req.User.DisplayName.Value, "")
			// Restrict 'admin' and 'determined' in display names.
			if !(curUser.Admin && curUser.ID == uid) && strings.Contains(strings.ToLower(displayName),
				"admin") {
				return nil, status.Error(codes.InvalidArgument, "Non-admin user cannot be renamed 'admin'")
			}
			if curUser.Username != "determined" && strings.Contains(strings.ToLower(displayName),
				"determined") {
				return nil, status.Error(codes.InvalidArgument, "User cannot be renamed 'determined'")
			}
			err = a.m.db.QueryProto("set_user_display_name", u, req.UserId, strings.TrimSpace(displayName))
		}
		if err == db.ErrNotFound {
			return nil, errUserNotFound
		} else if err != nil {
			return nil, err
		}
	}

	fullUser, err := getUser(a.m.db, uid)
	return &apiv1.PatchUserResponse{User: fullUser}, err
}

func (a *apiServer) GetUserSetting(
	ctx context.Context, req *apiv1.GetUserSettingRequest,
) (*apiv1.GetUserSettingResponse, error) {
	user, _, err := grpcutil.GetUser(ctx, a.m.db, &a.m.config.InternalConfig.ExternalSessions)
	if err != nil {
		return nil, err
	}
	settings, err := db.GetUserSetting(user.ID)
	return &apiv1.GetUserSettingResponse{Settings: settings}, err
}

func (a *apiServer) PostUserSetting(
	ctx context.Context, req *apiv1.PostUserSettingRequest,
) (*apiv1.PostUserSettingResponse, error) {
	user, _, err := grpcutil.GetUser(ctx, a.m.db, &a.m.config.InternalConfig.ExternalSessions)
	if err != nil {
		return nil, err
	}
	settingModel := model.UserWebSetting{
		UserID:      user.ID,
		Key:         req.Setting.Key,
		Value:       req.Setting.Value,
		StoragePath: req.StoragePath,
	}
	err = db.UpdateUserSetting(&settingModel)
	return &apiv1.PostUserSettingResponse{}, err
}

func (a *apiServer) ResetUserSetting(
	ctx context.Context, req *apiv1.ResetUserSettingRequest,
) (*apiv1.ResetUserSettingResponse, error) {
	user, _, err := grpcutil.GetUser(ctx, a.m.db, &a.m.config.InternalConfig.ExternalSessions)
	if err != nil {
		return nil, err
	}
	err = db.ResetUserSetting(user.ID)
	return &apiv1.ResetUserSettingResponse{}, err
}
