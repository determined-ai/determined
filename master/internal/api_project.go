package internal

import (
	"context"
	"strings"

	"github.com/pkg/errors"
	log "github.com/sirupsen/logrus"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"

	"github.com/determined-ai/determined/master/internal/db"
	"github.com/determined-ai/determined/master/internal/grpcutil"
	"github.com/determined-ai/determined/master/internal/project"
	"github.com/determined-ai/determined/master/pkg/model"
	"github.com/determined-ai/determined/proto/pkg/apiv1"
	"github.com/determined-ai/determined/proto/pkg/projectv1"
	"github.com/determined-ai/determined/proto/pkg/workspacev1"
)

func (a *apiServer) GetProjectByID(
	ctx context.Context, id int32, curUser model.User,
) (*projectv1.Project, error) {
	notFoundErr := status.Errorf(codes.NotFound, "project (%d) not found", id)
	p := &projectv1.Project{}
	if err := a.m.db.QueryProto("get_project", p, id); errors.Is(err, db.ErrNotFound) {
		return nil, notFoundErr
	} else if err != nil {
		return nil, errors.Wrapf(err, "error fetching project (%d) from database", id)
	}

	if ok, err := project.AuthZProvider.Get().CanGetProject(ctx, curUser, p); err != nil {
		return nil, err
	} else if !ok {
		return nil, notFoundErr
	}
	return p, nil
}

func (a *apiServer) getProjectAndCheckCanDoActions(
	ctx context.Context, projectID int32,
	canDoActions ...func(context.Context, model.User, *projectv1.Project) error,
) (*projectv1.Project, model.User, error) {
	curUser, _, err := grpcutil.GetUser(ctx)
	if err != nil {
		return nil, model.User{}, err
	}
	p, err := a.GetProjectByID(ctx, projectID, *curUser)
	if err != nil {
		return nil, model.User{}, err
	}

	for _, canDoAction := range canDoActions {
		if err = canDoAction(ctx, *curUser, p); err != nil {
			return nil, model.User{}, status.Error(codes.PermissionDenied, err.Error())
		}
	}
	return p, *curUser, nil
}

func (a *apiServer) CheckParentWorkspaceUnarchived(project *projectv1.Project) error {
	w := &workspacev1.Workspace{}
	err := a.m.db.QueryProto("get_workspace_from_project", w, project.Id)
	if err != nil {
		return errors.Wrapf(err,
			"error fetching project (%v)'s workspace from database", project.Id)
	}

	if w.Archived {
		return errors.Errorf("This project belongs to an archived workspace. " +
			"To make changes, first unarchive the workspace.")
	}
	return nil
}

func (a *apiServer) GetProject(
	ctx context.Context, req *apiv1.GetProjectRequest,
) (*apiv1.GetProjectResponse, error) {
	curUser, _, err := grpcutil.GetUser(ctx)
	if err != nil {
		return nil, err
	}

	p, err := a.GetProjectByID(ctx, req.Id, *curUser)
	return &apiv1.GetProjectResponse{Project: p}, err
}

func (a *apiServer) PostProject(
	ctx context.Context, req *apiv1.PostProjectRequest,
) (*apiv1.PostProjectResponse, error) {
	curUser, _, err := grpcutil.GetUser(ctx)
	if err != nil {
		return nil, err
	}
	w, err := a.GetWorkspaceByID(ctx, req.WorkspaceId, *curUser, true)
	if err != nil {
		return nil, err
	}
	if err = project.AuthZProvider.Get().CanCreateProject(ctx, *curUser, w); err != nil {
		return nil, status.Error(codes.PermissionDenied, err.Error())
	}

	p := &projectv1.Project{}
	err = a.m.db.QueryProto("insert_project", p, req.Name, req.Description,
		req.WorkspaceId, curUser.ID)

	return &apiv1.PostProjectResponse{Project: p},
		errors.Wrapf(err, "error creating project %s in database", req.Name)
}

func (a *apiServer) AddProjectNote(
	ctx context.Context, req *apiv1.AddProjectNoteRequest,
) (*apiv1.AddProjectNoteResponse, error) {
	p, _, err := a.getProjectAndCheckCanDoActions(ctx, req.ProjectId,
		project.AuthZProvider.Get().CanSetProjectNotes)
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

func (a *apiServer) PutProjectNotes(
	ctx context.Context, req *apiv1.PutProjectNotesRequest,
) (*apiv1.PutProjectNotesResponse, error) {
	_, _, err := a.getProjectAndCheckCanDoActions(ctx, req.ProjectId,
		project.AuthZProvider.Get().CanSetProjectNotes)
	if err != nil {
		return nil, err
	}

	newp := &projectv1.Project{}
	err = a.m.db.QueryProto("insert_project_note", newp, req.ProjectId, req.Notes)
	return &apiv1.PutProjectNotesResponse{Notes: newp.Notes},
		errors.Wrapf(err, "error putting project notes")
}

func (a *apiServer) PatchProject(
	ctx context.Context, req *apiv1.PatchProjectRequest,
) (*apiv1.PatchProjectResponse, error) {
	currProject, currUser, err := a.getProjectAndCheckCanDoActions(ctx, req.Id)
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
		if err = project.AuthZProvider.Get().CanSetProjectName(ctx, currUser, currProject); err != nil {
			return nil, status.Error(codes.PermissionDenied, err.Error())
		}

		log.Infof("project (%d) name changing from \"%s\" to \"%s\"",
			currProject.Id, currProject.Name, req.Project.Name.Value)
		madeChanges = true
		currProject.Name = req.Project.Name.Value
	}

	if req.Project.Description != nil && req.Project.Description.Value != currProject.Description {
		if err = project.AuthZProvider.Get().
			CanSetProjectDescription(ctx, currUser, currProject); err != nil {
			return nil, status.Error(codes.PermissionDenied, err.Error())
		}

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

func (a *apiServer) deleteProject(ctx context.Context, projectID int32,
	expList []*model.Experiment,
) (err error) {
	holder := &projectv1.Project{}
	user, _, err := grpcutil.GetUser(ctx)
	if err != nil {
		log.WithError(err).Errorf("failed to access user and delete project %d", projectID)
		_ = a.m.db.QueryProto("delete_fail_project", holder, projectID, err.Error())
		return err
	}

	log.Debugf("deleting project %d experiments", projectID)
	for _, exp := range expList {
		if err = a.deleteExperiment(exp, user); err != nil {
			log.WithError(err).Errorf("failed to delete experiment %d", exp.ID)
			_ = a.m.db.QueryProto("delete_fail_project", holder, projectID, err.Error())
			return err
		}
	}
	log.Debugf("project %d experiments deleted successfully", projectID)
	err = a.m.db.QueryProto("delete_project", holder, projectID)
	if err != nil {
		log.WithError(err).Errorf("failed to delete project %d", projectID)
		_ = a.m.db.QueryProto("delete_fail_project", holder, projectID, err.Error())
		return err
	}
	log.Debugf("project %d deleted successfully", projectID)
	return nil
}

func (a *apiServer) DeleteProject(
	ctx context.Context, req *apiv1.DeleteProjectRequest) (*apiv1.DeleteProjectResponse,
	error,
) {
	_, _, err := a.getProjectAndCheckCanDoActions(ctx, req.Id,
		project.AuthZProvider.Get().CanDeleteProject)
	if err != nil {
		return nil, err
	}

	holder := &projectv1.Project{}
	err = a.m.db.QueryProto("deletable_project", holder, req.Id)
	if holder.Id == 0 {
		return nil, errors.Wrapf(err, "project (%d) does not exist or not deletable by this user",
			req.Id)
	}

	expList, err := a.m.db.ProjectExperiments(int(req.Id))
	if err != nil {
		return nil, err
	}

	if len(expList) == 0 {
		err = a.m.db.QueryProto("delete_project", holder, req.Id)
		return &apiv1.DeleteProjectResponse{Completed: (err == nil)},
			errors.Wrapf(err, "error deleting project (%d)", req.Id)
	}
	go func() {
		_ = a.deleteProject(ctx, req.Id, expList)
	}()
	return &apiv1.DeleteProjectResponse{Completed: false},
		errors.Wrapf(err, "error deleting project (%d)", req.Id)
}

func (a *apiServer) MoveProject(
	ctx context.Context, req *apiv1.MoveProjectRequest) (*apiv1.MoveProjectResponse,
	error,
) {
	curUser, _, err := grpcutil.GetUser(ctx)
	if err != nil {
		return nil, err
	}
	p, err := a.GetProjectByID(ctx, req.ProjectId, *curUser)
	if err != nil { // Can view project?
		return nil, err
	}
	// Allow projects to be moved from immutable workspaces but not to immutable workspaces.
	from, err := a.GetWorkspaceByID(ctx, p.WorkspaceId, *curUser, false)
	if err != nil {
		return nil, err
	}
	to, err := a.GetWorkspaceByID(ctx, req.DestinationWorkspaceId, *curUser, true)
	if err != nil {
		return nil, err
	}
	if err = project.AuthZProvider.Get().CanMoveProject(ctx, *curUser, p, from, to); err != nil {
		return nil, status.Error(codes.PermissionDenied, err.Error())
	}

	holder := &projectv1.Project{}
	err = a.m.db.QueryProto("move_project", holder, req.ProjectId, req.DestinationWorkspaceId)
	if err != nil {
		return nil, errors.Wrapf(err, "error moving project (%d)", req.ProjectId)
	}
	if holder.Id == 0 {
		return nil, errors.Wrapf(err, "project (%d) does not exist or not moveable by this user",
			req.ProjectId)
	}

	return &apiv1.MoveProjectResponse{}, nil
}

func (a *apiServer) ArchiveProject(
	ctx context.Context, req *apiv1.ArchiveProjectRequest) (*apiv1.ArchiveProjectResponse,
	error,
) {
	p, _, err := a.getProjectAndCheckCanDoActions(ctx, req.Id,
		project.AuthZProvider.Get().CanArchiveProject)
	if err != nil {
		return nil, err
	}
	if err = a.CheckParentWorkspaceUnarchived(p); err != nil {
		return nil, err
	}

	holder := &projectv1.Project{}
	if err = a.m.db.QueryProto("archive_project", holder, req.Id, true); err != nil {
		return nil, errors.Wrapf(err, "error archiving project (%d)", req.Id)
	}
	if holder.Id == 0 {
		return nil, errors.Wrapf(err, "project (%d) is not archive-able by this user",
			req.Id)
	}

	return &apiv1.ArchiveProjectResponse{}, nil
}

func (a *apiServer) UnarchiveProject(
	ctx context.Context, req *apiv1.UnarchiveProjectRequest) (*apiv1.UnarchiveProjectResponse,
	error,
) {
	p, _, err := a.getProjectAndCheckCanDoActions(ctx, req.Id,
		project.AuthZProvider.Get().CanUnarchiveProject)
	if err != nil {
		return nil, err
	}
	if err = a.CheckParentWorkspaceUnarchived(p); err != nil {
		return nil, err
	}

	holder := &projectv1.Project{}
	if err = a.m.db.QueryProto("archive_project", holder, req.Id, false); err != nil {
		return nil, errors.Wrapf(err, "error unarchiving project (%d)", req.Id)
	}
	if holder.Id == 0 {
		return nil, errors.Wrapf(err, "project (%d) is not unarchive-able by this user",
			req.Id)
	}
	return &apiv1.UnarchiveProjectResponse{}, nil
}

func (a *apiServer) GetExperimentGroups(
	ctx context.Context, req *apiv1.GetExperimentGroupsRequest) (*apiv1.GetExperimentGroupsResponse,
	error,
) {
	resp := &apiv1.GetExperimentGroupsResponse{}
	err := a.m.db.QueryProto("get_experiment_groups", &resp.Groups, req.ProjectId)
	if err != nil {
		return nil, err
	}

	return resp, nil
}

func (a *apiServer) GetExperimentGroupByID(id int32) (*projectv1.ExperimentGroup, error) {
	notFoundErr := status.Errorf(codes.NotFound, "experiment group (%d) not found", id)
	g := &projectv1.ExperimentGroup{}
	if err := a.m.db.QueryProto("get_experiment_group", g, id); errors.Is(err, db.ErrNotFound) {
		return nil, notFoundErr
	} else if err != nil {
		return nil, errors.Wrapf(err, "error fetching experiment group (%d) from database", id)
	}

	return g, nil
}

func (a *apiServer) PostExperimentGroup(
	ctx context.Context, req *apiv1.PostExperimentGroupRequest) (*apiv1.PostExperimentGroupResponse,
	error,
) {
	currProject, _, err := a.getProjectAndCheckCanDoActions(ctx, req.ProjectId)
	if err != nil {
		return nil, err
	}
	if currProject.Archived {
		return nil, errors.Errorf("project (%d) is archived and cannot have attributes updated.",
			currProject.Id)
	}
	if len(strings.TrimSpace(req.Name)) == 0 {
		return nil, status.Errorf(codes.InvalidArgument,
			"`name` must not be an empty or whitespace string.")
	}

	g := &projectv1.ExperimentGroup{}
	err = a.m.db.QueryProto("insert_experiment_group", g, req.Name, req.ProjectId)

	return &apiv1.PostExperimentGroupResponse{Group: g},
		errors.Wrapf(err, "error creating experiment group %s in database", req.Name)
}

func (a *apiServer) PatchExperimentGroup(
	ctx context.Context, req *apiv1.PatchExperimentGroupRequest,
) (*apiv1.PatchExperimentGroupResponse, error) {
	currProject, _, err := a.getProjectAndCheckCanDoActions(ctx, req.ProjectId)
	if err != nil {
		return nil, err
	}
	if currProject.Archived {
		return nil, errors.Errorf("project (%d) is archived and cannot have attributes updated.",
			currProject.Id)
	}

	currExperimentGroup, err := a.GetExperimentGroupByID(req.GroupId)
	if err != nil {
		return nil, err
	}

	madeChanges := false
	if req.Group.Name != nil && *req.Group.Name != currExperimentGroup.Name {
		if len(strings.TrimSpace(*req.Group.Name)) == 0 {
			return nil, status.Errorf(codes.InvalidArgument,
				"`name` must not be an empty or whitespace string.")
		}
		log.Infof("experiment group (%d) name changing from \"%s\" to \"%s\"",
			currExperimentGroup.Id, currExperimentGroup.Name, *req.Group.Name)
		madeChanges = true
		currExperimentGroup.Name = *req.Group.Name
	}

	if !madeChanges {
		return &apiv1.PatchExperimentGroupResponse{Group: currExperimentGroup}, nil
	}

	finalExperimentGroup := &projectv1.ExperimentGroup{}
	err = a.m.db.QueryProto("update_experiment_group",
		finalExperimentGroup, currExperimentGroup.Id, currExperimentGroup.Name)

	return &apiv1.PatchExperimentGroupResponse{Group: finalExperimentGroup},
		errors.Wrapf(err, "error updating experiment group (%d) in database", currExperimentGroup.Id)
}

func (a *apiServer) DeleteExperimentGroup(
	ctx context.Context, req *apiv1.DeleteExperimentGroupRequest) (*apiv1.DeleteExperimentGroupResponse,
	error,
) {
	currProject, _, err := a.getProjectAndCheckCanDoActions(ctx, req.ProjectId)
	if err != nil {
		return nil, err
	}
	if currProject.Archived {
		return nil, errors.Errorf("project (%d) is archived and cannot have attributes updated.",
			currProject.Id)
	}

	holder := &projectv1.ExperimentGroup{}
	err = a.m.db.QueryProto("delete_experiment_group", holder, req.GroupId)
	return &apiv1.DeleteExperimentGroupResponse{},
		errors.Wrapf(err, "error deleting experiment group (%d)", req.GroupId)
}
