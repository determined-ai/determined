package internal

import (
	"fmt"
	"time"

	"github.com/determined-ai/determined/master/pkg/logger"
	"github.com/determined-ai/determined/master/pkg/model"

	"github.com/pkg/errors"

	"github.com/determined-ai/determined/master/internal/db"
	"github.com/determined-ai/determined/master/internal/sproto"
	"github.com/determined-ai/determined/master/internal/task"
	"github.com/determined-ai/determined/master/pkg/actor"
	"github.com/determined-ai/determined/master/pkg/tasks"
)

type checkpointGCTask struct {
	rm *actor.Ref
	db *db.PgDB

	taskID       model.TaskID
	allocationID model.AllocationID
	tasks.GCCkptSpec
	jobID             model.JobID
	jobSubmissionTime time.Time

	allocation *actor.Ref
	// TODO (DET-789): Set up proper log handling for checkpoint GC.
	taskLogger *task.Logger

	logCtx logger.Context
}

func (t *checkpointGCTask) Receive(ctx *actor.Context) error {
	switch msg := ctx.Message().(type) {
	case actor.PreStart:
		t.logCtx = logger.MergeContexts(t.logCtx, logger.Context{
			"task-id":   t.taskID,
			"task-type": model.TaskTypeCheckpointGC,
		})
		ctx.AddLabels(t.logCtx)

		if err := t.db.AddTask(&model.Task{
			TaskID:     t.taskID,
			TaskType:   model.TaskTypeCheckpointGC,
			StartTime:  ctx.Self().RegisteredTime(),
			JobID:      &t.jobID,
			LogVersion: model.CurrentTaskLogVersion,
		}); err != nil {
			return errors.Wrapf(err, "persisting GC task %s", t.taskID)
		}

		t.allocationID = model.AllocationID(fmt.Sprintf("%s.%d", t.taskID, 1))

		allocation := task.NewAllocation(t.logCtx, sproto.AllocateRequest{
			TaskID:            t.taskID,
			JobID:             t.jobID,
			JobSubmissionTime: t.jobSubmissionTime,
			AllocationID:      t.allocationID,
			Name:              fmt.Sprintf("Checkpoint GC (Experiment %d)", t.ExperimentID),
			FittingRequirements: sproto.FittingRequirements{
				SingleAgent: true,
			},
			TaskActor: ctx.Self(),
		}, t.db, sproto.GetRM(ctx.Self().System()), t.taskLogger)

		t.allocation, _ = ctx.ActorOf(t.allocationID, allocation)
	case task.BuildTaskSpec:
		if ctx.ExpectingResponse() {
			ctx.Respond(t.ToTaskSpec())
		}
	case *task.AllocationExited:
		t.completeTask(ctx)
		return errors.Errorf("checkpoint GC task exited (*task.AllocationExited)")
	case actor.ChildStopped:
	case actor.ChildFailed:
		if msg.Child.Address().Local() == t.allocationID.String() {
			t.completeTask(ctx)
			return errors.Wrapf(msg.Error, "checkpoint GC task failed (actor.ChildFailed)")
		}
	case actor.PostStop:
	default:
		return actor.ErrUnexpectedMessage(ctx)
	}

<<<<<<< HEAD
	t.markCheckpointsDeleted(ctx)

	return nil
}

func (t *checkpointGCTask) markCheckpointsDeleted(ctx *actor.Context) {
	var deleteCheckpoints []uuid.UUID
	if err := json.Unmarshal(t.ToDelete, &deleteCheckpoints); err != nil {
		ctx.Log().WithError(err).Error("unmarshalling ToDelete in checkpoint GC task")
	}

	var deleteCheckpointsStr []string
	for _, dC := range deleteCheckpoints {
		deleteCheckpointsStr = append(deleteCheckpointsStr, dC.String())
	}
=======
	deleteCheckpoints := t.ToDelete
>>>>>>> 957d8afae (changes for cli, gcraw, ToDelete)

	if err := t.db.MarkCheckpointsDeleted(deleteCheckpoints); err != nil {
		ctx.Log().WithError(err).Error("updating checkpoints to delete state in checkpoint GC Task")
	}
}

func (t *checkpointGCTask) completeTask(ctx *actor.Context) {
	if err := t.db.CompleteTask(t.taskID, time.Now().UTC()); err != nil {
		ctx.Log().WithError(err).Error("marking GC task complete")
	}

	ctx.Self().Stop()
}
