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
	id                int
	taskID            model.TaskID
	jobID             model.JobID
	jobSubmissionTime time.Time
	idSet             bool
	experimentID      int

	// System dependencies.
	rm     *actor.Ref
	logger *actor.Ref
	db     db.DB

	// Fields that are essentially configuration for the trial.
	config              expconf.ExperimentConfig
	taskSpec            *tasks.TaskSpec
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

	// a ref to the current allocation
	allocation *actor.Ref
}

// newTrial creates a trial which will try to schedule itself after it receives its first workload.
func newTrial(
	taskID model.TaskID,
	jobID model.JobID,
	jobSubmissionTime time.Time,
	experimentID int,
	initialState model.State,
	searcher trialSearcherState,
	rm, logger *actor.Ref,
	db db.DB,
	config expconf.ExperimentConfig,
	warmStartCheckpoint *model.Checkpoint,
	taskSpec *tasks.TaskSpec,
) *trial {
	return &trial{
		taskID:            taskID,
		jobID:             jobID,
		jobSubmissionTime: jobSubmissionTime,
		experimentID:      experimentID,
		state:             initialState,
		searcher:          searcher,

		rm:     rm,
		logger: logger,
		db:     db,

		config:              config,
		taskSpec:            taskSpec,
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
		return t.maybeAllocateTask(ctx)
	case actor.PostStop:
		if !t.idSet {
			return nil
		}
		if !model.TerminalStates[t.state] {
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
	runID, restarts, err := t.db.TrialRunIDAndRestarts(t.id)
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

	ctx.Log().Info("decided to allocate trial")
	t.allocation, _ = ctx.ActorOf(t.runID, taskAllocator(sproto.AllocateRequest{
		AllocationID:      model.NewAllocationID(fmt.Sprintf("%s.%d", t.taskID, t.runID)),
		TaskID:            t.taskID,
		JobID:             &t.jobID,
		JobSubmissionTime: &t.jobSubmissionTime,
		Name:              name,
		TaskActor:         ctx.Self(),
		Group:             ctx.Self().Parent(),

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
	// It is possible the trial state changed from active since we decided to launch this
	// allocation but that, in quick succession, the resource manager provided the allocation with
	// resources and we sent the cancel message. In this case, rather than let the allocation start,
	// we send it a cancellation. If this is the first allocation, it will also prevent us from
	// adding a trial when we are not active, which breaks some other invariants.
	if t.state != model.ActiveState {
		return tasks.TaskSpec{}, task.ErrAlreadyCancelled{}
	}

	if t.generatedKeys == nil {
		generatedKeys, err := ssh.GenerateKey(t.taskSpec.SSHRsaSize, nil)
		if err != nil {
			return tasks.TaskSpec{}, errors.Wrap(err, "failed to generate keys for trial")
		}
		t.generatedKeys = &generatedKeys
	}

	if !t.idSet {
		modelTrial := model.NewTrial(
			t.jobID,
			t.taskID,
			t.searcher.Create.RequestID,
			t.experimentID,
			model.JSONObj(t.searcher.Create.Hparams),
			t.warmStartCheckpoint,
			int64(t.searcher.Create.TrialSeed))

		if err := t.db.AddTrial(modelTrial); err != nil {
			return tasks.TaskSpec{}, errors.Wrap(err, "failed to save trial to database")
		}
		t.id = modelTrial.ID
		t.idSet = true
		ctx.AddLabel("trial-id", t.id)
		ctx.Tell(t.rm, sproto.SetTaskName{
			Name:        fmt.Sprintf("Trial %d (Experiment %d)", t.id, t.experimentID),
			TaskHandler: t.allocation,
		})
		ctx.Tell(ctx.Self().Parent(), trialCreated{requestID: t.searcher.Create.RequestID})
	}

	t.runID++
	if err := t.db.UpdateTrialRunID(t.id, t.runID); err != nil {
		return tasks.TaskSpec{}, errors.Wrap(err, "failed to save trial run ID")
	}

	var latestBatch int
	latestCheckpoint, err := t.db.LatestCheckpointForTrial(t.id)
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

		ExperimentID:     t.experimentID,
		TrialID:          t.id,
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
	case model.StoppingStates[t.state]:
		if exit.Err != nil {
			return t.transition(ctx, model.ErrorState)
		}
		return t.transition(ctx, model.StoppingToTerminalStates[t.state])
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
		if err := t.db.UpdateTrialRestarts(t.id, t.restarts); err != nil {
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
	case model.TerminalStates[t.state]:
		ctx.Log().Infof("ignoring transition in terminal state (%s -> %s)", t.state, state)
		return nil
	case model.TerminalStates[state]:
		ctx.Log().Infof("ignoring patch to terminal state %s", state)
		return nil
	case t.state == state: // Order is important, else below will prevent re-sending kills.
		ctx.Log().Infof("resending actions for transition for %s", t.state)
		return t.transition(ctx, state)
	case model.StoppingStates[t.state] && !model.TrialTransitions[t.state][state]:
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
	if t.state != state {
		ctx.Log().Infof("trial changed from state %s to %s", t.state, state)
		if t.idSet {
			if err := t.db.UpdateTrial(t.id, state); err != nil {
				return errors.Wrap(err, "updating trial with end state")
			}
		}
		t.state = state
	}

	// Rectify our state and the allocation state with the transition.
	switch {
	case t.state == model.ActiveState:
		return t.maybeAllocateTask(ctx)
	case t.state == model.PausedState:
		if t.allocation != nil {
			ctx.Log().Infof("decided to %s trial due to pause", task.Terminate)
			ctx.Tell(t.allocation, task.Terminate)
		}
	case model.StoppingStates[t.state]:
		switch {
		case t.allocation == nil:
			ctx.Log().Info("stopping trial before resources are requested")
			return t.transition(ctx, model.StoppingToTerminalStates[t.state])
		default:
			if action, ok := map[model.State]task.AllocationSignal{
				model.StoppingCanceledState: task.Terminate,
				model.StoppingKilledState:   task.Kill,
				model.StoppingErrorState:    task.Kill,
			}[t.state]; ok {
				ctx.Log().Infof("decided to %s trial", action)
				ctx.Tell(t.allocation, action)
			}
		}
	case model.TerminalStates[t.state]:
		if t.state == model.ErrorState {
			ctx.Tell(ctx.Self().Parent(), trialReportEarlyExit{
				requestID: t.searcher.Create.RequestID,
				reason:    model.Errored,
			})
		}
		ctx.Self().Stop()
	default:
		panic(fmt.Errorf("unmatched state in transition %s", t.state))
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
