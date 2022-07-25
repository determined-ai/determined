package usergroup

import (
	"context"

	"github.com/sirupsen/logrus"
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

func (a *ApiServer) GetGroups(ctx context.Context, req *apiv1.GroupSearchRequest,
) (resp *apiv1.GroupSearchResponse, err error) {
	// Detect whether we're returning special errors and convert to gRPC error
	defer func() {
		err = mapAndFilterErrors(err)
	}()

	if req.Limit > maxLimit || req.Limit == 0 {
		return nil, errInvalidLimit
	}

	groups, count, err := SearchGroups(ctx, req.Name, model.UserID(req.UserId), int(req.Offset),
		int(req.Limit))
	if err != nil {
		return nil, err
	}

	return &apiv1.GroupSearchResponse{
		Groups: Groups(groups).Proto(),
		Pagination: &apiv1.Pagination{
			Offset:     req.Offset,
			Limit:      req.Limit,
			StartIndex: req.Offset,
			EndIndex:   req.Offset + int32(len(groups)),
			Total:      int32(count),
		},
	}, nil
}

func (a *ApiServer) GetGroup(ctx context.Context, req *apiv1.GetGroupRequest,
) (resp *apiv1.GetGroupResponse, err error) {
	// Detect whether we're returning special errors and convert to gRPC error
	defer func() {
		err = mapAndFilterErrors(err)
	}()

	gid := int(req.GroupId)
	g, err := GroupByID(ctx, nil, gid)
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

func (a *ApiServer) UpdateGroup(ctx context.Context, req *apiv1.UpdateGroupRequest,
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

	resp = &apiv1.GroupWriteResponse{
		Group: &groupv1.GroupDetails{
			GroupId: req.GroupId,
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

const (
	maxLimit = 500
)

var (
	errBadRequest      = status.Error(codes.InvalidArgument, "Bad request")
	errInvalidLimit    = status.Errorf(codes.InvalidArgument, "Bad request: limit is required and must be <= %d", maxLimit)
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

	logrus.WithError(err).Debug("suppressing error at API boundary")

	return errInternal
}
