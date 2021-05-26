package internal

import (
	"fmt"

	"github.com/pkg/errors"

	"github.com/determined-ai/determined/master/internal/db"
	"github.com/determined-ai/determined/master/internal/sproto"
	"github.com/determined-ai/determined/master/pkg/actor"
	"github.com/determined-ai/determined/master/pkg/container"
	"github.com/determined-ai/determined/master/pkg/model"
	"github.com/determined-ai/determined/master/pkg/schemas/expconf"
	"github.com/determined-ai/determined/master/pkg/tasks"
)

type checkpointGCTask struct {
	rm             *actor.Ref
	db             *db.PgDB
	experiment     *model.Experiment
	legacyConfig   expconf.LegacyConfig
	gcTensorboards bool

	keepExperimentBest int
	keepTrialBest      int
	keepTrialLatest    int

	agentUserGroup *model.AgentUserGroup
	taskSpec       *tasks.TaskSpec

	task *sproto.AllocateRequest
	// TODO (DET-789): Set up proper log handling for checkpoint GC.
	logs []sproto.ContainerLog
}

func (t *checkpointGCTask) Receive(ctx *actor.Context) error {
	switch msg := ctx.Message().(type) {
	case actor.PreStart:
		t.task = &sproto.AllocateRequest{
			ID:   sproto.NewTaskID(),
			Name: fmt.Sprintf("Checkpoint GC (Experiment %d)", t.experiment.ID),
			FittingRequirements: sproto.FittingRequirements{
				SingleAgent: true,
			},
			TaskActor:      ctx.Self(),
			NonPreemptible: true,
		}
		ctx.Tell(t.rm, *t.task)

	case sproto.ResourcesAllocated:
		taskToken, err := t.db.StartTaskSession(string(msg.ID))
		if err != nil {
			return errors.Wrap(err, "cannot start a new task session for a GC task")
		}

		checkpoints, err := t.db.ExperimentCheckpointsToGCRaw(
			t.experiment.ID,
			t.keepExperimentBest,
			t.keepTrialBest,
			t.keepTrialLatest,
			true,
		)
		if err != nil {
			return err
		}

		ctx.Log().Info("starting checkpoint garbage collection")

		for _, a := range msg.Allocations {
			taskSpec := *t.taskSpec
			taskSpec.AgentUserGroup = t.agentUserGroup
			taskSpec.TaskToken = taskToken
			taskSpec.SetInner(&tasks.GCCheckpoints{
				ExperimentID:       t.experiment.ID,
				LegacyConfig:       t.legacyConfig,
				ToDelete:           checkpoints,
				DeleteTensorboards: t.gcTensorboards,
			})
			a.Start(ctx, taskSpec)
		}
	case sproto.ReleaseResources:
		// Ignore the release resource message and wait for the GC job to finish.

	case sproto.TaskContainerStateChanged:
		if msg.Container.State != container.Terminated {
			return nil
		}
		status := msg.ContainerStopped

		if msg.ContainerStopped.Failure != nil {
			ctx.Log().Errorf("checkpoint garbage collection failed: %v", status)
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
			if err := t.db.DeleteTaskSessionByTaskID(string(t.task.ID)); err != nil {
				ctx.Log().WithError(err).Error("cannot delete task session for a GC task")
			}
		}

	default:
		return actor.ErrUnexpectedMessage(ctx)
	}
	return nil
}
