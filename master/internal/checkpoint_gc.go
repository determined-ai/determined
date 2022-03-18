package internal

import (
	"fmt"
	"time"

	"github.com/determined-ai/determined/master/pkg/model"

	"github.com/pkg/errors"

	"github.com/determined-ai/determined/master/internal/db"
	"github.com/determined-ai/determined/master/internal/sproto"
	"github.com/determined-ai/determined/master/internal/task"
	"github.com/determined-ai/determined/master/pkg/actor"
	"github.com/determined-ai/determined/master/pkg/cproto"
	"github.com/determined-ai/determined/master/pkg/tasks"
)

type checkpointGCTask struct {
	rm *actor.Ref
	db *db.PgDB

	taskID model.TaskID
	tasks.GCCkptSpec
	jobID             model.JobID
	jobSubmissionTime time.Time

	task *sproto.AllocateRequest
	// TODO (DET-789): Set up proper log handling for checkpoint GC.
	logs []sproto.ContainerLog
}

func (t *checkpointGCTask) Receive(ctx *actor.Context) error {
	switch msg := ctx.Message().(type) {
	case actor.PreStart:
		t.task = &sproto.AllocateRequest{
			TaskID:            t.taskID,
			JobID:             t.jobID,
			JobSubmissionTime: t.jobSubmissionTime,
			AllocationID:      model.NewAllocationID(fmt.Sprintf("%s.%d", t.taskID, 1)),
			Name:              fmt.Sprintf("Checkpoint GC (Experiment %d)", t.ExperimentID),
			FittingRequirements: sproto.FittingRequirements{
				SingleAgent: true,
			},
			TaskActor: ctx.Self(),
		}
		ctx.Tell(t.rm, *t.task)

	case sproto.ResourcesAllocated:
		ctx.Log().Info("starting checkpoint garbage collection")

		allocationToken, err := t.db.StartAllocationSession(msg.ID)
		if err != nil {
			return errors.Wrap(err, "cannot start a new task session for a GC task")
		}

		if len(msg.Resources) != 1 {
			return errors.New("multi-reservation checkpoint gc is wrong")
		}

		msg.Resources[0].Start(ctx, t.ToTaskSpec(allocationToken), sproto.ResourcesRuntimeInfo{
			Token:        allocationToken,
			AgentRank:    0,
			IsMultiAgent: false,
		})
	case sproto.ReleaseResources, task.AllocationSignal:
		// Ignore the release resource message and wait for the GC job to finish.

	case sproto.ResourcesStateChanged:
		if msg.Container.State != cproto.Terminated {
			return nil
		}

		if exit := msg.ResourcesStopped; exit.Failure != nil {
			ctx.Log().Errorf("checkpoint garbage collection failed: %v", exit)
			for _, log := range t.logs {
				ctx.Log().Error(log.String())
			}
		} else {
			ctx.Log().Info("finished checkpoint garbage collection")
		}
		ctx.Self().Stop()

	case sproto.ContainerLog:
		t.logs = append(t.logs, msg)

	case actor.PostStop:
		if t.task != nil {
			if err := t.db.DeleteAllocationSession(t.task.AllocationID); err != nil {
				ctx.Log().WithError(err).Error("cannot delete task session for a GC task")
			}
		}

	default:
		return actor.ErrUnexpectedMessage(ctx)
	}
	return nil
}
