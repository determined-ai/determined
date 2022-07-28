package internal

import (
	"context"

	"github.com/pkg/errors"
	log "github.com/sirupsen/logrus"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"

	"github.com/determined-ai/determined/master/internal/db"
	"github.com/determined-ai/determined/master/internal/grpcutil"
	"github.com/determined-ai/determined/proto/pkg/apiv1"
	"github.com/determined-ai/determined/proto/pkg/projectv1"
	"github.com/determined-ai/determined/proto/pkg/workspacev1"
)

func (a *apiServer) GetProjectByID(id int32) (*projectv1.Project, error) {
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

func (a *apiServer) CheckParentWorkspaceUnarchived(pid int32) error {
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
	_ context.Context, req *apiv1.GetProjectRequest,
) (*apiv1.GetProjectResponse, error) {
	p, err := a.GetProjectByID(req.Id)
	return &apiv1.GetProjectResponse{Project: p}, err
}

func (a *apiServer) PostProject(
	ctx context.Context, req *apiv1.PostProjectRequest,
) (*apiv1.PostProjectResponse, error) {
	user, err := a.CurrentUser(ctx, &apiv1.CurrentUserRequest{})
	if err != nil {
		return nil, err
	}

	_, err = a.GetWorkspaceByID(req.WorkspaceId, 0, true)
	if err != nil {
		return nil, err
	}

	p := &projectv1.Project{}
	err = a.m.db.QueryProto("insert_project", p, req.Name, req.Description,
		req.WorkspaceId, user.User.Id)

	return &apiv1.PostProjectResponse{Project: p},
		errors.Wrapf(err, "error creating project %s in database", req.Name)
}

func (a *apiServer) AddProjectNote(
	_ context.Context, req *apiv1.AddProjectNoteRequest,
) (*apiv1.AddProjectNoteResponse, error) {
	p, err := a.GetProjectByID(req.ProjectId)
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
	_ context.Context, req *apiv1.PutProjectNotesRequest,
) (*apiv1.PutProjectNotesResponse, error) {
	newp := &projectv1.Project{}
	err := a.m.db.QueryProto("insert_project_note", newp, req.ProjectId, req.Notes)
	return &apiv1.PutProjectNotesResponse{Notes: newp.Notes},
		errors.Wrapf(err, "error putting project notes")
}

func (a *apiServer) PatchProject(
	_ context.Context, req *apiv1.PatchProjectRequest,
) (*apiv1.PatchProjectResponse, error) {
	// Verify current project exists and can be edited.
	currProject, err := a.GetProjectByID(req.Id)
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
	error,
) {
	user, _, err := grpcutil.GetUser(ctx, a.m.db, &a.m.config.InternalConfig.ExternalSessions)
	if err != nil {
		return nil, err
	}

	holder := &projectv1.Project{}
	err = a.m.db.QueryProto("deletable_project", holder, req.Id, user.ID, user.Admin)

	if holder.Id == 0 {
		return nil, errors.Wrapf(err, "project (%d) does not exist or not deletable by this user",
			req.Id)
	}

	expList, err := a.m.db.ProjectExperiments(int(req.Id))
	if err != nil {
		return nil, err
	}
	go func() {
		log.WithError(err).Errorf("deleting project %d experiments", req.Id)
		for _, exp := range expList {
			if err := a.deleteExperiment(exp, user); err != nil {
				log.WithError(err).Errorf("failed to delete experiment %d", exp.ID)
				return
			}
		}
		log.WithError(err).Errorf("project %d experiments deleted successfully", req.Id)
		err = a.m.db.QueryProto("delete_project", holder, req.Id, user.ID, user.Admin)
		if err != nil {
			log.WithError(err).Errorf("failed to delete project %d", req.Id)
			return
		}
		log.WithError(err).Errorf("project %d deleted successfully", req.Id)
	}()

	if len(expList) == 0 {
		err = a.m.db.QueryProto("delete_project", holder, req.Id, user.ID, user.Admin)
		return &apiv1.DeleteProjectResponse{Completed: (err == nil)},
			errors.Wrapf(err, "error deleting project (%d)", req.Id)
	} else {
		return &apiv1.DeleteProjectResponse{Completed: false},
			errors.Wrapf(err, "error deleting experiments on project (%d)", req.Id)
	}
}

func (a *apiServer) MoveProject(
	ctx context.Context, req *apiv1.MoveProjectRequest) (*apiv1.MoveProjectResponse,
	error,
) {
	_, err := a.GetWorkspaceByID(req.DestinationWorkspaceId, 0, true)
	if err != nil {
		return nil, err
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
	error,
) {
	user, err := a.CurrentUser(ctx, &apiv1.CurrentUserRequest{})
	if err != nil {
		return nil, err
	}

	err = a.CheckParentWorkspaceUnarchived(req.Id)
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
	error,
) {
	user, err := a.CurrentUser(ctx, &apiv1.CurrentUserRequest{})
	if err != nil {
		return nil, err
	}

	err = a.CheckParentWorkspaceUnarchived(req.Id)
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
