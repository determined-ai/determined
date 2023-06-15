package internal

import (
	"fmt"
	"strings"
	"time"

	"github.com/google/uuid"

	"github.com/determined-ai/determined/master/pkg/logger"
	"github.com/determined-ai/determined/master/pkg/model"
	"github.com/determined-ai/determined/master/pkg/schemas/expconf"

	"github.com/pkg/errors"

	"github.com/determined-ai/determined/master/internal/config"
	"github.com/determined-ai/determined/master/internal/db"
	"github.com/determined-ai/determined/master/internal/rm"
	"github.com/determined-ai/determined/master/internal/sproto"
	"github.com/determined-ai/determined/master/internal/task"
	"github.com/determined-ai/determined/master/pkg/actor"
	"github.com/determined-ai/determined/master/pkg/protoutils/protoconverter"
	"github.com/determined-ai/determined/master/pkg/tasks"
)

type checkpointGCTask struct {
	db *db.PgDB
	rm rm.ResourceManager

	taskID       model.TaskID
	allocationID model.AllocationID
	tasks.GCCkptSpec
	jobID             model.JobID
	jobSubmissionTime time.Time

	allocation *actor.Ref

	logCtx logger.Context
}

const fullDeleteGlob = "**/*"

func newCheckpointGCTask(
	rm rm.ResourceManager, db *db.PgDB, taskID model.TaskID,
	jobID model.JobID, jobSubmissionTime time.Time, taskSpec tasks.TaskSpec, expID int,
	legacyConfig expconf.LegacyConfig, toDeleteCheckpoints []uuid.UUID, checkpointGlobs []string,
	deleteTensorboards bool,
	agentUserGroup *model.AgentUserGroup, owner *model.User, logCtx logger.Context,
) *checkpointGCTask {
	taskSpec.AgentUserGroup = agentUserGroup
	taskSpec.Owner = owner
	conv := &protoconverter.ProtoConverter{}
	checkpointStrIDs := conv.ToStringList(toDeleteCheckpoints)
	deleteCheckpointsStr := strings.Join(checkpointStrIDs, ",")

	return &checkpointGCTask{
		taskID:            taskID,
		jobID:             jobID,
		jobSubmissionTime: jobSubmissionTime,
		GCCkptSpec: tasks.GCCkptSpec{
			Base:               taskSpec,
			ExperimentID:       expID,
			LegacyConfig:       legacyConfig,
			ToDelete:           deleteCheckpointsStr,
			CheckpointGlobs:    checkpointGlobs,
			DeleteTensorboards: deleteTensorboards,
		},
		db: db,
		rm: rm,

		logCtx: logCtx,
	}
}

func (t *checkpointGCTask) Receive(ctx *actor.Context) error {
	switch msg := ctx.Message().(type) {
	case actor.PreStart:
		if len(t.ToDelete) == 0 && !t.DeleteTensorboards {
			// Early return as nothing to do
			ctx.Self().Stop()
			return nil
		}
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

		rp, err := t.rm.ResolveResourcePool(ctx, "", 0)
		if err != nil {
			return fmt.Errorf("resolving resource pool: %w", err)
		}

		allocation := task.NewAllocation(t.logCtx, sproto.AllocateRequest{
			TaskID:            t.taskID,
			JobID:             t.jobID,
			JobSubmissionTime: t.jobSubmissionTime,
			AllocationID:      t.allocationID,
			Name:              fmt.Sprintf("Checkpoint GC (Experiment %d)", t.ExperimentID),
			FittingRequirements: sproto.FittingRequirements{
				SingleAgent: true,
			},
			AllocationRef: ctx.Self(),
			ResourcePool:  rp,
		}, t.db, t.rm, t.GCCkptSpec)

		t.allocation, _ = ctx.ActorOf(t.allocationID, allocation)

		// t.Base is just a shallow copy of the m.taskSpec on the master, so
		// use caution when mutating it.
		t.Base.TaskContainerDefaults, err = t.rm.TaskContainerDefaults(
			ctx,
			rp,
			config.GetMasterConfig().TaskContainerDefaults)
		if err != nil {
			return fmt.Errorf("creating task container defaults: %v", err)
		}
	case *task.AllocationExited:
		if msg.Err != nil {
			ctx.Log().WithError(msg.Err).Error("wasn't able to delete checkpoints from checkpoint storage")
			t.completeTask(ctx)
			return errors.Wrapf(msg.Err, "checkpoint GC task failed because allocation failed")
		}

		t.completeTask(ctx)
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

	return nil
}

func (t *checkpointGCTask) completeTask(ctx *actor.Context) {
	if err := t.db.CompleteTask(t.taskID, time.Now().UTC()); err != nil {
		ctx.Log().WithError(err).Error("marking GC task complete")
	}

	ctx.Self().Stop()
}
