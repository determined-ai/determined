package internal

import (
	"fmt"

	"github.com/determined-ai/determined/master/internal/scheduler"

	"github.com/determined-ai/determined/master/internal/db"
	"github.com/determined-ai/determined/master/internal/sproto"
	"github.com/determined-ai/determined/master/pkg/actor"
	"github.com/determined-ai/determined/master/pkg/container"
	"github.com/determined-ai/determined/master/pkg/model"
	"github.com/determined-ai/determined/master/pkg/tasks"
)

type checkpointGCTask struct {
	rp         *actor.Ref
	db         *db.PgDB
	experiment *model.Experiment

	agentUserGroup *model.AgentUserGroup

	// TODO (DET-789): Set up proper log handling for checkpoint GC.
	logs []sproto.ContainerLog
}

func (t *checkpointGCTask) Receive(ctx *actor.Context) error {
	switch msg := ctx.Message().(type) {
	case actor.PreStart:
		ctx.Tell(t.rp, scheduler.AssignResource{
			Name: fmt.Sprintf("Checkpoint GC (Experiment %d)", t.experiment.ID),
			FittingRequirements: scheduler.FittingRequirements{
				SingleAgent: true,
			},
			TaskHandler: ctx.Self(),
		})

	case scheduler.ResourceAssigned:
		config := t.experiment.Config.CheckpointStorage

		checkpoints, err := t.db.ExperimentCheckpointsToGCRaw(t.experiment.ID,
			&config.SaveExperimentBest, &config.SaveTrialBest, &config.SaveTrialLatest, true)
		if err != nil {
			return err
		}

		ctx.Log().Info("starting checkpoint garbage collection")

		for _, a := range msg.Assignments {
			a.StartContainer(tasks.TaskSpec{
				GCCheckpoints: &tasks.GCCheckpoints{
					AgentUserGroup:   t.agentUserGroup,
					ExperimentID:     t.experiment.ID,
					ExperimentConfig: t.experiment.Config,
					ToDelete:         checkpoints,
				},
			})
		}
	case scheduler.ReleaseResource:

	case sproto.ContainerStateChanged:
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

	case scheduler.ContainerStarted:
	case actor.PostStop:

	default:
		return actor.ErrUnexpectedMessage(ctx)
	}
	return nil
}
