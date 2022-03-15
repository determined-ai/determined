package internal

import (
	"context"

	"github.com/pkg/errors"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"

	"github.com/determined-ai/determined/master/internal/db"
	"github.com/determined-ai/determined/proto/pkg/apiv1"
	"github.com/determined-ai/determined/proto/pkg/projectv1"
)

func (a *apiServer) GetProject(
	_ context.Context, req *apiv1.GetProjectRequest) (*apiv1.GetProjectResponse, error) {
	p := &projectv1.Project{}
	switch err := a.m.db.QueryProto("get_project", p, req.Id); err {
	case db.ErrNotFound:
		return nil, status.Errorf(
			codes.NotFound, "project \"%d\" not found", req.Id)
	default:
		return &apiv1.GetProjectResponse{Project: p},
			errors.Wrapf(err, "error fetching project \"%d\" from database", req.Id)
	}
}

func (a *apiServer) PostProject(
	ctx context.Context, req *apiv1.PostProjectRequest) (*apiv1.PostProjectResponse, error) {
	user, err := a.CurrentUser(ctx, &apiv1.CurrentUserRequest{})
	if err != nil {
		return nil, err
	}

	p := &projectv1.Project{}
	err = a.m.db.QueryProto("insert_project", p, req.Name, req.Description, user.User.Id)

	return &apiv1.PostProjectResponse{Project: p},
		errors.Wrapf(err, "error creating project %s in database", req.Name)
}
