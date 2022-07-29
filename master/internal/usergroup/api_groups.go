package usergroup

import (
	"context"
	"errors"

	"github.com/sirupsen/logrus"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"

	"github.com/determined-ai/determined/master/internal/db"
	"github.com/determined-ai/determined/master/pkg/model"
	"github.com/determined-ai/determined/proto/pkg/apiv1"
	"github.com/determined-ai/determined/proto/pkg/groupv1"
)

// APIServer is an embedded api server struct.
type APIServer struct{}

// CreateGroup creates a group and adds members to it, if any.
func (a *APIServer) CreateGroup(ctx context.Context, req *apiv1.CreateGroupRequest,
) (resp *apiv1.GroupWriteResponse, err error) {
	// Detect whether we're returning special errors and convert to gRPC error
	defer func() {
		err = mapAndFilterErrors(err)
	}()

	group := Group{
		Name: req.Name,
	}
	uids := intsToUserIDs(req.AddUsers)

	createdGroup, users, err := AddGroupWithMembers(ctx, group, uids...)
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

// GetGroups searches for groups that fulfills the criteria given by the user.
func (a *APIServer) GetGroups(ctx context.Context, req *apiv1.GetGroupsRequest,
) (resp *apiv1.GetGroupsResponse, err error) {
	// Detect whether we're returning special errors and convert to gRPC error
	defer func() {
		err = mapAndFilterErrors(err)
	}()

	if req.Limit > maxLimit || req.Limit == 0 {
		return nil, errInvalidLimit
	}

	groups, memberCounts, tableCount, err := SearchGroups(ctx,
		req.Name, model.UserID(req.UserId), int(req.Offset), int(req.Limit))
	if err != nil {
		return nil, err
	}

	searchResults := make([]*groupv1.GroupSearchResult, len(groups))
	for i, g := range groups {
		searchResults[i] = &groupv1.GroupSearchResult{
			Group:      g.Proto(),
			NumMembers: memberCounts[i],
		}
	}

	return &apiv1.GetGroupsResponse{
		Groups: searchResults,
		Pagination: &apiv1.Pagination{
			Offset:     req.Offset,
			Limit:      req.Limit,
			StartIndex: req.Offset,
			EndIndex:   req.Offset + int32(len(groups)),
			Total:      int32(tableCount),
		},
	}, nil
}

// GetGroup finds and returns details of the group specified.
func (a *APIServer) GetGroup(ctx context.Context, req *apiv1.GetGroupRequest,
) (resp *apiv1.GetGroupResponse, err error) {
	// Detect whether we're returning special errors and convert to gRPC error
	defer func() {
		err = mapAndFilterErrors(err)
	}()

	gid := int(req.GroupId)
	g, err := GroupByIDTx(ctx, nil, gid)
	if err != nil {
		return nil, err
	}

	users, err := UsersInGroupTx(ctx, nil, gid)
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

// UpdateGroup updates the group and returns the newly updated group details.
func (a *APIServer) UpdateGroup(ctx context.Context, req *apiv1.UpdateGroupRequest,
) (resp *apiv1.GroupWriteResponse, err error) {
	// Detect whether we're returning special errors and convert to gRPC error
	defer func() {
		err = mapAndFilterErrors(err)
	}()

	var addUsers []model.UserID
	var removeUsers []model.UserID

	if len(req.AddUsers) > 0 {
		addUsers = intsToUserIDs(req.AddUsers)
	}

	if len(req.RemoveUsers) > 0 {
		removeUsers = intsToUserIDs(req.RemoveUsers)
	}

	users, newName, err := UpdateGroupAndMembers(ctx,
		int(req.GroupId), req.Name, addUsers, removeUsers)
	if err != nil {
		return nil, err
	}

	resp = &apiv1.GroupWriteResponse{
		Group: &groupv1.GroupDetails{
			GroupId: req.GroupId,
			Name:    newName,
			Users:   model.Users(users).Proto(),
		},
	}

	return resp, nil
}

// DeleteGroup deletes the database entry for the group.
func (a *APIServer) DeleteGroup(ctx context.Context, req *apiv1.DeleteGroupRequest,
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

const (
	maxLimit = 500
)

var (
	errBadRequest   = status.Error(codes.InvalidArgument, "Bad request")
	errInvalidLimit = status.Errorf(codes.InvalidArgument,
		"Bad request: limit is required and must be <= %d", maxLimit)
	errNotFound        = status.Error(codes.NotFound, "Not found")
	errDuplicateRecord = status.Error(codes.AlreadyExists, "Duplicate record")
	errInternal        = status.Error(codes.Internal, "Internal server error")
	errPassthroughMap  = map[error]bool{
		nil:                true,
		errBadRequest:      true,
		errInvalidLimit:    true,
		errNotFound:        true,
		errDuplicateRecord: true,
		errInternal:        true,
	}
)

func mapAndFilterErrors(err error) error {
	// FIXME: whitelist might not work.
	if whitelisted := errPassthroughMap[err]; whitelisted {
		return err
	}

	switch {
	case errors.Is(err, db.ErrNotFound):
		return status.Error(codes.NotFound, err.Error())
	case errors.Is(err, db.ErrDuplicateRecord):
		return status.Error(codes.AlreadyExists, err.Error())
	}

	logrus.WithError(err).Debug("suppressing error at API boundary")

	return errInternal // TODO: delete comment: deliberately don't wrap this error
}
