package internal

import (
	"context"
	"sort"

	"github.com/pkg/errors"

	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
	wrapperspb "google.golang.org/protobuf/types/known/wrapperspb"
	"gopkg.in/guregu/null.v3"

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
	}
}

func getUser(d *db.PgDB, username string) (*userv1.User, error) {
	user, err := d.UserByUsername(username)
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

func (a *apiServer) GetUsers(
	context.Context, *apiv1.GetUsersRequest) (*apiv1.GetUsersResponse, error) {
	users, err := a.m.db.UserList()
	if err != nil {
		return nil, err
	}
	result := &apiv1.GetUsersResponse{}
	for _, user := range users {
		result.Users = append(result.Users, toProtoUserFromFullUser(user))
	}
	sort.Slice(result.Users, func(i, j int) bool {
		return result.Users[i].Username < result.Users[j].Username
	})
	return result, nil
}

func (a *apiServer) GetUser(
	_ context.Context, req *apiv1.GetUserRequest) (*apiv1.GetUserResponse, error) {
	fullUser, err := getUser(a.m.db, req.Username)
	return &apiv1.GetUserResponse{User: fullUser}, err
}

func (a *apiServer) PostUser(
	ctx context.Context, req *apiv1.PostUserRequest) (*apiv1.PostUserResponse, error) {
	curUser, _, err := grpcutil.GetUser(ctx, a.m.db, &a.m.config.InternalConfig.ExternalSessions)
	if err != nil {
		return nil, err
	}
	if !curUser.Admin {
		return nil, grpcutil.ErrPermissionDenied
	}
	if err = grpcutil.ValidateRequest(
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
	if err = user.UpdatePasswordHash(req.Password); err != nil {
		return nil, err
	}
	var agentUserGroup *model.AgentUserGroup
	if req.User.AgentUserGroup != nil {
		agentUserGroup = &model.AgentUserGroup{
			UID: int(req.User.AgentUserGroup.AgentUid),
			GID: int(req.User.AgentUserGroup.AgentGid),
		}
	}

	switch err = a.m.db.AddUser(user, agentUserGroup); {
	case err == db.ErrDuplicateRecord:
		return nil, status.Error(codes.InvalidArgument, "user already exists")
	case err != nil:
		return nil, err
	}
	fullUser, err := getUser(a.m.db, req.User.Username)
	return &apiv1.PostUserResponse{User: fullUser}, err
}

func (a *apiServer) SetUserPassword(
	ctx context.Context, req *apiv1.SetUserPasswordRequest) (*apiv1.SetUserPasswordResponse, error) {
	// TODO if ExternalSessions is there, don't even allow this
	passwordHash := replicateClientSideSaltAndHash(req.Password)
	getResp, err := a.PatchUser(ctx, &apiv1.PatchUserRequest{Username: req.Username,
		User: &userv1.PatchUser{Password: &wrapperspb.StringValue{Value: passwordHash}}})
	if err != nil {
		return nil, err
	}
	return &apiv1.SetUserPasswordResponse{User: getResp.User}, err
}
func (a *apiServer) PatchUser(
	ctx context.Context, req *apiv1.PatchUserRequest) (*apiv1.PatchUserResponse, error) {
	curUser, _, err := grpcutil.GetUser(ctx, a.m.db, &a.m.config.InternalConfig.ExternalSessions)
	if err != nil {
		return nil, err
	}
	if !curUser.Admin && curUser.Username != req.Username {
		return nil, grpcutil.ErrPermissionDenied
	}
	// TODO: handle any field name:
	updateUser := &model.User{Username: req.Username}
	if req.User.DisplayName != "" {
		updateUser.DisplayName = null.StringFrom(req.User.DisplayName)
	}
	if req.User.Password != nil {
		if err = updateUser.UpdatePasswordHash(req.User.Password.Value); err != nil {
			return nil, err
		}
	}
	// return the finalUser object so we can be assured only public fields are passed on
	finalUser := &userv1.User{}
	err = a.m.db.QueryProto(
		"update_user", finalUser, updateUser.Username, updateUser.DisplayName, updateUser.PasswordHash)
	return &apiv1.PatchUserResponse{User: finalUser},
		errors.Wrapf(err, "error updating user \"%s\" in database", finalUser.Username)
}
