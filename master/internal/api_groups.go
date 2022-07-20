package internal

import (
	"context"

	"github.com/uptrace/bun"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"

	"github.com/determined-ai/determined/master/internal/db"
	"github.com/determined-ai/determined/master/internal/usergroup"
	"github.com/determined-ai/determined/master/pkg/model"
	"github.com/determined-ai/determined/proto/pkg/apiv1"
	"github.com/determined-ai/determined/proto/pkg/groupv1"
)

// FIXME: look at how errors are handled in the rest of the API and follow that pattern
func (a *apiServer) CreateGroup(ctx context.Context, req *apiv1.CreateGroupRequest,
) (resp *apiv1.GroupWriteResponse, err error) {
	// Detect whether we're returning special errors and convert to gRPC error
	defer func() {
		err = mapAndFilterErrors(err)
	}()

	group := usergroup.Group{
		Name: req.Name,
	}

	createdGroup, err := usergroup.AddGroup(ctx, group)
	if err != nil {
		return nil, err
	}

	uids := intsToUserIDs(req.AddUsers)
	err = usergroup.AddUsersToGroup(ctx, createdGroup.ID, uids...)
	if err != nil {
		return nil, err
	}

	users, err := usergroup.GetUsersInGroup(ctx, createdGroup.ID)
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

func (a *apiServer) GetGroups(ctx context.Context, req *apiv1.GroupSearchRequest,
) (resp *apiv1.GroupSearchResponse, err error) {
	// Detect whether we're returning special errors and convert to gRPC error
	defer func() {
		err = mapAndFilterErrors(err)
	}()

	groups, err := usergroup.SearchGroups(ctx, req.Name, model.UserID(req.UserId))
	if err != nil {
		return nil, err
	}

	return &apiv1.GroupSearchResponse{
		Groups: usergroup.Groups(groups).Proto(),
	}, nil
}

func (a *apiServer) GetGroup(ctx context.Context, req *apiv1.GetGroupRequest,
) (resp *apiv1.GetGroupResponse, err error) {
	// Detect whether we're returning special errors and convert to gRPC error
	defer func() {
		err = mapAndFilterErrors(err)
	}()

	gid := int(req.GroupId)
	g, err := usergroup.GroupByID(ctx, gid)
	if err != nil {
		return nil, err
	}

	users, err := usergroup.GetUsersInGroup(ctx, gid)
	if err != nil {
		return nil, err
	}

	gProto := g.Proto()
	gDetail := groupv1.GroupDetails{
		GroupId: gProto.GroupId,
		Name:    gProto.Name,
		Users:   model.Users(users).Proto(),
	}

	return &apiv1.GetGroupResponse{
		Group: &gDetail,
	}, nil
}

func (a *apiServer) UpdateGroup(ctx context.Context, req *apiv1.UpdateGroupRequest,
) (resp *apiv1.GroupWriteResponse, err error) {
	// Detect whether we're returning special errors and convert to gRPC error
	defer func() {
		err = mapAndFilterErrors(err)
	}()

	oldGroup, err := usergroup.GroupByID(ctx, int(req.GetGroupId()))
	if err != nil {
		return nil, err
	}

	newName := oldGroup.Name
	if req.GetName() != "" {
		newName = req.GetName()
	}
	err = usergroup.UpdateGroup(ctx, usergroup.Group{
		BaseModel: bun.BaseModel{},
		ID:        int(req.GetGroupId()),
		Name:      newName,
		OwnerID:   oldGroup.OwnerID,
	})
	if err != nil {
		return nil, err
	}

	if len(req.GetAddUsers()) > 0 {
		var users []model.UserID
		for _, id := range req.GetAddUsers() {
			users = append(users, model.UserID(id))
		}
		err := usergroup.AddUsersToGroup(ctx, int(req.GetGroupId()), users...)
		if err != nil {
			return nil, err
		}
	}

	if len(req.GetRemoveUsers()) > 0 {
		var users []model.UserID
		for _, id := range req.GetRemoveUsers() {
			users = append(users, model.UserID(id))
		}
		err := usergroup.RemoveUsersFromGroup(ctx, int(req.GetGroupId()), users...)
		if err != nil {
			return nil, err
		}
	}

	users, err := usergroup.GetUsersInGroup(ctx, int(req.GetGroupId()))
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

func (a *apiServer) DeleteGroup(ctx context.Context, req *apiv1.DeleteGroupRequest,
) (resp *apiv1.DeleteGroupResponse, err error) {
	// Detect whether we're returning special errors and convert to gRPC error
	defer func() {
		err = mapAndFilterErrors(err)
	}()

	err = usergroup.DeleteGroup(ctx, int(req.GetGroupId()))
	if err != nil {
		return nil, err
	}
	return resp, nil
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
	errorWhitelist     = map[error]bool{
		nil:                true,
		errBadRequest:      true,
		errNotFound:        true,
		errDuplicateRecord: true,
	}
)

func mapAndFilterErrors(err error) error {
	if whitelisted := errorWhitelist[err]; whitelisted {
		return err
	}

	switch err {
	case db.ErrNotFound:
		return errNotFound
	case db.ErrDuplicateRecord:
		return errDuplicateRecord
	}

	return status.Error(codes.Internal, "Internal server error")
}
