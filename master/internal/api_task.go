package internal

import (
	"context"

	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"

	"github.com/pkg/errors"

	"github.com/determined-ai/determined/master/internal/db"
	"github.com/determined-ai/determined/proto/pkg/apiv1"
	"github.com/determined-ai/determined/proto/pkg/taskv1"
	"github.com/determined-ai/determined/proto/pkg/tensorboardv1"
)

func (a *apiServer) GetTask(
	ctx context.Context, req *apiv1.GetTaskRequest,
) (resp *apiv1.GetTaskResponse, err error) {
	t := &taskv1.Task{}
	switch err := a.m.db.QueryProto("get_task", t, req.TaskId); {
	case errors.Is(err, db.ErrNotFound):
		return nil, status.Errorf(
			codes.NotFound, "task %s not found", req.TaskId)
	default:
		// For tensor board task, put it in WAITING state if metrics is not ready.
		// Use first allocation as an indicator.
		if t.TaskType == "TENSORBOARD" && len(t.Allocations) > 0 &&
			t.Allocations[0].State == taskv1.State_STATE_PENDING {
			var tb *tensorboardv1.Tensorboard
			if err = a.ask(
				tensorboardsAddr.Child(req.TaskId), &tensorboardv1.Tensorboard{}, &tb,
			); err == nil {
				if len(tb.ExperimentIds) > 0 {
					metricExist, errm := db.MetricsExist(tb.ExperimentIds)
					if errm == nil && !metricExist {
						t.Allocations[0].State = taskv1.State_STATE_WAITING
					}
				}
			}
		}
		return &apiv1.GetTaskResponse{Task: t},
			errors.Wrapf(err, "error fetching task %s from database", req.TaskId)
	}
}
