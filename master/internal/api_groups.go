package internal

import (
	"context"

	"github.com/uptrace/bun"

	"github.com/determined-ai/determined/master/internal/usergroup"
	"github.com/determined-ai/determined/master/pkg/model"
	"github.com/determined-ai/determined/proto/pkg/apiv1"
	"github.com/determined-ai/determined/proto/pkg/groupv1"
)

// FIXME: look at how errors are handled in the rest of the API and follow that pattern
func (a *apiServer) CreateGroup(ctx context.Context, req *apiv1.CreateGroupRequest,
) (*apiv1.GroupWriteResponse, error) {
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
) (*apiv1.GroupSearchResponse, error) {
	groups, err := usergroup.SearchGroups(ctx, model.UserID(req.UserId))
	if err != nil {
		return nil, err
	}

	return &apiv1.GroupSearchResponse{
		Groups: usergroup.Groups(groups).Proto(),
	}, nil
}

func (a *apiServer) GetGroup(ctx context.Context, req *apiv1.GetGroupRequest,
) (*apiv1.GetGroupResponse, error) {
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

	resp.Group.GroupId = int32(oldGroup.ID)
	resp.Group.Name = newName
	for _, user := range users {
		resp.Group.Users = append(resp.Group.Users, user.Proto())
	}

	return resp, nil
}

func (a *apiServer) GetUsersInGroup(ctx context.Context, req *apiv1.GetUsersInGroupRequest,
) (resp *apiv1.GetUsersInGroupResponse, err error) {
	users, err := usergroup.GetUsersInGroup(ctx, int(req.GetGroupId()))
	if err != nil {
		return nil, err
	}

	for _, user := range users {
		resp.Users = append(resp.Users, user.Proto())
	}

	return resp, nil
}

func (a *apiServer) DeleteGroup(ctx context.Context, req *apiv1.DeleteGroupRequest,
) (resp *apiv1.DeleteGroupResponse, err error) {
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
