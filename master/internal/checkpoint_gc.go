package internal

import (
	"fmt"
	"strings"
	"sync"
	"time"

	"github.com/google/uuid"
	"github.com/sirupsen/logrus"

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
	mu   sync.Mutex
	stop func()

	db     *db.PgDB
	rm     rm.ResourceManager
	syslog *logrus.Entry

	taskID       model.TaskID
	allocationID model.AllocationID
	tasks.GCCkptSpec
	jobID             model.JobID
	jobSubmissionTime time.Time

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
		db:     db,
		rm:     rm,
		syslog: logrus.WithField("component", "checkpointgc"),

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

		logCtx: logCtx,
	}
}

func (t *checkpointGCTask) Receive(ctx *actor.Context) error {
	t.mu.Lock()
	defer t.mu.Unlock()

	switch ctx.Message().(type) {
	case actor.PreStart:
		t.stop = ctx.Self().Stop
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

		task.DefaultService.StartAllocation(t.logCtx, sproto.AllocateRequest{
			TaskID:            t.taskID,
			JobID:             t.jobID,
			RequestTime:       time.Now().UTC(),
			JobSubmissionTime: t.jobSubmissionTime,
			AllocationID:      t.allocationID,
			Name:              fmt.Sprintf("Checkpoint GC (Experiment %d)", t.ExperimentID),
			FittingRequirements: sproto.FittingRequirements{
				SingleAgent: true,
			},
			Group:        ctx.Self(),
			ResourcePool: rp,
		}, t.db, t.rm, t.GCCkptSpec, ctx.Self().System(), func(ae *task.AllocationExited) {})

		// t.Base is just a shallow copy of the m.taskSpec on the master, so
		// use caution when mutating it.
		t.Base.TaskContainerDefaults, err = t.rm.TaskContainerDefaults(
			ctx,
			rp,
			config.GetMasterConfig().TaskContainerDefaults)
		if err != nil {
			return fmt.Errorf("creating task container defaults: %v", err)
		}
	case actor.PostStop:
	default:
		return actor.ErrUnexpectedMessage(ctx)
	}

	return nil
}

func (t *checkpointGCTask) AllocationExitedCallback(msg *task.AllocationExited) {
	t.mu.Lock()
	defer t.mu.Unlock()

	if msg.Err != nil {
		t.syslog.WithError(msg.Err).Error("wasn't able to delete checkpoints from checkpoint storage")
	}

	err := t.db.CompleteTask(t.taskID, time.Now().UTC())
	if err != nil {
		t.syslog.WithError(err).Error("marking GC task complete")
	}
	t.stop()
}
