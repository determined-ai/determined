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
	if err := a.canDoActionsOnTask(ctx, model.TaskID(req.TaskId),
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
