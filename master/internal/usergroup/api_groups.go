package usergroup

import (
	"context"

	"github.com/uptrace/bun"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"

	"github.com/determined-ai/determined/master/internal/db"
	"github.com/determined-ai/determined/master/pkg/model"
	"github.com/determined-ai/determined/proto/pkg/apiv1"
	"github.com/determined-ai/determined/proto/pkg/groupv1"
)

type ApiServer struct{}

func (a *ApiServer) CreateGroup(ctx context.Context, req *apiv1.CreateGroupRequest,
) (resp *apiv1.GroupWriteResponse, err error) {
	// Detect whether we're returning special errors and convert to gRPC error
	defer func() {
		err = mapAndFilterErrors(err)
	}()

	group := Group{
		Name: req.Name,
	}

	createdGroup, err := AddGroup(ctx, group)
	if err != nil {
		return nil, err
	}

	uids := intsToUserIDs(req.AddUsers)
	err = AddUsersToGroup(ctx, createdGroup.ID, uids...)
	if err != nil {
		return nil, err
	}

	users, err := UsersInGroup(ctx, createdGroup.ID)
	if err != nil {
		return nil, err
	}

	return &apiv1.GroupWriteResponse{
		Group: &groupv1.GroupDetails{
			GroupId: int32(createdGroup.ID),
			Name:    createdGroup.Name,
			Users:   model.Users(users).Proto(),
		},
	}, nil
}

func (a *ApiServer) GetGroups(ctx context.Context, req *apiv1.GroupSearchRequest,
) (resp *apiv1.GroupSearchResponse, err error) {
	// Detect whether we're returning special errors and convert to gRPC error
	defer func() {
		err = mapAndFilterErrors(err)
	}()

	groups, err := SearchGroups(ctx, req.Name, model.UserID(req.UserId), int(req.Offset),
		int(req.Limit))
	if err != nil {
		return nil, err
	}

	return &apiv1.GroupSearchResponse{
		Groups: Groups(groups).Proto(),
	}, nil
}

func (a *ApiServer) GetGroup(ctx context.Context, req *apiv1.GetGroupRequest,
) (resp *apiv1.GetGroupResponse, err error) {
	// Detect whether we're returning special errors and convert to gRPC error
	defer func() {
		err = mapAndFilterErrors(err)
	}()

	gid := int(req.GroupId)
	g, err := GroupByID(ctx, gid)
	if err != nil {
		return nil, err
	}

	users, err := UsersInGroup(ctx, gid)
	if err != nil {
		return nil, err
	}

	gDetail := groupv1.GroupDetails{
		GroupId: int32(g.ID),
		Name:    g.Name,
		Users:   model.Users(users).Proto(),
	}

	return &apiv1.GetGroupResponse{
		Group: &gDetail,
	}, nil
}

func (a *ApiServer) UpdateGroup(ctx context.Context, req *apiv1.UpdateGroupRequest,
) (resp *apiv1.GroupWriteResponse, err error) {
	// Detect whether we're returning special errors and convert to gRPC error
	defer func() {
		err = mapAndFilterErrors(err)
	}()

	oldGroup, err := GroupByID(ctx, int(req.GroupId))
	if err != nil {
		return nil, err
	}

	newName := oldGroup.Name
	if req.Name != "" {
		newName = req.Name
	}
	err = UpdateGroup(ctx, Group{
		BaseModel: bun.BaseModel{},
		ID:        int(req.GroupId),
		Name:      newName,
		OwnerID:   oldGroup.OwnerID,
	})
	if err != nil {
		return nil, err
	}

	if len(req.AddUsers) > 0 {
		var users []model.UserID
		for _, id := range req.AddUsers {
			users = append(users, model.UserID(id))
		}
		err = AddUsersToGroup(ctx, int(req.GroupId), users...)
		if err != nil {
			return nil, err
		}
	}

	if len(req.RemoveUsers) > 0 {
		var users []model.UserID
		for _, id := range req.RemoveUsers {
			users = append(users, model.UserID(id))
		}
		err = RemoveUsersFromGroup(ctx, int(req.GroupId), users...)
		if err != nil {
			return nil, err
		}
	}

	users, err := UsersInGroup(ctx, int(req.GroupId))
	if err != nil {
		return nil, err
	}

	resp = &apiv1.GroupWriteResponse{
		Group: &groupv1.GroupDetails{
			GroupId: int32(oldGroup.ID),
			Name:    newName,
			Users:   model.Users(users).Proto(),
		},
	}

	return resp, nil
}

func (a *ApiServer) DeleteGroup(ctx context.Context, req *apiv1.DeleteGroupRequest,
) (resp *apiv1.DeleteGroupResponse, err error) {
	// Detect whether we're returning special errors and convert to gRPC error
	defer func() {
		err = mapAndFilterErrors(err)
	}()

	err = DeleteGroup(ctx, int(req.GroupId))
	if err != nil {
		return nil, err
	}
	return &apiv1.DeleteGroupResponse{}, nil
}

func intsToUserIDs(ints []int32) []model.UserID {
	ids := make([]model.UserID, len(ints))

	for i := range ints {
		ids[i] = model.UserID(ints[i])
	}

	return ids
}

var (
	errBadRequest      = status.Error(codes.InvalidArgument, "Bad request")
	errNotFound        = status.Error(codes.NotFound, "Not found")
	errDuplicateRecord = status.Error(codes.AlreadyExists, "Duplicate record")
	errInternal        = status.Error(codes.Internal, "Internal server error")
	errPassthroughMap  = map[error]bool{
		nil:                true,
		errBadRequest:      true,
		errNotFound:        true,
		errDuplicateRecord: true,
		errInternal:        true,
	}
)

func mapAndFilterErrors(err error) error {
	if whitelisted := errPassthroughMap[err]; whitelisted {
		return err
	}

	switch err {
	case db.ErrNotFound:
		return errNotFound
	case db.ErrDuplicateRecord:
		return errDuplicateRecord
	}

	return errInternal
}
