package internal

import (
	"context"

	"github.com/determined-ai/determined/proto/pkg/apiv1"
)

func (a *apiServer) CreateGroup(_ context.Context, req *apiv1.CreateGroupRequest,
) (resp *apiv1.GroupWriteResponse, err error) {
	return
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