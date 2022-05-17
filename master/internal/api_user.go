package internal

import (
	"context"
	"sort"

	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
	timestamppb "google.golang.org/protobuf/types/known/timestamppb"
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
	fullUser, err := getUser(a.m.db, model.UserID(req.UserId))
	return &apiv1.GetUserResponse{User: fullUser}, err
}

func (a *apiServer) PostUser(
	ctx context.Context, req *apiv1.PostUserRequest) (*apiv1.PostUserResponse, error) {
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

	switch err := a.m.db.AddUser(user, agentUserGroup); {
	case err == db.ErrDuplicateRecord:
		return nil, status.Error(codes.InvalidArgument, "user already exists")
	case err != nil:
		return nil, err
	}
	fullUser, err := getUser(a.m.db, model.UserID(req.User.Id))
	return &apiv1.PostUserResponse{User: fullUser}, err
}

func (a *apiServer) SetUserPassword(
	ctx context.Context, req *apiv1.SetUserPasswordRequest) (*apiv1.SetUserPasswordResponse, error) {
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
	ctx context.Context, req *apiv1.PatchUserRequest) (*apiv1.PatchUserResponse, error) {
	curUser, _, err := grpcutil.GetUser(ctx, a.m.db, &a.m.config.InternalConfig.ExternalSessions)
	if err != nil {
		return nil, err
	}
	if !curUser.Admin && curUser.ID != model.UserID(req.UserId) {
		return nil, grpcutil.ErrPermissionDenied
	}
	user := &model.User{ID: model.UserID(req.UserId)}
	// TODO: handle any field name:
	if req.User.DisplayName != "" {
		user.DisplayName = null.StringFrom(req.User.DisplayName)
		switch err = a.m.db.UpdateUser(user, []string{"display_name"}, nil); {
		case err == db.ErrNotFound:
			return nil, errUserNotFound
		case err != nil:
			return nil, err
		}
	}
	fullUser, err := getUser(a.m.db, model.UserID(req.UserId))
	return &apiv1.PatchUserResponse{User: fullUser}, err
}
