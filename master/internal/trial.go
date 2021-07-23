package internal

import (
	"fmt"
	"time"

	"github.com/determined-ai/determined/master/internal/task"

	"github.com/determined-ai/determined/master/pkg/ptrs"
	"github.com/determined-ai/determined/master/pkg/workload"

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

// A trial is a task actor which is responsible for handling:
//  - messages from the resource manager,
//  - messages from the experiment,
//  - messages from the trial container(s), and
//  - keeping the trial table of the database up-to-date.
//
// The trial's desired state is dictated to it by the experiment, searcher and user; they push
// it to states like 'ACTIVE', 'PAUSED' and kill or wake it when more work is available. It takes
// this information and works with the resource manager, allocation, etc, to push us towards
// a terminal state, by requesting resources, managing them and restarting them on failures.
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
	searcher trialSearcherState
	// restarts is a failure count, it increments when the trial fails and we retry it.
	restarts int
	// runID is a count of how many times the task container(s) have stopped and restarted, which
	// could be due to a failure or due to normal pausing and continuing. When RunID increments,
	// it effectively invalidates many outstanding messages associated with the previous run.
	runID int

	// the current allocation request.
	req *sproto.AllocateRequest
	// all the state of the current allocation.
	allocation *task.Allocation
}

// newTrial creates a trial which will try to schedule itself after it receives its first workload.
func newTrial(
	taskID model.TaskID,
	experimentID int,
	initialState model.State,
	searcher trialSearcherState,
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
	case trialSearcherState:
		t.searcher = msg
		switch {
		case !t.searcher.Complete:
			return t.maybeAllocateTask(ctx)
		case t.searcher.Complete && t.searcher.Closed:
			return t.transition(ctx, model.StoppingCompletedState)
		}
		return nil

	case sproto.ResourcesAllocated:
		return t.processAllocation(ctx, msg)
	case task.AllocationTerminated:
		return t.processAllocationExit(ctx, msg)

	case sproto.TaskContainerStateChanged,
		sproto.ReleaseResources, task.WatchRendezvousInfo, task.UnwatchRendezvousInfo,
		task.RendezvousTimeout, task.WatchPreemption, task.AckPreemption,
		task.UnwatchPreemption, task.PreemptionTimeout:
		if t.allocation == nil {
			return task.ErrNoAllocation{Action: fmt.Sprintf("%T", msg)}
		}
		return t.allocation.Receive(ctx)

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

// recover recovers the trial minimal (hopefully to stay) state for a trial actor.
// Separately, the experiment stores and recovers our searcher state.
func (t *trial) recover() error {
	runID, restarts, err := t.db.TrialRunIDAndRestarts(t.id)
	if err != nil {
		return errors.Wrap(err, "restoring old trial state")
	}
	t.runID = runID + 1
	t.restarts = restarts
	return nil
}

// maybeAllocateTask checks if the trial should allocate state and allocates it if so.
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

// processAllocation receives resources and starts them, taking care of house keeping like
// saving metadata, materializing the task spec and initializing the trialAllocation.
func (t *trial) processAllocation(ctx *actor.Context, msg sproto.ResourcesAllocated) error {
	if t.req == nil {
		ctx.Log().Infof("no allocation requested by received %v", msg)
		return nil
	} else if msg.ID != t.req.AllocationID {
		ctx.Log().Infof("ignoring stale allocation %v (actual = %v)", msg, t.req)
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
		ctx.Tell(ctx.Self().Parent(), trialCreated{requestID: t.searcher.Create.RequestID})
	}

	t.runID++
	if err := t.db.UpdateTrialRunID(t.id, t.runID); err != nil {
		return errors.Wrap(err, "failed to save trial run AllocationID")
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

	t.allocation = task.NewAllocation(
		t.taskID, *t.req, msg.Reservations, trialSpec, t.generatedKeys, t.db)
	return t.allocation.Prestart(ctx)
}

// processAllocationExit cleans up after an allocation exit and exits permanently or reallocates.
func (t *trial) processAllocationExit(ctx *actor.Context, exit task.AllocationTerminated) error {
	ctx.Log().Info("trial allocation exited")

	// Clean up old stuff.
	if err := t.allocation.PostStop(ctx); err != nil {
		return err
	}
	t.req = nil
	t.allocation = nil
	ctx.Tell(t.rm, sproto.ResourcesReleased{TaskActor: ctx.Self()})

	// Decide if this is permanent.
	switch {
	case exit.Err != nil:
		ctx.Log().
			WithError(exit.Err).
			Errorf("trial failed (restart %d/%d)", t.restarts, t.config.MaxRestarts())
		t.restarts++
		if err := t.db.UpdateTrialRestarts(t.id, t.restarts); err != nil {
			return err
		}
		if t.restarts > t.config.MaxRestarts() {
			ctx.Tell(ctx.Self().Parent(), trialReportEarlyExit{
				requestID: t.searcher.Create.RequestID,
				reason:    workload.Errored,
			})
			return t.transition(ctx, model.ErrorState)
		}
	case exit.UserRequestedStop:
		ctx.Tell(ctx.Self().Parent(), trialReportEarlyExit{
			requestID: t.searcher.Create.RequestID,
			reason:    workload.UserCanceled,
		})
		return t.transition(ctx, model.CompletedState)
	case t.searcher.Complete && t.searcher.Closed:
		return t.transition(ctx, model.CompletedState)
	case model.StoppingStates[t.state]:
		return t.transition(ctx, model.StoppingToTerminalStates[t.state])
	}

	// Maybe reschedule.
	return errors.Wrap(t.maybeAllocateTask(ctx), "failed to reschedule trial")
}

// transition transitions the trial and rectifies the underlying allocation state with it.
func (t *trial) transition(ctx *actor.Context, state model.State) error {
	// Transition the trial.
	ctx.Log().Infof("trial changed from state %s to %s", t.state, state)
	if t.idSet {
		if err := t.db.UpdateTrial(t.id, state); err != nil {
			return errors.Wrap(err, "updating trial with end state")
		}
	}
	t.state = state

	// Rectify the allocation state with the transition.
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
			return t.allocation.Terminate(ctx, task.Graceful)
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
			action := map[model.State]task.TerminationType{
				model.StoppingCompletedState: task.Noop,
				model.StoppingCanceledState:  task.Graceful,
				model.StoppingKilledState:    task.Kill,
				model.StoppingErrorState:     task.Kill,
			}[t.state]
			return t.allocation.Terminate(ctx, action)
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
