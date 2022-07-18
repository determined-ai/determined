package internal

import (
	"context"

	"github.com/determined-ai/determined/master/internal/usergroup"
	"github.com/determined-ai/determined/master/pkg/model"
	"github.com/determined-ai/determined/proto/pkg/apiv1"
	"github.com/determined-ai/determined/proto/pkg/groupv1"
)

func (a *apiServer) CreateGroup(ctx context.Context, req *apiv1.CreateGroupRequest,
) (resp *apiv1.GroupWriteResponse, err error) {

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

func (a *apiServer) GetGroups(_ context.Context, req *apiv1.GroupSearchRequest,
) (resp *apiv1.GroupSearchResponse, err error) {
	return
}

func (a *apiServer) GetGroup(_ context.Context, req *apiv1.GetGroupRequest,
) (resp *apiv1.GetGroupResponse, err error) {
	return
}

func (a *apiServer) UpdateGroup(_ context.Context, req *apiv1.UpdateGroupRequest,
) (resp *apiv1.GroupWriteResponse, err error) {
	return
}

func (a *apiServer) DeleteGroup(_ context.Context, req *apiv1.DeleteGroupRequest,
) (resp *apiv1.DeleteGroupResponse, err error) {
	return
}

func intsToUserIDs(ints []int32) []model.UserID {
	ids := make([]model.UserID, len(ints))

	for i := range ints {
		ids[i] = model.UserID(ints[i])
	}

	return ids
}
