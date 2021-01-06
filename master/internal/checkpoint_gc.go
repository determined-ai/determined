package internal

import (
	"fmt"

	"github.com/determined-ai/determined/master/internal/db"
	"github.com/determined-ai/determined/master/internal/resourcemanagers"
	"github.com/determined-ai/determined/master/internal/sproto"
	"github.com/determined-ai/determined/master/pkg/actor"
	"github.com/determined-ai/determined/master/pkg/container"
	"github.com/determined-ai/determined/master/pkg/model"
	"github.com/determined-ai/determined/master/pkg/tasks"
)

type checkpointGCTask struct {
	rm         *actor.Ref
	db         *db.PgDB
	experiment *model.Experiment

	agentUserGroup *model.AgentUserGroup
	taskSpec       *tasks.TaskSpec

	// TODO (DET-789): Set up proper log handling for checkpoint GC.
	logs []sproto.ContainerLog
}

func (t *checkpointGCTask) Receive(ctx *actor.Context) error {
	switch msg := ctx.Message().(type) {
	case actor.PreStart:
		ctx.Tell(t.rm, resourcemanagers.AllocateRequest{
			Name: fmt.Sprintf("Checkpoint GC (Experiment %d)", t.experiment.ID),
			FittingRequirements: resourcemanagers.FittingRequirements{
				SingleAgent: true,
			},
			TaskActor:      ctx.Self(),
			NonPreemptible: true,
		})

	case resourcemanagers.ResourcesAllocated:
		config := t.experiment.Config.CheckpointStorage

		checkpoints, err := t.db.ExperimentCheckpointsToGCRaw(t.experiment.ID,
			config.SaveExperimentBest, config.SaveTrialBest, config.SaveTrialLatest, true)
		if err != nil {
			return err
		}

		ctx.Log().Info("starting checkpoint garbage collection")

		for _, a := range msg.Allocations {
			taskSpec := *t.taskSpec
			taskSpec.GCCheckpoints = &tasks.GCCheckpoints{
				AgentUserGroup:   t.agentUserGroup,
				ExperimentID:     t.experiment.ID,
				ExperimentConfig: t.experiment.Config,
				ToDelete:         checkpoints,
			}
			a.Start(ctx, taskSpec)
		}
	case resourcemanagers.ReleaseResources:
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

	default:
		return actor.ErrUnexpectedMessage(ctx)
	}
	return nil
}
