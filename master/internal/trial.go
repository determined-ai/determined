package internal

import (
	"fmt"
	"strconv"
	"strings"
	"time"

	"github.com/determined-ai/determined/master/internal/task"

	"github.com/determined-ai/determined/master/pkg/ptrs"

	"github.com/pkg/errors"

	"github.com/determined-ai/determined/master/internal/db"
	"github.com/determined-ai/determined/master/internal/sproto"
	"github.com/determined-ai/determined/master/pkg/actor"
	"github.com/determined-ai/determined/master/pkg/aproto"
	"github.com/determined-ai/determined/master/pkg/model"
	"github.com/determined-ai/determined/master/pkg/schemas"
	"github.com/determined-ai/determined/master/pkg/schemas/expconf"
	"github.com/determined-ai/determined/master/pkg/ssh"
	"github.com/determined-ai/determined/master/pkg/tasks"
)

func TrialAddr(trialID int) string {
	return fmt.Sprintf("trial-%d", trialID)
}

func MustParseTrialAddr(addr string) int {
	if addr[:6] != "trial-" {
		panic("cannot parse trial address")
	}
	intVar, err := strconv.Atoi(addr[6:])
	if err != nil {
		panic(errors.Wrap(err, "cannot parse trial address"))
	}
	return intVar
}

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
	model    model.Trial
	searcher trialSearcherState
	// restarts is a failure count, it increments when the trial fails and we retry it.
	restarts int
	// runID is a count of how many times the task container(s) have stopped and restarted, which
	// could be due to a failure or due to normal pausing and continuing. When RunID increments,
	// it effectively invalidates many outstanding messages associated with the previous run.
	runID int

	// System dependencies.
	rm     *actor.Ref
	logger *actor.Ref
	db     db.DB

	// Fields that are retrieved or generated on the fly.
	config              expconf.ExperimentConfig
	taskSpec            *tasks.TaskSpec
	warmStartCheckpoint *model.Checkpoint
	generatedKeys       *ssh.PrivateAndPublicKeys

	// a ref to the current allocation
	allocation *actor.Ref
}

// newTrial creates a trial which will try to schedule itself after it receives its first workload.
func newTrial(
	model model.Trial,
	searcher trialSearcherState,
	restored bool,
	rm, logger *actor.Ref,
	db db.DB,
	config expconf.ExperimentConfig,
	warmStartCheckpoint *model.Checkpoint,
	taskSpec *tasks.TaskSpec,
) (*trial, error) {
	t := &trial{
		model:    model,
		searcher: searcher,

		rm:     rm,
		logger: logger,
		db:     db,

		config:              config,
		taskSpec:            taskSpec,
		warmStartCheckpoint: warmStartCheckpoint,
	}

	// TODO: this should be moved to loading model.Trial from the database.
	if restored {
		if err := t.recover(); err != nil {
			return nil, err
		}
	}
	return t, nil
}

func (t *trial) Receive(ctx *actor.Context) error {
	switch msg := ctx.Message().(type) {
	case actor.PreStart:
		ctx.AddLabel("task-run-id", t.runID)
		return t.maybeAllocateTask(ctx)
	case actor.PostStop:
		if !model.TerminalStates[t.model.State] {
			return t.transition(ctx, model.ErrorState)
		}
		return nil
	case actor.ChildStopped:
		if t.allocation != nil && t.runID == mustParseTrialRunID(msg.Child) {
			return t.allocationExited(ctx, &task.AllocationExited{
				Err: errors.New("trial allocation exited without reporting"),
			})
		}
	case actor.ChildFailed:
		if t.allocation != nil && t.runID == mustParseTrialRunID(msg.Child) {
			return t.allocationExited(ctx, &task.AllocationExited{
				Err: errors.Wrapf(msg.Error, "trial allocation failed"),
			})
		}

	case model.State:
		return t.patchState(ctx, msg)
	case trialSearcherState:
		t.searcher = msg
		switch {
		case !t.searcher.Complete:
			return t.maybeAllocateTask(ctx)
		case t.searcher.Complete && t.searcher.Closed:
			return t.patchState(ctx, model.StoppingCompletedState)
		}
		return nil

	case task.BuildTaskSpec:
		if spec, err := t.buildTaskSpec(ctx); err != nil {
			ctx.Respond(err)
		} else {
			ctx.Respond(spec)
		}
	case *task.AllocationExited:
		return t.allocationExited(ctx, msg)
	case sproto.ContainerLog:
		if log, err := t.enrichTrialLog(model.TrialLog{
			ContainerID: ptrs.StringPtr(string(msg.Container.ID)),
			Log:         ptrs.StringPtr(msg.Message()),
			Level:       msg.Level,
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
	runID, restarts, err := t.db.TrialRunIDAndRestarts(t.model.ID)
	if err != nil {
		return errors.Wrap(err, "restoring old trial state")
	}
	t.runID = runID + 1
	t.restarts = restarts
	return nil
}

// To change in testing.
var taskAllocator = task.NewAllocation

// maybeAllocateTask checks if the trial should allocate state and allocates it if so.
func (t *trial) maybeAllocateTask(ctx *actor.Context) error {
	if !(t.allocation == nil && !t.searcher.Complete && t.model.State == model.ActiveState) {
		ctx.Log().Debugf("decided not to allocate trial: "+
			"allocation exists %v, t.model.State=%v, searcher.Create=%v, searcher.Op=%v, " +
			"searcher.Complete=%v, searcher.Closed=%v",
			t.allocation != nil, t.model.State, t.searcher.Create.String(), t.searcher.Op.String(),
			t.searcher.Complete, t.searcher.Closed,
		)
		return nil
	}

	ctx.Log().Info("decided to allocate trial")
	t.allocation, _ = ctx.ActorOf(t.runID, taskAllocator(sproto.AllocateRequest{
		AllocationID: model.NewAllocationID(fmt.Sprintf("%s.%d", t.model.TaskID, t.runID)),
		TaskID:       t.model.TaskID,
		Name:         fmt.Sprintf("Trial %d (Experiment %d)", t.model.ID, t.model.ExperimentID),
		TaskActor:    ctx.Self(),
		Group:        ctx.Self().Parent(),

		SlotsNeeded:  t.config.Resources().SlotsPerTrial(),
		Label:        t.config.Resources().AgentLabel(),
		ResourcePool: t.config.Resources().ResourcePool(),
		FittingRequirements: sproto.FittingRequirements{
			SingleAgent: false,
		},

		Preemptible:  true,
		DoRendezvous: true,
	}, t.db, t.rm))
	return nil
}

func (t *trial) buildTaskSpec(ctx *actor.Context) (tasks.TaskSpec, error) {
	if t.generatedKeys == nil {
		generatedKeys, err := ssh.GenerateKey(t.taskSpec.SSHRsaSize, nil)
		if err != nil {
			return tasks.TaskSpec{}, errors.Wrap(err, "failed to generate keys for trial")
		}
		t.generatedKeys = &generatedKeys
	}

	t.runID++
	if err := t.db.UpdateTrialRunID(t.model.ID, t.runID); err != nil {
		return tasks.TaskSpec{}, errors.Wrap(err, "failed to save trial run ID")
	}

	var latestBatch int
	latestCheckpoint, err := t.db.LatestCheckpointForTrial(t.model.ID)
	switch {
	case err != nil:
		return tasks.TaskSpec{}, errors.Wrapf(err, "failed to query latest checkpoint for trial")
	case latestCheckpoint == nil:
		latestCheckpoint = t.warmStartCheckpoint
	default:
		latestBatch = latestCheckpoint.TotalBatches
	}

	return tasks.TrialSpec{
		Base: *t.taskSpec,

		ExperimentID:     t.model.ExperimentID,
		TrialID:          t.model.ID,
		TrialRunID:       t.runID,
		ExperimentConfig: schemas.Copy(t.config).(expconf.ExperimentConfig),
		HParams:          t.searcher.Create.Hparams,
		TrialSeed:        t.searcher.Create.TrialSeed,
		LatestBatch:      latestBatch,
		LatestCheckpoint: latestCheckpoint,
	}.ToTaskSpec(t.generatedKeys), nil
}

// allocationExited cleans up after an allocation exit and exits permanently or reallocates.
func (t *trial) allocationExited(ctx *actor.Context, exit *task.AllocationExited) error {
	if err := t.allocation.AwaitTermination(); err != nil {
		ctx.Log().WithError(err).Error("trial allocation failed")
	}
	t.allocation = nil

	// Decide if this is permanent.
	switch {
	case model.StoppingStates[t.model.State]:
		if exit.Err != nil {
			return t.transition(ctx, model.ErrorState)
		}
		return t.transition(ctx, model.StoppingToTerminalStates[t.model.State])
	case t.searcher.Complete && t.searcher.Closed:
		if exit.Err != nil {
			return t.transition(ctx, model.ErrorState)
		}
		return t.transition(ctx, model.CompletedState)
	case exit.Err != nil && !aproto.IsRestartableSystemError(exit.Err):
		ctx.Log().
			WithError(exit.Err).
			Errorf("trial failed (restart %d/%d)", t.restarts, t.config.MaxRestarts())
		t.restarts++
		if err := t.db.UpdateTrialRestarts(t.model.ID, t.restarts); err != nil {
			return err
		}
		if t.restarts > t.config.MaxRestarts() {
			return t.transition(ctx, model.ErrorState)
		}
	case exit.UserRequestedStop:
		ctx.Tell(ctx.Self().Parent(), trialReportEarlyExit{
			requestID: t.searcher.Create.RequestID,
			reason:    model.UserCanceled,
		})
		return t.transition(ctx, model.CompletedState)
	}

	// Maybe reschedule.
	return errors.Wrap(t.maybeAllocateTask(ctx), "failed to reschedule trial")
}

// patchState decide if the state patch is valid. If so, we'll transition the trial.
func (t *trial) patchState(ctx *actor.Context, state model.State) error {
	switch {
	case model.TerminalStates[t.model.State]:
		ctx.Log().Infof("ignoring transition in terminal state (%s -> %s)", t.model.State, state)
		return nil
	case model.TerminalStates[state]:
		ctx.Log().Infof("ignoring patch to terminal state %s", state)
		return nil
	case t.model.State == state: // Order is important, else below will prevent re-sending kills.
		ctx.Log().Infof("resending actions for transition for %s", t.model.State)
		return t.transition(ctx, state)
	case model.StoppingStates[t.model.State] && !model.TrialTransitions[t.model.State][state]:
		ctx.Log().Infof("ignoring patch to less severe stopping state (%s)", state)
		return nil
	default:
		ctx.Log().Debugf("patching state after request (%s)", state)
		return t.transition(ctx, state)
	}
}

// transition the trial by rectifying the desired state with our actual state to determined
// a target state, and then propogating the appropriate signals to the allocation if there is any.
func (t *trial) transition(ctx *actor.Context, state model.State) error {
	if t.model.State != state {
		ctx.Log().Infof("trial changed from state %s to %s", t.model.State, state)
		if err := t.db.UpdateTrial(t.model.ID, state); err != nil {
			return errors.Wrap(err, "updating trial with end state")
		}
		t.model.State = state
	}

	// Rectify our state and the allocation state with the transition.
	switch {
	case t.model.State == model.ActiveState:
		return t.maybeAllocateTask(ctx)
	case t.model.State == model.PausedState:
		if t.allocation != nil {
			ctx.Log().Infof("decided to %s trial due to pause", task.Terminate)
			ctx.Tell(t.allocation, task.Terminate)
		}
	case model.StoppingStates[t.model.State]:
		switch {
		case t.allocation == nil:
			ctx.Log().Info("stopping trial before resources are requested")
			return t.transition(ctx, model.StoppingToTerminalStates[t.model.State])
		default:
			if action, ok := map[model.State]task.AllocationSignal{
				model.StoppingCanceledState: task.Terminate,
				model.StoppingKilledState:   task.Kill,
				model.StoppingErrorState:    task.Kill,
			}[t.model.State]; ok {
				ctx.Log().Infof("decided to %s trial", action)
				ctx.Tell(t.allocation, action)
			}
		}
	case model.TerminalStates[t.model.State]:
		if t.model.State == model.ErrorState {
			ctx.Tell(ctx.Self().Parent(), trialReportEarlyExit{
				requestID: t.searcher.Create.RequestID,
				reason:    model.Errored,
			})
		}
		ctx.Self().Stop()
	default:
		panic(fmt.Errorf("unmatched state in transition %s", t.model.State))
	}
	return nil
}

func (t *trial) enrichTrialLog(log model.TrialLog) (model.TrialLog, error) {
	log.TrialID = t.model.ID
	log.Message += "\n"
	if log.Log != nil && !strings.HasSuffix(*log.Log, "\n") {
		log.Log = ptrs.StringPtr(*log.Log + "\n")
	}
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

func mustParseTrialRunID(child *actor.Ref) int {
	idStr := child.Address().Local()
	id, err := strconv.Atoi(idStr)
	if err != nil {
		panic(errors.Wrapf(err, "could not parse run id %s", idStr))
	}
	return id
}
