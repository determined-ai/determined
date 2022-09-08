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
	"github.com/determined-ai/determined/master/internal/grpcutil"
	"github.com/determined-ai/determined/master/internal/workspace"
	"github.com/determined-ai/determined/master/pkg/model"
	"github.com/determined-ai/determined/proto/pkg/apiv1"
	"github.com/determined-ai/determined/proto/pkg/projectv1"
	"github.com/determined-ai/determined/proto/pkg/workspacev1"
)

func (a *apiServer) GetWorkspaceByID(
	id int32, curUser model.User, rejectImmutable bool,
) (*workspacev1.Workspace, error) {
	notFoundErr := status.Errorf(codes.NotFound, "workspace (%d) not found", id)
	w := &workspacev1.Workspace{}

	if err := a.m.db.QueryProto("get_workspace", w, id, curUser.ID); errors.Is(err, db.ErrNotFound) {
		return nil, notFoundErr
	} else if err != nil {
		return nil, errors.Wrapf(err, "error fetching workspace (%d) from database", id)
	}

	if ok, err := workspace.AuthZProvider.Get().CanGetWorkspace(curUser, w); err != nil {
		return nil, err
	} else if !ok {
		return nil, notFoundErr
	}

	if rejectImmutable && w.Immutable {
		return nil, errors.Errorf("workspace (%v) is immutable and cannot add new projects.", w.Id)
	}
	if rejectImmutable && w.Archived {
		return nil, errors.Errorf("workspace (%v) is archived and cannot add new projects.", w.Id)
	}
	return w, nil
}

func (a *apiServer) getWorkspaceAndCheckCanDoActions(ctx context.Context, workspaceID int32,
	rejectImmutable bool, canDoActions ...func(model.User, *workspacev1.Workspace) error,
) (*workspacev1.Workspace, model.User, error) {
	curUser, _, err := grpcutil.GetUser(ctx, a.m.db, &a.m.config.InternalConfig.ExternalSessions)
	if err != nil {
		return nil, model.User{}, err
	}
	w, err := a.GetWorkspaceByID(workspaceID, *curUser, rejectImmutable)
	if err != nil {
		return nil, model.User{}, err
	}

	for _, canDoAction := range canDoActions {
		if err = canDoAction(*curUser, w); err != nil {
			return nil, model.User{}, status.Error(codes.PermissionDenied, err.Error())
		}
	}
	return w, *curUser, nil
}

func (a *apiServer) GetWorkspace(
	ctx context.Context, req *apiv1.GetWorkspaceRequest,
) (*apiv1.GetWorkspaceResponse, error) {
	curUser, _, err := grpcutil.GetUser(ctx, a.m.db, &a.m.config.InternalConfig.ExternalSessions)
	if err != nil {
		return nil, err
	}

	w, err := a.GetWorkspaceByID(req.Id, *curUser, false)
	return &apiv1.GetWorkspaceResponse{Workspace: w}, err
}

func (a *apiServer) GetWorkspaceProjects(
	ctx context.Context, req *apiv1.GetWorkspaceProjectsRequest,
) (*apiv1.GetWorkspaceProjectsResponse, error) {
	curUser, _, err := grpcutil.GetUser(ctx, a.m.db, &a.m.config.InternalConfig.ExternalSessions)
	if err != nil {
		return nil, err
	}
	if req.Id != 0 {
		if _, err = a.GetWorkspaceByID(req.Id, *curUser, false); err != nil {
			return nil, err
		}
	}

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
	err = a.m.db.QueryProtof(
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

	resp.Projects, err = workspace.AuthZProvider.Get().
		FilterWorkspaceProjects(*curUser, resp.Projects)
	if err != nil {
		return nil, err
	}

	return resp, a.paginate(&resp.Pagination, &resp.Projects, req.Offset, req.Limit)
}

func (a *apiServer) GetWorkspaces(
	ctx context.Context, req *apiv1.GetWorkspacesRequest,
) (*apiv1.GetWorkspacesResponse, error) {
	curUser, _, err := grpcutil.GetUser(ctx, a.m.db, &a.m.config.InternalConfig.ExternalSessions)
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
		curUser.ID,
	)
	if err != nil {
		return nil, err
	}

	resp.Workspaces, err = workspace.AuthZProvider.Get().
		FilterWorkspaces(*curUser, resp.Workspaces)
	if err != nil {
		return nil, err
	}

	return resp, a.paginate(&resp.Pagination, &resp.Workspaces, req.Offset, req.Limit)
}

func (a *apiServer) PostWorkspace(
	ctx context.Context, req *apiv1.PostWorkspaceRequest,
) (*apiv1.PostWorkspaceResponse, error) {
	curUser, _, err := grpcutil.GetUser(ctx, a.m.db, &a.m.config.InternalConfig.ExternalSessions)
	if err != nil {
		return nil, err
	}
	if err = workspace.AuthZProvider.Get().CanCreateWorkspace(*curUser); err != nil {
		return nil, status.Error(codes.PermissionDenied, err.Error())
	}

	w := &workspacev1.Workspace{}
	err = a.m.db.QueryProto("insert_workspace", w, req.Name, curUser.ID)
	if err == nil && w.Id > 0 {
		holder := &workspacev1.Workspace{}
		err = a.m.db.QueryProto("pin_workspace", holder, w.Id, curUser.ID)
		if err == nil {
			w.Pinned = true
		}
	}

	return &apiv1.PostWorkspaceResponse{Workspace: w},
		errors.Wrapf(err, "error creating workspace %s in database", req.Name)
}

func (a *apiServer) PatchWorkspace(
	ctx context.Context, req *apiv1.PatchWorkspaceRequest,
) (*apiv1.PatchWorkspaceResponse, error) {
	currWorkspace, currUser, err := a.getWorkspaceAndCheckCanDoActions(ctx, req.Id, true)
	if err != nil {
		return nil, err
	}

	madeChanges := false
	if req.Workspace.Name != nil && req.Workspace.Name.Value != currWorkspace.Name {
		if err = workspace.AuthZProvider.Get().
			CanSetWorkspacesName(currUser, currWorkspace); err != nil {
			return nil, status.Error(codes.PermissionDenied, err.Error())
		}

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
		finalWorkspace, currWorkspace.Id, currWorkspace.Name, currUser.ID)

	return &apiv1.PatchWorkspaceResponse{Workspace: finalWorkspace},
		errors.Wrapf(err, "error updating workspace (%d) in database", currWorkspace.Id)
}

func (a *apiServer) deleteWorkspace(
	ctx context.Context, workspaceID int32, projects []*projectv1.Project,
) {
	log.Errorf("deleting workspace %d projects", workspaceID)
	holder := &workspacev1.Workspace{}
	for _, pj := range projects {
		expList, err := a.m.db.ProjectExperiments(int(pj.Id))
		if err != nil {
			log.WithError(err).Errorf("error fetching experiments on project %d while deleting workspace %d",
				pj.Id, workspaceID)
			_ = a.m.db.QueryProto("delete_fail_workspace", holder, workspaceID, err.Error())
			return
		}
		err = a.deleteProject(ctx, pj.Id, expList)
		if err != nil {
			log.WithError(err).Errorf("error deleting project %d while deleting workspace %d", pj.Id,
				workspaceID)
			_ = a.m.db.QueryProto("delete_fail_workspace", holder, workspaceID, err.Error())
			return
		}
	}
	err := a.m.db.QueryProto("delete_workspace", holder, workspaceID)
	if err != nil {
		log.WithError(err).Errorf("failed to delete workspace %d", workspaceID)
		_ = a.m.db.QueryProto("delete_fail_workspace", holder, workspaceID, err.Error())
		return
	}
	log.Errorf("workspace %d deleted successfully", workspaceID)
}

func (a *apiServer) DeleteWorkspace(
	ctx context.Context, req *apiv1.DeleteWorkspaceRequest) (*apiv1.DeleteWorkspaceResponse,
	error,
) {
	_, _, err := a.getWorkspaceAndCheckCanDoActions(ctx, req.Id, false,
		workspace.AuthZProvider.Get().CanDeleteWorkspace)
	if err != nil {
		return nil, err
	}

	holder := &workspacev1.Workspace{}
	err = a.m.db.QueryProto("deletable_workspace", holder, req.Id)
	if holder.Id == 0 {
		return nil, errors.Wrapf(err, "workspace (%d) does not exist or not deletable by this user",
			req.Id)
	}

	projects := []*projectv1.Project{}
	err = a.m.db.QueryProtof(
		"get_workspace_projects",
		[]interface{}{"id ASC"},
		&projects,
		req.Id,
		"",
		"",
		"",
	)
	if err != nil {
		return nil, err
	}
	if len(projects) == 0 {
		err = a.m.db.QueryProto("delete_workspace", holder, req.Id)
		return &apiv1.DeleteWorkspaceResponse{Completed: (err == nil)},
			errors.Wrapf(err, "error deleting workspace (%d)", req.Id)
	}
	go func() {
		a.deleteWorkspace(ctx, req.Id, projects)
	}()
	return &apiv1.DeleteWorkspaceResponse{Completed: false},
		errors.Wrapf(err, "error deleting workspace (%d)", req.Id)
}

func (a *apiServer) ArchiveWorkspace(
	ctx context.Context, req *apiv1.ArchiveWorkspaceRequest) (*apiv1.ArchiveWorkspaceResponse,
	error,
) {
	_, _, err := a.getWorkspaceAndCheckCanDoActions(ctx, req.Id, false,
		workspace.AuthZProvider.Get().CanArchiveWorkspace)
	if err != nil {
		return nil, err
	}

	holder := &workspacev1.Workspace{}
	if err = a.m.db.QueryProto("archive_workspace", holder, req.Id, true); err != nil {
		return nil, errors.Wrapf(err, "error archiving workspace (%d)", req.Id)
	}
	if holder.Id == 0 {
		return nil, errors.Wrapf(err, "workspace (%d) does not exist or not archive-able by this user",
			req.Id)
	}
	return &apiv1.ArchiveWorkspaceResponse{}, nil
}

func (a *apiServer) UnarchiveWorkspace(
	ctx context.Context, req *apiv1.UnarchiveWorkspaceRequest) (*apiv1.UnarchiveWorkspaceResponse,
	error,
) {
	_, _, err := a.getWorkspaceAndCheckCanDoActions(ctx, req.Id, false,
		workspace.AuthZProvider.Get().CanUnarchiveWorkspace)
	if err != nil {
		return nil, err
	}

	holder := &workspacev1.Workspace{}
	if err = a.m.db.QueryProto("archive_workspace", holder, req.Id, false); err != nil {
		return nil, errors.Wrapf(err, "error unarchiving workspace (%d)", req.Id)
	}
	if holder.Id == 0 {
		return nil, errors.Wrapf(err,
			"workspace (%d) does not exist or not unarchive-able by this user", req.Id)
	}
	return &apiv1.UnarchiveWorkspaceResponse{}, nil
}

func (a *apiServer) PinWorkspace(
	ctx context.Context, req *apiv1.PinWorkspaceRequest,
) (*apiv1.PinWorkspaceResponse, error) {
	_, currUser, err := a.getWorkspaceAndCheckCanDoActions(ctx, req.Id, false,
		workspace.AuthZProvider.Get().CanPinWorkspace)
	if err != nil {
		return nil, err
	}

	err = a.m.db.QueryProto("pin_workspace", &workspacev1.Workspace{}, req.Id, currUser.ID)

	return &apiv1.PinWorkspaceResponse{},
		errors.Wrapf(err, "error pinning workspace (%d)", req.Id)
}

func (a *apiServer) UnpinWorkspace(
	ctx context.Context, req *apiv1.UnpinWorkspaceRequest,
) (*apiv1.UnpinWorkspaceResponse, error) {
	_, currUser, err := a.getWorkspaceAndCheckCanDoActions(ctx, req.Id, false,
		workspace.AuthZProvider.Get().CanUnpinWorkspace)
	if err != nil {
		return nil, err
	}

	err = a.m.db.QueryProto("unpin_workspace", &workspacev1.Workspace{}, req.Id, currUser.ID)

	return &apiv1.UnpinWorkspaceResponse{},
		errors.Wrapf(err, "error un-pinning workspace (%d)", req.Id)
}
