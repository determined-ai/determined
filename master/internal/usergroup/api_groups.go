package usergroup

import (
	"context"
	"strings"

	"github.com/pkg/errors"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"

	"github.com/determined-ai/determined/master/internal/api/apiutils"
	"github.com/determined-ai/determined/master/internal/authz"
	"github.com/determined-ai/determined/master/internal/db"
	"github.com/determined-ai/determined/master/internal/grpcutil"
	"github.com/determined-ai/determined/master/internal/user"
	"github.com/determined-ai/determined/master/pkg/model"
	"github.com/determined-ai/determined/proto/pkg/apiv1"
	"github.com/determined-ai/determined/proto/pkg/groupv1"
)

// UserGroupAPIServer is an embedded api server struct.
type UserGroupAPIServer struct{}

// CreateGroup creates a group and adds members to it, if any.
func (a *UserGroupAPIServer) CreateGroup(ctx context.Context, req *apiv1.CreateGroupRequest,
) (resp *apiv1.CreateGroupResponse, err error) {
	if strings.Contains(req.Name, user.PersonalGroupPostfix) {
		return nil, status.Error(codes.InvalidArgument,
			"group name cannot contain 'DeterminedPersonalGroup'")
	}

	// Detect whether we're returning special errors and convert to gRPC error
	defer func() {
		err = apiutils.MapAndFilterErrors(err, nil, nil)
	}()

	curUser, _, err := grpcutil.GetUser(ctx)
	if err != nil {
		return nil, err
	}
	err = AuthZProvider.Get().CanUpdateGroups(ctx, *curUser)
	if err != nil {
		return nil, err
	}

	group := model.Group{
		Name: req.Name,
	}
	uids := intsToUserIDs(req.AddUsers)

	createdGroup, users, err := AddGroupWithMembers(ctx, group, uids...)
	if err != nil {
		return nil, err
	}

	return &apiv1.CreateGroupResponse{
		Group: &groupv1.GroupDetails{
			GroupId: int32(createdGroup.ID),
			Name:    createdGroup.Name,
			Users:   model.Users(users).Proto(),
		},
	}, nil
}

// GetGroups searches for groups that fulfills the criteria given by the user.
func (a *UserGroupAPIServer) GetGroups(ctx context.Context, req *apiv1.GetGroupsRequest,
) (resp *apiv1.GetGroupsResponse, err error) {
	// Detect whether we're returning special errors and convert to gRPC error
	defer func() {
		err = apiutils.MapAndFilterErrors(err, nil, nil)
	}()

	if req.Limit > apiutils.MaxLimit || req.Limit == 0 {
		return nil, apiutils.ErrInvalidLimit
	}

	query := SearchGroupsQuery(req.Name, model.UserID(req.UserId), false)

	curUser, _, err := grpcutil.GetUser(ctx)
	if err != nil {
		return nil, err
	}

	query, err = AuthZProvider.Get().FilterGroupsList(ctx, *curUser, query)
	if err != nil {
		return nil, err
	}

	groups, memberCounts, tableCount, err := SearchGroupsPaginated(ctx,
		query, int(req.Offset), int(req.Limit))
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
func (a *UserGroupAPIServer) GetGroup(ctx context.Context, req *apiv1.GetGroupRequest,
) (resp *apiv1.GetGroupResponse, err error) {
	// Detect whether we're returning special errors and convert to gRPC error
	defer func() {
		err = apiutils.MapAndFilterErrors(err, nil, nil)
	}()

	curUser, _, err := grpcutil.GetUser(ctx)
	if err != nil {
		return nil, err
	}

	gid := int(req.GroupId)

	if err := AuthZProvider.Get().CanGetGroup(ctx, *curUser, gid); err != nil {
		return nil, authz.SubIfUnauthorized(err,
			errors.Wrapf(db.ErrNotFound, "Error getting group %d", gid))
	}

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
func (a *UserGroupAPIServer) UpdateGroup(ctx context.Context, req *apiv1.UpdateGroupRequest,
) (resp *apiv1.UpdateGroupResponse, err error) {
	// Detect whether we're returning special errors and convert to gRPC error
	defer func() {
		err = apiutils.MapAndFilterErrors(err, nil, nil)
	}()

	curUser, _, err := grpcutil.GetUser(ctx)
	if err != nil {
		return nil, err
	}

	err = AuthZProvider.Get().CanUpdateGroups(ctx, *curUser)
	if err != nil {
		return nil, err
	}

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

	resp = &apiv1.UpdateGroupResponse{
		Group: &groupv1.GroupDetails{
			GroupId: req.GroupId,
			Name:    newName,
			Users:   model.Users(users).Proto(),
		},
	}

	return resp, nil
}

// DeleteGroup deletes the database entry for the group.
func (a *UserGroupAPIServer) DeleteGroup(ctx context.Context, req *apiv1.DeleteGroupRequest,
) (resp *apiv1.DeleteGroupResponse, err error) {
	// Detect whether we're returning special errors and convert to gRPC error
	defer func() {
		err = apiutils.MapAndFilterErrors(err, nil, nil)
	}()

	curUser, _, err := grpcutil.GetUser(ctx)
	if err != nil {
		return nil, err
	}

	err = AuthZProvider.Get().CanUpdateGroups(ctx, *curUser)
	if err != nil {
		return nil, err
	}

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
