package internal

import (
	"context"

	"github.com/pkg/errors"
	log "github.com/sirupsen/logrus"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"

	"github.com/determined-ai/determined/master/internal/db"
	"github.com/determined-ai/determined/proto/pkg/apiv1"
	"github.com/determined-ai/determined/proto/pkg/workspacev1"
)

func (a *apiServer) GetWorkspaceFromID(id int32, userID int32) (*workspacev1.Workspace,
	error) {
	w := &workspacev1.Workspace{}
	switch err := a.m.db.QueryProto("get_workspace", w, id, userID); err {
	case db.ErrNotFound:
		return nil, status.Errorf(
			codes.NotFound, "workspace (%d) not found", id)
	default:
		return w, errors.Wrapf(err,
			"error fetching workspace (%d) from database", id)
	}
}

func (a *apiServer) PostWorkspace(
	ctx context.Context, req *apiv1.PostWorkspaceRequest) (*apiv1.PostWorkspaceResponse, error) {
	user, err := a.CurrentUser(ctx, &apiv1.CurrentUserRequest{})
	if err != nil {
		return nil, err
	}

	w := &workspacev1.Workspace{}
	err = a.m.db.QueryProto("insert_workspace", w, req.Name, user.User.Id)

	return &apiv1.PostWorkspaceResponse{Workspace: w},
		errors.Wrapf(err, "error creating workspace %s in database", req.Name)
}

func (a *apiServer) PatchWorkspace(
	ctx context.Context, req *apiv1.PatchWorkspaceRequest) (*apiv1.PatchWorkspaceResponse, error) {
	user, err := a.CurrentUser(ctx, &apiv1.CurrentUserRequest{})
	if err != nil {
		return nil, err
	}

	// Verify current workspace exists and can be edited.
	currWorkspace, err := a.GetWorkspaceFromID(req.Id, user.User.Id)
	if err != nil {
		return nil, err
	}
	if currWorkspace.Archived {
		return nil, errors.Errorf("workspace (%v) is archived and cannot have attributes updated.",
			currWorkspace.Id)
	}
	if currWorkspace.Immutable {
		return nil, errors.Errorf("workspace (%v) is immutable and cannot have attributes updated.",
			currWorkspace.Id)
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
	error) {
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
	error) {
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
	error) {
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
	ctx context.Context, req *apiv1.PinWorkspaceRequest) (*apiv1.PinWorkspaceResponse, error) {
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
	ctx context.Context, req *apiv1.UnpinWorkspaceRequest) (*apiv1.UnpinWorkspaceResponse, error) {
	user, err := a.CurrentUser(ctx, &apiv1.CurrentUserRequest{})
	if err != nil {
		return nil, err
	}

	holder := &workspacev1.Workspace{}
	err = a.m.db.QueryProto("unpin_workspace", holder, req.Id, user.User.Id)

	return &apiv1.UnpinWorkspaceResponse{},
		errors.Wrapf(err, "error un-pinning workspace (%d)", req.Id)
}
