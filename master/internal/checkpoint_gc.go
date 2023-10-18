package internal

import (
	"fmt"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/pkg/errors"
	"github.com/sirupsen/logrus"

	"github.com/determined-ai/determined/master/internal/config"
	"github.com/determined-ai/determined/master/internal/db"
	"github.com/determined-ai/determined/master/internal/rm"
	"github.com/determined-ai/determined/master/internal/rm/tasklist"
	"github.com/determined-ai/determined/master/internal/sproto"
	"github.com/determined-ai/determined/master/internal/task"
	"github.com/determined-ai/determined/master/pkg/logger"
	"github.com/determined-ai/determined/master/pkg/model"
	"github.com/determined-ai/determined/master/pkg/protoutils/protoconverter"
	"github.com/determined-ai/determined/master/pkg/schemas/expconf"
	"github.com/determined-ai/determined/master/pkg/tasks"
)

const fullDeleteGlob = "**/*"

func runCheckpointGCTask(
	rm rm.ResourceManager,
	db *db.PgDB,
	taskID model.TaskID,
	jobID model.JobID,
	jobSubmissionTime time.Time,
	taskSpec tasks.TaskSpec,
	expID int,
	legacyConfig expconf.LegacyConfig,
	toDeleteCheckpoints []uuid.UUID,
	checkpointGlobs []string,
	deleteTensorboards bool,
	agentUserGroup *model.AgentUserGroup,
	owner *model.User,
	logCtx logger.Context,
) error {
	conv := &protoconverter.ProtoConverter{}
	checkpointStrIDs := conv.ToStringList(toDeleteCheckpoints)
	deleteCheckpointsStr := strings.Join(checkpointStrIDs, ",")

	if len(deleteCheckpointsStr) == 0 && !deleteTensorboards {
		// Early return as nothing to do
		return nil
	}

	rp, err := rm.ResolveResourcePool("", -1, 0)
	if err != nil {
		return fmt.Errorf("resolving resource pool: %w", err)
	}

	// t.Base is just a shallow copy of the m.taskSpec on the master, so
	// use caution when mutating it.
	tcd, err := rm.TaskContainerDefaults(
		rp,
		config.GetMasterConfig().TaskContainerDefaults)
	if err != nil {
		return fmt.Errorf("creating task container defaults: %v", err)
	}
	taskSpec.TaskContainerDefaults = tcd

	taskSpec.AgentUserGroup = agentUserGroup
	taskSpec.Owner = owner

	gcSpec := tasks.GCCkptSpec{
		Base:               taskSpec,
		ExperimentID:       expID,
		LegacyConfig:       legacyConfig,
		ToDelete:           deleteCheckpointsStr,
		CheckpointGlobs:    checkpointGlobs,
		DeleteTensorboards: deleteTensorboards,
	}

	logCtx = logger.MergeContexts(logCtx, logger.Context{
		"task-id":   taskID,
		"task-type": model.TaskTypeCheckpointGC,
	})
	syslog := logrus.WithField("component", "checkpointgc").WithFields(logCtx.Fields())

	if err := db.AddTask(&model.Task{
		TaskID:     taskID,
		TaskType:   model.TaskTypeCheckpointGC,
		StartTime:  time.Now().UTC(),
		JobID:      &jobID,
		LogVersion: model.CurrentTaskLogVersion,
	}); err != nil {
		return errors.Wrapf(err, "persisting GC task %s", taskID)
	}

	allocationID := model.AllocationID(fmt.Sprintf("%s.%d", taskID, 1))
	gcJobID := model.JobID(fmt.Sprintf("checkpoint_gc-%s", allocationID))

	resultChan := make(chan error, 1)
	onExit := func(ae *task.AllocationExited) {
		if err := db.CompleteTask(taskID, time.Now().UTC()); err != nil {
			syslog.WithError(err).Error("marking GC task complete")
		}
		if err := tasklist.GroupPriorityChangeRegistry.Delete(gcJobID); err != nil {
			syslog.WithError(err).Error("deleting group priority change registry")
		}
		resultChan <- ae.Err
	}

	if err := tasklist.GroupPriorityChangeRegistry.Add(gcJobID, nil); err != nil {
		return err
	}
	err = task.DefaultService.StartAllocation(logCtx, sproto.AllocateRequest{
		TaskID:            taskID,
		JobID:             gcJobID,
		JobSubmissionTime: jobSubmissionTime,
		AllocationID:      allocationID,
		Name:              fmt.Sprintf("Checkpoint GC (Experiment %d)", expID),
		FittingRequirements: sproto.FittingRequirements{
			SingleAgent: true,
		},
		ResourcePool: rp,
	}, db, rm, gcSpec, onExit)
	if err != nil {
		return err
	}
	return <-resultChan
}
