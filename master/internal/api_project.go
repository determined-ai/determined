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
	"github.com/determined-ai/determined/proto/pkg/projectv1"
	"github.com/determined-ai/determined/proto/pkg/workspacev1"
)

func (a *apiServer) GetProjectFromID(id int32) (*projectv1.Project, error) {
	p := &projectv1.Project{}
	switch err := a.m.db.QueryProto("get_project", p, id); err {
	case db.ErrNotFound:
		return nil, status.Errorf(
			codes.NotFound, "project (%d) not found", id)
	default:
		return p, errors.Wrapf(err,
			"error fetching project (%d) from database", id)
	}
}

func (a *apiServer) ConfirmParentWorkspaceUnarchived(pid int32) error {
	w := &workspacev1.Workspace{}
	err := a.m.db.QueryProto("get_workspace_from_project", w, pid)
	if err != nil {
		return errors.Wrapf(err,
			"error fetching project (%v)'s workspace from database", pid)
	}

	if w.Archived {
		return errors.Errorf("This project belongs to an archived workspace. " +
			"To make changes, first unarchive the workspace.")
	}
	return nil
}

func (a *apiServer) GetProject(
	_ context.Context, req *apiv1.GetProjectRequest) (*apiv1.GetProjectResponse, error) {
	p, err := a.GetProjectFromID(req.Id)
	return &apiv1.GetProjectResponse{Project: p}, err
}

func (a *apiServer) GetProjectExperiments(_ context.Context,
	req *apiv1.GetProjectExperimentsRequest) (*apiv1.GetProjectExperimentsResponse,
	error) {
	// Verify that project exists.
	if _, err := a.GetProjectFromID(req.Id); err != nil {
		return nil, err
	}

	// Construct the experiment filtering expression.
	var allStates []string
	for _, state := range req.States {
		allStates = append(allStates, strings.TrimPrefix(state.String(), "STATE_"))
	}
	stateFilterExpr := strings.Join(allStates, ",")
	userFilterExpr := strings.Join(req.Users, ",")
	labelFilterExpr := strings.Join(req.Labels, ",")
	archivedExpr := ""
	if req.Archived != nil {
		archivedExpr = strconv.FormatBool(req.Archived.Value)
	}

	// Construct the ordering expression.
	orderColMap := map[apiv1.GetProjectExperimentsRequest_SortBy]string{
		apiv1.GetProjectExperimentsRequest_SORT_BY_UNSPECIFIED: "id",
		apiv1.GetProjectExperimentsRequest_SORT_BY_ID:          "id",
		apiv1.GetProjectExperimentsRequest_SORT_BY_DESCRIPTION: "description",
		apiv1.GetProjectExperimentsRequest_SORT_BY_NAME:        "name",
		apiv1.GetProjectExperimentsRequest_SORT_BY_START_TIME:  "start_time",
		apiv1.GetProjectExperimentsRequest_SORT_BY_END_TIME:    "end_time",
		apiv1.GetProjectExperimentsRequest_SORT_BY_STATE:       "state",
		apiv1.GetProjectExperimentsRequest_SORT_BY_NUM_TRIALS:  "num_trials",
		apiv1.GetProjectExperimentsRequest_SORT_BY_PROGRESS:    "COALESCE(progress, 0)",
		apiv1.GetProjectExperimentsRequest_SORT_BY_USER:        "username",
	}
	sortByMap := map[apiv1.OrderBy]string{
		apiv1.OrderBy_ORDER_BY_UNSPECIFIED: "ASC",
		apiv1.OrderBy_ORDER_BY_ASC:         "ASC",
		apiv1.OrderBy_ORDER_BY_DESC:        "DESC NULLS LAST",
	}
	orderExpr := ""
	switch _, ok := orderColMap[req.SortBy]; {
	case !ok:
		return nil, fmt.Errorf("unsupported sort by %s", req.SortBy)
	case orderColMap[req.SortBy] != "id":
		orderExpr = fmt.Sprintf(
			"%s %s, id %s",
			orderColMap[req.SortBy], sortByMap[req.OrderBy], sortByMap[req.OrderBy],
		)
	default:
		orderExpr = fmt.Sprintf("id %s", sortByMap[req.OrderBy])
	}

	resp := &apiv1.GetProjectExperimentsResponse{}
	return resp, a.m.db.QueryProtof(
		"get_experiments",
		[]interface{}{orderExpr},
		resp,
		stateFilterExpr,
		archivedExpr,
		userFilterExpr,
		labelFilterExpr,
		req.Description,
		req.Name,
		req.Id,
		req.Offset,
		req.Limit,
	)
}

func (a *apiServer) PostProject(
	ctx context.Context, req *apiv1.PostProjectRequest) (*apiv1.PostProjectResponse, error) {
	user, err := a.CurrentUser(ctx, &apiv1.CurrentUserRequest{})
	if err != nil {
		return nil, err
	}

	w, err := a.GetWorkspaceFromID(req.WorkspaceId)
	if err != nil {
		return nil, err
	}
	if w.Immutable {
		return nil, errors.Errorf("workspace (%v) is immutable and cannot add new projects.",
			w.Id)
	}
	if w.Archived {
		return nil, errors.Errorf("workspace (%v) is archived and cannot add new projects.",
			w.Id)
	}

	p := &projectv1.Project{}
	err = a.m.db.QueryProto("insert_project", p, req.Name, req.Description,
		req.WorkspaceId, user.User.Id)

	return &apiv1.PostProjectResponse{Project: p},
		errors.Wrapf(err, "error creating project %s in database", req.Name)
}

func (a *apiServer) AddProjectNote(
	_ context.Context, req *apiv1.AddProjectNoteRequest) (*apiv1.AddProjectNoteResponse, error) {
	p, err := a.GetProjectFromID(req.ProjectId)
	if err != nil {
		return nil, err
	}

	notes := p.Notes
	notes = append(notes, &projectv1.Note{
		Name:     req.Note.Name,
		Contents: req.Note.Contents,
	})

	newp := &projectv1.Project{}
	err = a.m.db.QueryProto("insert_project_note", newp, req.ProjectId, notes)
	return &apiv1.AddProjectNoteResponse{Notes: newp.Notes},
		errors.Wrapf(err, "error adding project note")
}

func (a *apiServer) PatchProject(
	_ context.Context, req *apiv1.PatchProjectRequest) (*apiv1.PatchProjectResponse, error) {
	// Verify current project exists and can be edited.
	currProject, err := a.GetProjectFromID(req.Id)
	if err != nil {
		return nil, err
	}
	if currProject.Archived {
		return nil, errors.Errorf("project (%d) is archived and cannot have attributes updated.",
			currProject.Id)
	}
	if currProject.Immutable {
		return nil, errors.Errorf("project (%v) is immutable and cannot have attributes updated.",
			currProject.Id)
	}

	madeChanges := false
	if req.Project.Name != nil && req.Project.Name.Value != currProject.Name {
		log.Infof("project (%d) name changing from \"%s\" to \"%s\"",
			currProject.Id, currProject.Name, req.Project.Name.Value)
		madeChanges = true
		currProject.Name = req.Project.Name.Value
	}

	if req.Project.Description != nil && req.Project.Description.Value != currProject.Description {
		log.Infof("project (%d) description changing from \"%s\" to \"%s\"",
			currProject.Id, currProject.Description, req.Project.Description.Value)
		madeChanges = true
		currProject.Description = req.Project.Description.Value
	}

	if !madeChanges {
		return &apiv1.PatchProjectResponse{Project: currProject}, nil
	}

	finalProject := &projectv1.Project{}
	err = a.m.db.QueryProto("update_project",
		finalProject, currProject.Id, currProject.Name, currProject.Description)

	return &apiv1.PatchProjectResponse{Project: finalProject},
		errors.Wrapf(err, "error updating project (%d) in database", currProject.Id)
}

func (a *apiServer) DeleteProject(
	ctx context.Context, req *apiv1.DeleteProjectRequest) (*apiv1.DeleteProjectResponse,
	error) {
	user, err := a.CurrentUser(ctx, &apiv1.CurrentUserRequest{})
	if err != nil {
		return nil, err
	}

	holder := &projectv1.Project{}
	err = a.m.db.QueryProto("delete_project", holder, req.Id, user.User.Id,
		user.User.Admin)

	if holder.Id == 0 {
		return nil, errors.Wrapf(err, "project (%d) does not exist or not deletable by this user",
			req.Id)
	}

	return &apiv1.DeleteProjectResponse{},
		errors.Wrapf(err, "error deleting project (%d)", req.Id)
}

func (a *apiServer) MoveProject(
	ctx context.Context, req *apiv1.MoveProjectRequest) (*apiv1.MoveProjectResponse,
	error) {
	w, err := a.GetWorkspaceFromID(req.DestinationWorkspaceId)
	if err != nil {
		return nil, err
	}
	if w.Immutable {
		return nil, errors.Errorf("workspace (%v) is immutable and cannot add new projects.",
			w.Id)
	}
	if w.Archived {
		return nil, errors.Errorf("workspace (%v) is archived and cannot add new projects.",
			w.Id)
	}

	user, err := a.CurrentUser(ctx, &apiv1.CurrentUserRequest{})
	if err != nil {
		return nil, err
	}

	holder := &projectv1.Project{}
	err = a.m.db.QueryProto("move_project", holder, req.ProjectId,
		req.DestinationWorkspaceId, user.User.Id, user.User.Admin)

	if holder.Id == 0 {
		return nil, errors.Wrapf(err, "project (%d) does not exist or not moveable by this user",
			req.ProjectId)
	}

	return &apiv1.MoveProjectResponse{},
		errors.Wrapf(err, "error moving project (%d)", req.ProjectId)
}

func (a *apiServer) ArchiveProject(
	ctx context.Context, req *apiv1.ArchiveProjectRequest) (*apiv1.ArchiveProjectResponse,
	error) {
	user, err := a.CurrentUser(ctx, &apiv1.CurrentUserRequest{})
	if err != nil {
		return nil, err
	}

	err = a.ConfirmParentWorkspaceUnarchived(req.Id)
	if err != nil {
		return nil, err
	}

	holder := &projectv1.Project{}
	err = a.m.db.QueryProto("archive_project", holder, req.Id, true,
		user.User.Id, user.User.Admin)

	if holder.Id == 0 {
		return nil, errors.Wrapf(err, "project (%d) is not archive-able by this user",
			req.Id)
	}

	return &apiv1.ArchiveProjectResponse{},
		errors.Wrapf(err, "error archiving project (%d)", req.Id)
}

func (a *apiServer) UnarchiveProject(
	ctx context.Context, req *apiv1.UnarchiveProjectRequest) (*apiv1.UnarchiveProjectResponse,
	error) {
	user, err := a.CurrentUser(ctx, &apiv1.CurrentUserRequest{})
	if err != nil {
		return nil, err
	}

	err = a.ConfirmParentWorkspaceUnarchived(req.Id)
	if err != nil {
		return nil, err
	}

	holder := &projectv1.Project{}
	err = a.m.db.QueryProto("archive_project", holder, req.Id, false,
		user.User.Id, user.User.Admin)

	if holder.Id == 0 {
		return nil, errors.Wrapf(err, "project (%d) is not unarchive-able by this user",
			req.Id)
	}

	return &apiv1.UnarchiveProjectResponse{},
		errors.Wrapf(err, "error unarchiving project (%d)", req.Id)
}
