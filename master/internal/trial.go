package internal

import (
	"fmt"
	"time"

	"github.com/determined-ai/determined/master/pkg/ptrs"
	"github.com/determined-ai/determined/master/pkg/workload"

	"github.com/determined-ai/determined/master/pkg/actor/actors"

	"github.com/pkg/errors"

	"github.com/determined-ai/determined/master/internal/db"
	"github.com/determined-ai/determined/master/internal/sproto"
	"github.com/determined-ai/determined/master/pkg/actor"
	"github.com/determined-ai/determined/master/pkg/archive"
	"github.com/determined-ai/determined/master/pkg/model"
	"github.com/determined-ai/determined/master/pkg/schemas"
	"github.com/determined-ai/determined/master/pkg/schemas/expconf"
	"github.com/determined-ai/determined/master/pkg/ssh"
	"github.com/determined-ai/determined/master/pkg/tasks"
)

// On the surface, a trial is an actor which is responsible for handling:
//  - messages from the scheduler,
//  - messages from the experiment,
//  - messages from the trial container(s), and
//  - keeping the trial table of the database up-to-date.
//
// At its heart, the trial consists of a task and allocation state machine
// constantly trying to rectify with each other. On one end the experiment
// starts a trial and, along with the user, sets its desired state, 'ACTIVE',
// 'PAUSED', 'STOPPING_CANCELED', etc. On the other end there is the
// state machine of the underlying task trying to rectify itself with
// the desired state. If the trial is 'ACTIVE' and has work, the task
// is trying to allocated, start containers and await their termination,
// if it is 'STOPPING_CANCELED', it is trying to preempt the task runner
// and move us to the 'CANCELED' state, and so on.
type trial struct {
	id           int
	taskID       model.TaskID
	idSet        bool
	experimentID int

	// System dependencies.
	rm     *actor.Ref
	logger *actor.Ref
	db     db.DB

	// Fields that are essentially configuration for the trial.
	config              expconf.ExperimentConfig
	taskSpec            *tasks.TaskSpec
	modelDefinition     archive.Archive
	warmStartCheckpoint *model.Checkpoint
	generatedKeys       *ssh.PrivateAndPublicKeys

	// state is the current state of the trial. It's patched by experiment changes and kill trial.
	state model.State
	// searcher encapsulates the searcher state of the trial.
	searcher TrialSearcherState
	// restarts is a failure count, it increments when the trial fails and we retry it.
	restarts int
	// runID is a count of how many times the task container(s) have stopped and restarted, which
	// could be due to a failure or due to normal pausing and continuing. When RunID increments,
	// it effectively invalidates many outstanding messages associated with the previous run.
	runID int

	// the current allocation request.
	req *sproto.AllocateRequest
	// all the state of the current allocation.
	allocation *trialAllocation
}

// newTrial creates a trial which will try to schedule itself after it receives its first workload.
func newTrial(
	taskID model.TaskID,
	experimentID int,
	initialState model.State,
	searcher TrialSearcherState,
	rm, logger *actor.Ref,
	db db.DB,
	config expconf.ExperimentConfig,
	warmStartCheckpoint *model.Checkpoint,
	taskSpec *tasks.TaskSpec,
	modelDefinition archive.Archive,
) *trial {
	return &trial{
		taskID: taskID,

		experimentID: experimentID,
		state:        initialState,
		searcher:     searcher,

		rm:     rm,
		logger: logger,
		db:     db,

		config:              config,
		taskSpec:            taskSpec,
		modelDefinition:     modelDefinition,
		warmStartCheckpoint: warmStartCheckpoint,
	}
}

func (t *trial) Receive(ctx *actor.Context) error {
	switch msg := ctx.Message().(type) {
	case actor.PreStart:
		ctx.AddLabel("experiment-id", t.experimentID)
		if t.idSet {
			ctx.AddLabel("trial-id", t.id)
			if err := t.recover(); err != nil {
				return err
			}
			ctx.AddLabel("task-run-id", t.runID)
		}
		return nil
	case actor.PostStop:
		if !t.idSet {
			return nil
		}
		if !model.TerminalStates[t.state] {
			return t.transition(ctx, model.ErrorState)
		}
		return nil

	case model.State:
		return t.transition(ctx, msg)
	case TrialSearcherState:
		t.searcher = msg
		switch {
		case !t.searcher.Complete:
			return t.maybeAllocateTask(ctx)
		case t.searcher.Finished():
			return t.transition(ctx, model.StoppingCompletedState)
		}
		return nil

	case sproto.ResourcesAllocated:
		return t.processAllocation(ctx, msg)

	case sproto.TaskContainerStateChanged,
		sproto.ReleaseResources, watchRendezvousInfo, unwatchRendezvousInfo,
		rendezvousTimeout, watchPreemption, ackPreemption, unwatchPreemption, preemptionTimeout:
		if t.allocation == nil {
			return errNoAllocation{action: fmt.Sprintf("%T", msg)}
		}
		switch status, err := t.allocation.process(ctx); {
		case err != nil:
			return err
		case status != nil:
			return t.processAllocationExit(ctx, *status)
		}
	case sproto.ContainerLog:
		if log, err := t.enrichTrialLog(model.TrialLog{
			ContainerID: ptrs.StringPtr(string(msg.Container.ID)),
			Message:     msg.Message(),
		}); err != nil {
			ctx.Log().WithError(err).Warn("dropping container log")
		} else {
			ctx.Tell(t.logger, log)
		}
	case model.TrialLog:
		if log, err := t.enrichTrialLog(msg); err != nil {
			ctx.Log().WithError(err).Warn("dropping trial log")
		} else {
			ctx.Tell(t.logger, log)
		}

	default:
		return actor.ErrUnexpectedMessage(ctx)
	}
	return nil
}

func (t *trial) recover() error {
	runID, restarts, err := t.db.TrialRunIDAndRestarts(t.id)
	if err != nil {
		return errors.Wrap(err, "restoring old trial state")
	}
	t.runID = runID + 1
	t.restarts = restarts
	return nil
}

// maybeAllocateTask checks if the trial's task is in an allocatable state and
// allocates it if so.
func (t *trial) maybeAllocateTask(ctx *actor.Context) error {
	if !(t.allocation == nil &&
		!t.searcher.Complete &&
		t.state == model.ActiveState) {
		return nil
	}

	var name string
	if t.idSet {
		name = fmt.Sprintf("Trial %d (Experiment %d)", t.id, t.experimentID)
	} else {
		name = fmt.Sprintf("Trial (Experiment %d)", t.experimentID)
	}

	t.req = &sproto.AllocateRequest{
		AllocationID:   model.NewAllocationID(fmt.Sprintf("%s-%d", t.taskID, t.runID)),
		Name:           name,
		Group:          ctx.Self().Parent(),
		SlotsNeeded:    t.config.Resources().SlotsPerTrial(),
		NonPreemptible: false,
		Label:          t.config.Resources().AgentLabel(),
		ResourcePool:   t.config.Resources().ResourcePool(),
		FittingRequirements: sproto.FittingRequirements{
			SingleAgent: false,
		},
		TaskActor: ctx.Self(),
	}
	if err := ctx.Ask(t.rm, *t.req).Error(); err != nil {
		return errors.Wrap(err, "failed to request allocation")
	}
	return nil
}

func (t *trial) processAllocation(ctx *actor.Context, msg sproto.ResourcesAllocated) error {
	// Ignore this message if it is from the last run of the trial.
	if t.req == nil || msg.ID != t.req.AllocationID {
		ctx.Log().
			WithField("allocation", t.req).
			Infof("ignoring and stale allocation %v", msg)
		return nil
	}

	if t.generatedKeys == nil {
		generatedKeys, err := ssh.GenerateKey(nil)
		if err != nil {
			return errors.Wrap(err, "failed to generate keys for trial")
		}
		t.generatedKeys = &generatedKeys
	}

	if !t.idSet {
		modelTrial := model.NewTrial(
			t.taskID,
			t.searcher.Create.RequestID,
			t.experimentID,
			model.JSONObj(t.searcher.Create.Hparams),
			t.warmStartCheckpoint,
			int64(t.searcher.Create.TrialSeed))
		if err := t.db.AddTrial(modelTrial); err != nil {
			return errors.Wrap(err, "failed to save trial to database")
		}
		t.id = modelTrial.ID
		t.idSet = true
		ctx.AddLabel("trial-id", t.id)
		ctx.Tell(t.rm, sproto.SetTaskName{
			Name:        fmt.Sprintf("Trial %d (Experiment %d)", t.id, t.experimentID),
			TaskHandler: ctx.Self(),
		})
		ctx.Tell(ctx.Self().Parent(), trialCreated{requestID: t.searcher.Create.RequestID, trialID: t.id})
	}

	t.runID++
	if err := t.db.UpdateTrialRunID(t.id, t.runID); err != nil {
		return errors.Wrap(err, "failed to save trial run AllocationID")
	}
	ctx.AddLabel("trial-run-id", t.runID)

	if err := t.db.AddAllocation(t.taskID, t.req.AllocationID, msg.ResourcePool); err != nil {
		return errors.Wrap(err, "failed to save trial allocation")
	}

	ctx.Log().Infof("starting trial container")

	taskToken, err := t.db.StartAllocationSession(t.req.AllocationID)
	if err != nil {
		return errors.Wrap(err, "cannot start a new task session for a trial")
	}

	var latestBatch int
	latestCheckpoint, err := t.db.LatestCheckpointForTrial(t.id)
	switch {
	case err != nil:
		return errors.Wrapf(err, "failed to query latest checkpoint for trial")
	case latestCheckpoint == nil:
		latestCheckpoint = t.warmStartCheckpoint
	default:
		latestBatch = latestCheckpoint.TotalBatches
	}

	trialSpec := &tasks.TrialSpec{
		Base: *t.taskSpec,

		ExperimentID:     t.experimentID,
		TrialID:          t.id,
		TrialRunID:       t.runID,
		ExperimentConfig: schemas.Copy(t.config).(expconf.ExperimentConfig),
		ModelDefinition:  t.modelDefinition,
		HParams:          t.searcher.Create.Hparams,
		TrialSeed:        t.searcher.Create.TrialSeed,
		LatestBatch:      latestBatch,
		LatestCheckpoint: latestCheckpoint,
		IsMultiAgent:     len(msg.Reservations) > 1,
	}

	t.allocation = newTrialAllocation(*t.req, msg.Reservations)
	actors.NotifyAfter(ctx, rendezvousTimeoutDuration, rendezvousTimeout{taskID: t.req.AllocationID})

	for rank, a := range msg.Reservations {
		a.Start(ctx, trialSpec.ToTaskSpec(t.generatedKeys, taskToken), rank)
	}

	return nil
}

func (t *trial) processAllocationExit(ctx *actor.Context, exit AllocationExitStatus) error {
	ctx.Log().
		WithField("preempt_ack", t.allocation.preemption.acknowledged()).
		WithField("killed", t.allocation.killed).
		Info("trial allocation exited")

	ctx.Tell(t.rm, sproto.ResourcesReleased{TaskActor: ctx.Self()})

	if err := t.db.DeleteAllocationSession(t.req.AllocationID); err != nil {
		return errors.Wrap(err, "error delete task session for a trial")
	}

	if err := t.db.CompleteAllocation(t.req.AllocationID); err != nil {
		return errors.Wrap(err, "failed to mark trial run completed")
	}

	switch {
	case exit.err != nil:
		ctx.Log().
			WithError(exit.err).
			Errorf("trial failed (restart %d/%d)", t.restarts, t.config.MaxRestarts())
		t.restarts++
		if err := t.db.UpdateTrialRestarts(t.id, t.restarts); err != nil {
			return err
		}
		if t.restarts > t.config.MaxRestarts() {
			ctx.Tell(ctx.Self().Parent(), trialReportEarlyExit{
				trialID: t.id,
				reason:  workload.Errored,
			})
			return t.transition(ctx, model.ErrorState)
		}
	case exit.userRequestedStop:
		ctx.Tell(ctx.Self().Parent(), trialReportEarlyExit{
			trialID: t.id,
			reason:  workload.UserCanceled,
		})
		return t.transition(ctx, model.CompletedState)
	case t.searcher.Finished():
		return t.transition(ctx, model.CompletedState)
	case model.StoppingStates[t.state]:
		return t.transition(ctx, model.StoppingToTerminalStates[t.state])
	}

	t.req = nil
	t.allocation = nil
	return errors.Wrap(t.maybeAllocateTask(ctx), "failed to reschedule trial")
}

// transition handles rectifying user and experiment requested states with task state, and
// moving us out of progressive states once they are reflected by the state of the task.
func (t *trial) transition(ctx *actor.Context, state model.State) error {
	// All the logic to transition a trial's state lives in the db layer, maybe it should
	// be moved.
	ctx.Log().Infof("trial changed from state %s to %s", t.state, state)
	if t.idSet {
		if err := t.db.UpdateTrial(t.id, state); err != nil {
			return errors.Wrap(err, "updating trial with end state")
		}
	}
	t.state = state

	switch {
	case t.state == model.ActiveState:
		return t.maybeAllocateTask(ctx)
	case t.state == model.PausedState:
		switch {
		case t.req == nil:
			ctx.Log().Info("terminating trial before resources are requested")
		case t.allocation == nil:
			ctx.Log().Info("terminating trial before resources are allocated")
			ctx.Tell(t.rm, sproto.ResourcesReleased{TaskActor: ctx.Self()})
		default:
			switch status, err := t.allocation.terminate(ctx, preempt); {
			case err != nil:
				return errors.Wrap(err, "error preempting allocation")
			case status != nil:
				return t.processAllocationExit(ctx, *status)
			}
		}
	case model.StoppingStates[t.state]:
		switch {
		case t.req == nil:
			ctx.Log().Info("terminating trial before resources are requested")
			return t.transition(ctx, model.StoppingToTerminalStates[t.state])
		case t.allocation == nil:
			ctx.Log().Info("terminating trial before resources are allocated")
			ctx.Tell(t.rm, sproto.ResourcesReleased{TaskActor: ctx.Self()})
			return t.transition(ctx, model.StoppingToTerminalStates[t.state])
		default:
			action := map[model.State]terminationType{
				model.StoppingCompletedState: noop,
				model.StoppingCanceledState:  preempt,
				model.StoppingKilledState:    kill,
				model.StoppingErrorState:     kill,
			}[t.state]
			switch status, err := t.allocation.terminate(ctx, action); {
			case err != nil:
				return errors.Wrapf(err, "error terminating allocation (%s)", action)
			case status != nil:
				return t.processAllocationExit(ctx, *status)
			}
		}
	case model.TerminalStates[t.state]:
		ctx.Self().Stop()
	}
	return nil
}

func (t *trial) enrichTrialLog(log model.TrialLog) (model.TrialLog, error) {
	if !t.idSet {
		return model.TrialLog{}, fmt.Errorf(
			"cannot handle trial log before ID is set: %v", log)
	}
	log.TrialID = t.id
	log.Message += "\n"
	if log.Timestamp == nil {
		log.Timestamp = ptrs.TimePtr(time.Now().UTC())
	}
	if log.Level == nil {
		log.Level = ptrs.StringPtr("INFO")
	}
	if log.Source == nil {
		log.Source = ptrs.StringPtr("master")
	}
	if log.StdType == nil {
		log.StdType = ptrs.StringPtr("stdout")
	}
	return log, nil
}
