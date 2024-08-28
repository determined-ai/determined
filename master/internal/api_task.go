package internal

import (
	"context"

	"github.com/pkg/errors"

	"github.com/determined-ai/determined/master/internal/api"
	"github.com/determined-ai/determined/master/internal/db"
	expauth "github.com/determined-ai/determined/master/internal/experiment"
	"github.com/determined-ai/determined/master/pkg/model"
	"github.com/determined-ai/determined/proto/pkg/apiv1"
	"github.com/determined-ai/determined/proto/pkg/taskv1"
)

func (a *apiServer) GetTask(
	ctx context.Context, req *apiv1.GetTaskRequest,
) (resp *apiv1.GetTaskResponse, err error) {
	if _, _, err := a.canDoActionsOnTask(ctx, model.TaskID(req.TaskId),
		expauth.AuthZProvider.Get().CanGetExperimentArtifacts); err != nil {
		return nil, err
	}

	t := &taskv1.Task{}
	switch err := a.m.db.QueryProto("get_task", t, req.TaskId); {
	case errors.Is(err, db.ErrNotFound):
		return nil, api.NotFoundErrs("task", req.TaskId, true)
	default:
		return &apiv1.GetTaskResponse{Task: t},
			errors.Wrapf(err, "error fetching task %s from database", req.TaskId)
	}
}

func (a *apiServer) GetGenericTaskConfig(
	ctx context.Context, req *apiv1.GetGenericTaskConfigRequest,
) (resp *apiv1.GetGenericTaskConfigResponse, err error) {
	if _, _, err := a.canDoActionsOnTask(ctx, model.TaskID(req.TaskId)); err != nil {
		return nil, err
	}

	t := &taskv1.Task{}
	switch err := db.Bun().NewSelect().Model(t).
		Column("config").
		Where("task_id = ?", req.TaskId).
		Scan(ctx); {
	case errors.Is(err, db.ErrNotFound):
		return nil, api.NotFoundErrs("task", req.TaskId, true)
	case err != nil:
		return nil, errors.Wrapf(err, "error fetching task config for task '%s' from database", req.TaskId)
	default:
		if t.Config != nil {
			return &apiv1.GetGenericTaskConfigResponse{Config: *t.Config}, nil
		}
		return &apiv1.GetGenericTaskConfigResponse{Config: ""}, nil
	}
}
