package internal

import (
	"context"
	"fmt"
	"strconv"
	"strings"

	"github.com/pkg/errors"
	log "github.com/sirupsen/logrus"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"

	"github.com/determined-ai/determined/master/internal/db"
	"github.com/determined-ai/determined/proto/pkg/apiv1"
	"github.com/determined-ai/determined/proto/pkg/workspacev1"
)

func (a *apiServer) GetWorkspaceByID(id int32, userID int32,
	rejectImmutable bool,
) (*workspacev1.Workspace, error) {
	w := &workspacev1.Workspace{}
	switch err := a.m.db.QueryProto("get_workspace", w, id, userID); err {
	case db.ErrNotFound:
		return nil, status.Errorf(
			codes.NotFound, "workspace (%d) not found", id)
	default:
		if rejectImmutable && w.Immutable {
			return nil, errors.Errorf("workspace (%v) is immutable and cannot add new projects.", w.Id)
		}
		if rejectImmutable && w.Archived {
			return nil, errors.Errorf("workspace (%v) is archived and cannot add new projects.", w.Id)
		}
		return w, errors.Wrapf(err,
			"error fetching workspace (%d) from database", id)
	}
}

func (a *apiServer) GetWorkspace(
	ctx context.Context, req *apiv1.GetWorkspaceRequest,
) (*apiv1.GetWorkspaceResponse, error) {
	user, err := a.CurrentUser(ctx, &apiv1.CurrentUserRequest{})
	if err != nil {
		return nil, err
	}

	w, err := a.GetWorkspaceByID(req.Id, user.User.Id, false)
	return &apiv1.GetWorkspaceResponse{Workspace: w}, err
}

func (a *apiServer) GetWorkspaceProjects(ctx context.Context,
	req *apiv1.GetWorkspaceProjectsRequest) (*apiv1.GetWorkspaceProjectsResponse,
	error,
) {
	nameFilter := req.Name
	archFilterExpr := ""
	if req.Archived != nil {
		archFilterExpr = strconv.FormatBool(req.Archived.Value)
	}
	userFilterExpr := strings.Join(req.Users, ",")
	// Construct the ordering expression.
	startTime := apiv1.GetWorkspaceProjectsRequest_SORT_BY_LAST_EXPERIMENT_START_TIME
	sortColMap := map[apiv1.GetWorkspaceProjectsRequest_SortBy]string{
		apiv1.GetWorkspaceProjectsRequest_SORT_BY_UNSPECIFIED:   "id",
		apiv1.GetWorkspaceProjectsRequest_SORT_BY_CREATION_TIME: "created_at",
		startTime: "last_experiment_started_at",
		apiv1.GetWorkspaceProjectsRequest_SORT_BY_ID:          "id",
		apiv1.GetWorkspaceProjectsRequest_SORT_BY_NAME:        "name",
		apiv1.GetWorkspaceProjectsRequest_SORT_BY_DESCRIPTION: "description",
	}
	orderByMap := map[apiv1.OrderBy]string{
		apiv1.OrderBy_ORDER_BY_UNSPECIFIED: "ASC",
		apiv1.OrderBy_ORDER_BY_ASC:         "ASC",
		apiv1.OrderBy_ORDER_BY_DESC:        "DESC",
	}
	orderExpr := ""
	switch _, ok := sortColMap[req.SortBy]; {
	case !ok:
		return nil, fmt.Errorf("unsupported sort by %s", req.SortBy)
	case sortColMap[req.SortBy] != "id":
		orderExpr = fmt.Sprintf(
			"%s %s, id %s",
			sortColMap[req.SortBy], orderByMap[req.OrderBy], orderByMap[req.OrderBy],
		)
	default:
		orderExpr = fmt.Sprintf("id %s", orderByMap[req.OrderBy])
	}

	resp := &apiv1.GetWorkspaceProjectsResponse{}
	err := a.m.db.QueryProtof(
		"get_workspace_projects",
		[]interface{}{orderExpr},
		&resp.Projects,
		req.Id,
		userFilterExpr,
		nameFilter,
		archFilterExpr,
	)
	if err != nil {
		return nil, err
	}

	return resp, a.paginate(&resp.Pagination, &resp.Projects, req.Offset, req.Limit)
}

func (a *apiServer) GetWorkspaces(
	ctx context.Context, req *apiv1.GetWorkspacesRequest,
) (*apiv1.GetWorkspacesResponse, error) {
	user, err := a.CurrentUser(ctx, &apiv1.CurrentUserRequest{})
	if err != nil {
		return nil, err
	}

	nameFilter := req.Name
	archFilterExpr := ""
	if req.Archived != nil {
		archFilterExpr = strconv.FormatBool(req.Archived.Value)
	}
	pinFilterExpr := ""
	if req.Pinned != nil {
		pinFilterExpr = strconv.FormatBool(req.Pinned.Value)
	}
	userFilterExpr := strings.Join(req.Users, ",")
	// Construct the ordering expression.
	sortColMap := map[apiv1.GetWorkspacesRequest_SortBy]string{
		apiv1.GetWorkspacesRequest_SORT_BY_UNSPECIFIED: "id",
		apiv1.GetWorkspacesRequest_SORT_BY_ID:          "id",
		apiv1.GetWorkspacesRequest_SORT_BY_NAME:        "name",
	}
	orderByMap := map[apiv1.OrderBy]string{
		apiv1.OrderBy_ORDER_BY_UNSPECIFIED: "ASC",
		apiv1.OrderBy_ORDER_BY_ASC:         "ASC",
		apiv1.OrderBy_ORDER_BY_DESC:        "DESC",
	}
	orderExpr := ""
	switch _, ok := sortColMap[req.SortBy]; {
	case !ok:
		return nil, fmt.Errorf("unsupported sort by %s", req.SortBy)
	case sortColMap[req.SortBy] != "id":
		orderExpr = fmt.Sprintf(
			"%s %s, id %s",
			sortColMap[req.SortBy], orderByMap[req.OrderBy], orderByMap[req.OrderBy],
		)
	default:
		orderExpr = fmt.Sprintf("id %s", orderByMap[req.OrderBy])
	}

	resp := &apiv1.GetWorkspacesResponse{}
	err = a.m.db.QueryProtof(
		"get_workspaces",
		[]interface{}{orderExpr},
		&resp.Workspaces,
		userFilterExpr,
		nameFilter,
		archFilterExpr,
		pinFilterExpr,
		user.User.Id,
	)
	if err != nil {
		return nil, err
	}
	return resp, a.paginate(&resp.Pagination, &resp.Workspaces, req.Offset, req.Limit)
}

func (a *apiServer) PostWorkspace(
	ctx context.Context, req *apiv1.PostWorkspaceRequest,
) (*apiv1.PostWorkspaceResponse, error) {
	user, err := a.CurrentUser(ctx, &apiv1.CurrentUserRequest{})
	if err != nil {
		return nil, err
	}

	w := &workspacev1.Workspace{}
	err = a.m.db.QueryProto("insert_workspace", w, req.Name, user.User.Id)

	if err == nil && w.Id > 0 {
		holder := &workspacev1.Workspace{}
		err = a.m.db.QueryProto("pin_workspace", holder, w.Id, user.User.Id)
	}

	return &apiv1.PostWorkspaceResponse{Workspace: w},
		errors.Wrapf(err, "error creating workspace %s in database", req.Name)
}

func (a *apiServer) PatchWorkspace(
	ctx context.Context, req *apiv1.PatchWorkspaceRequest,
) (*apiv1.PatchWorkspaceResponse, error) {
	user, err := a.CurrentUser(ctx, &apiv1.CurrentUserRequest{})
	if err != nil {
		return nil, err
	}

	// Verify current workspace exists and can be edited.
	currWorkspace, err := a.GetWorkspaceByID(req.Id, user.User.Id, true)
	if err != nil {
		return nil, err
	}

	madeChanges := false
	if req.Workspace.Name != nil && req.Workspace.Name.Value != currWorkspace.Name {
		log.Infof("workspace (%d) name changing from \"%s\" to \"%s\"",
			currWorkspace.Id, currWorkspace.Name, req.Workspace.Name.Value)
		madeChanges = true
		currWorkspace.Name = req.Workspace.Name.Value
	}

	if !madeChanges {
		return &apiv1.PatchWorkspaceResponse{Workspace: currWorkspace}, nil
	}

	finalWorkspace := &workspacev1.Workspace{}
	err = a.m.db.QueryProto("update_workspace",
		finalWorkspace, currWorkspace.Id, currWorkspace.Name, user.User.Id)

	return &apiv1.PatchWorkspaceResponse{Workspace: finalWorkspace},
		errors.Wrapf(err, "error updating workspace (%d) in database", currWorkspace.Id)
}

func (a *apiServer) DeleteWorkspace(
	ctx context.Context, req *apiv1.DeleteWorkspaceRequest) (*apiv1.DeleteWorkspaceResponse,
	error,
) {
	user, err := a.CurrentUser(ctx, &apiv1.CurrentUserRequest{})
	if err != nil {
		return nil, err
	}

	holder := &workspacev1.Workspace{}
	err = a.m.db.QueryProto("delete_workspace", holder, req.Id, user.User.Id,
		user.User.Admin)

	if holder.Id == 0 {
		return nil, errors.Wrapf(err, "workspace (%d) does not exist or not deletable by this user",
			req.Id)
	}

	return &apiv1.DeleteWorkspaceResponse{},
		errors.Wrapf(err, "error deleting workspace (%d)", req.Id)
}

func (a *apiServer) ArchiveWorkspace(
	ctx context.Context, req *apiv1.ArchiveWorkspaceRequest) (*apiv1.ArchiveWorkspaceResponse,
	error,
) {
	user, err := a.CurrentUser(ctx, &apiv1.CurrentUserRequest{})
	if err != nil {
		return nil, err
	}

	holder := &workspacev1.Workspace{}
	err = a.m.db.QueryProto("archive_workspace", holder, req.Id, true,
		user.User.Id, user.User.Admin)

	if holder.Id == 0 {
		return nil, errors.Wrapf(err, "workspace (%d) does not exist or not archive-able by this user",
			req.Id)
	}

	return &apiv1.ArchiveWorkspaceResponse{},
		errors.Wrapf(err, "error archiving workspace (%d)", req.Id)
}

func (a *apiServer) UnarchiveWorkspace(
	ctx context.Context, req *apiv1.UnarchiveWorkspaceRequest) (*apiv1.UnarchiveWorkspaceResponse,
	error,
) {
	user, err := a.CurrentUser(ctx, &apiv1.CurrentUserRequest{})
	if err != nil {
		return nil, err
	}

	holder := &workspacev1.Workspace{}
	err = a.m.db.QueryProto("archive_workspace", holder, req.Id, false,
		user.User.Id, user.User.Admin)

	if holder.Id == 0 {
		return nil, errors.Wrapf(err, "workspace (%d) does not exist or not unarchive-able by this user",
			req.Id)
	}

	return &apiv1.UnarchiveWorkspaceResponse{},
		errors.Wrapf(err, "error unarchiving workspace (%d)", req.Id)
}

func (a *apiServer) PinWorkspace(
	ctx context.Context, req *apiv1.PinWorkspaceRequest,
) (*apiv1.PinWorkspaceResponse, error) {
	user, err := a.CurrentUser(ctx, &apiv1.CurrentUserRequest{})
	if err != nil {
		return nil, err
	}

	holder := &workspacev1.Workspace{}
	err = a.m.db.QueryProto("pin_workspace", holder, req.Id, user.User.Id)

	return &apiv1.PinWorkspaceResponse{},
		errors.Wrapf(err, "error pinning workspace (%d)", req.Id)
}

func (a *apiServer) UnpinWorkspace(
	ctx context.Context, req *apiv1.UnpinWorkspaceRequest,
) (*apiv1.UnpinWorkspaceResponse, error) {
	user, err := a.CurrentUser(ctx, &apiv1.CurrentUserRequest{})
	if err != nil {
		return nil, err
	}

	holder := &workspacev1.Workspace{}
	err = a.m.db.QueryProto("unpin_workspace", holder, req.Id, user.User.Id)

	return &apiv1.UnpinWorkspaceResponse{},
		errors.Wrapf(err, "error un-pinning workspace (%d)", req.Id)
}
