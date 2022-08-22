package internal

import (
	"context"
	"fmt"
	"regexp"
	"strconv"
	"strings"
	"time"

	"github.com/determined-ai/determined/master/internal/config"
	"github.com/determined-ai/determined/master/internal/prom"
	"github.com/determined-ai/determined/master/internal/rm"
	"github.com/determined-ai/determined/master/internal/task"

	"github.com/determined-ai/determined/master/pkg/actor/actors"
	"github.com/determined-ai/determined/master/pkg/logger"
	"github.com/determined-ai/determined/master/pkg/mathx"
	"github.com/determined-ai/determined/master/pkg/ptrs"

	"github.com/pkg/errors"

	"github.com/determined-ai/determined/master/internal/db"
	"github.com/determined-ai/determined/master/internal/sproto"
	"github.com/determined-ai/determined/master/pkg/actor"
	"github.com/determined-ai/determined/master/pkg/model"
	"github.com/determined-ai/determined/master/pkg/schemas"
	"github.com/determined-ai/determined/master/pkg/schemas/expconf"
	"github.com/determined-ai/determined/master/pkg/ssh"
	"github.com/determined-ai/determined/master/pkg/tasks"
)

// A list of errors for which we don't want to attempt any retries of the experiment.
// These are errors that no matter how many times we retry, the outcome will still result
// in the same error.
var nonRetryableErrors = []*regexp.Regexp{
	// This error is typically seen when you request resources that SLURM is not able to satisfy.
	regexp.MustCompile("sbatch: error: Batch job submission failed"),
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
	id                int
	taskID            model.TaskID
	jobID             model.JobID
	jobSubmissionTime time.Time
	idSet             bool
	experimentID      int
	restored          bool

	// System dependencies.
	taskLogger *task.Logger
	db         db.DB
	rm         rm.ResourceManager

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
	// a note of the user initated exit reason, if any.
	userInitiatedExit *model.ExitedReason

	logCtx logger.Context
}

// newTrial creates a trial which will try to schedule itself after it receives its first workload.
func newTrial(
	logCtx logger.Context,
	taskID model.TaskID,
	jobID model.JobID,
	jobSubmissionTime time.Time,
	experimentID int,
	initialState model.State,
	searcher trialSearcherState,
	taskLogger *task.Logger,
	rm rm.ResourceManager,
	db db.DB,
	config expconf.ExperimentConfig,
	warmStartCheckpoint *model.Checkpoint,
	taskSpec *tasks.TaskSpec,
	restored bool,
) *trial {
	return &trial{
		taskID:            taskID,
		jobID:             jobID,
		jobSubmissionTime: jobSubmissionTime,
		experimentID:      experimentID,
		state:             initialState,
		searcher:          searcher,

		taskLogger: taskLogger,
		db:         db,
		rm:         rm,

		config:              config,
		taskSpec:            taskSpec,
		warmStartCheckpoint: warmStartCheckpoint,

		logCtx: logger.MergeContexts(logCtx, logger.Context{
			"task-id":   taskID,
			"task-type": model.TaskTypeTrial,
		}),
		restored: restored,
	}
}

// Returns true if the error message matches one of the errors in the non-retryable list.
func isNonRetryableError(err error) bool {
	for _, nonRetryableError := range nonRetryableErrors {
		if nonRetryableError.MatchString(err.Error()) {
			return true
		}
	}

	return false
}

func (t *trial) Receive(ctx *actor.Context) error {
	switch msg := ctx.Message().(type) {
	case actor.PreStart:
		if t.idSet {
			if err := t.recover(); err != nil {
				return err
			}
			t.logCtx = logger.MergeContexts(t.logCtx, logger.Context{
				"trial-id":     t.id,
				"trial-run-id": t.runID,
			})
		}
		ctx.AddLabels(t.logCtx)

		return t.maybeAllocateTask(ctx)
	case actor.PostStop:
		if !t.idSet {
			return nil
		}
		if !model.TerminalStates[t.state] {
			return t.transition(ctx, model.StateWithReason{
				State:               model.ErrorState,
				InformationalReason: "trial did not finish properly",
			})
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
		return t.patchState(ctx, model.StateWithReason{State: msg})
	case model.StateWithReason:
		return t.patchState(ctx, msg)
	case trialSearcherState:
		t.searcher = msg
		switch {
		case !t.searcher.Complete:
			return t.maybeAllocateTask(ctx)
		case t.searcher.Complete && t.searcher.Closed:
			return t.patchState(ctx, model.StateWithReason{
				State:               model.StoppingCompletedState,
				InformationalReason: "hp search is finished",
			})
		}
		return nil
	case sproto.ChangeRP:
		resources := t.config.Resources()
		resources.SetResourcePool(msg.ResourcePool)
		t.config.SetResources(resources)
		if t.allocation != nil {
			ctx.Tell(t.allocation, msg)
		}
	case task.BuildTaskSpec:
		if spec, err := t.buildTaskSpec(ctx); err != nil {
			ctx.Respond(err)
		} else {
			ctx.Respond(spec)
		}
	case userInitiatedEarlyExit:
		if err := t.handleUserInitiatedStops(ctx, msg); err != nil {
			ctx.Respond(err)
		}
	case *task.AllocationExited:
		if t.allocation != nil && t.runID == mustParseTrialRunID(ctx.Sender()) {
			return t.allocationExited(ctx, msg)
		}
	case sproto.ContainerLog:
		if log, err := t.enrichTaskLog(model.TaskLog{
			ContainerID: ptrs.Ptr(string(msg.Container.ID)),
			Log:         msg.Message(),
			Level:       msg.Level,
		}); err != nil {
			ctx.Log().WithError(err).Warn("dropping container log")
		} else {
			t.taskLogger.Insert(ctx, log)
		}
	case model.TaskLog:
		if log, err := t.enrichTaskLog(msg); err != nil {
			ctx.Log().WithError(err).Warn("dropping trial log")
		} else {
			t.taskLogger.Insert(ctx, log)
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
	t.runID = runID
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
	t.logCtx = logger.MergeContexts(t.logCtx, logger.Context{"trial-run-id": t.runID})
	ctx.AddLabel("trial-run-id", t.runID)
	if err := t.addTask(); err != nil {
		return err
	}

	restoredAllocation, err := t.maybeRestoreAllocation(ctx)
	if err != nil {
		ctx.Log().WithError(err).Warn("failed to restore trial allocation")
	} else if restoredAllocation != nil {
		ar := sproto.AllocateRequest{
			AllocationID:      restoredAllocation.AllocationID,
			TaskID:            t.taskID,
			JobID:             t.jobID,
			JobSubmissionTime: t.jobSubmissionTime,
			IsUserVisible:     true,
			Name:              name,
			AllocationRef:     ctx.Self(),
			Group:             ctx.Self().Parent(),
			SlotsNeeded:       t.config.Resources().SlotsPerTrial(),
			Label:             t.config.Resources().AgentLabel(),
			ResourcePool:      t.config.Resources().ResourcePool(),
			FittingRequirements: sproto.FittingRequirements{
				SingleAgent: false,
			},

			Preemptible: true,
			Restore:     true,
		}
		ctx.Log().
			WithField("allocation-id", ar.AllocationID).
			Infof("starting restored trial allocation")
		t.allocation, _ = ctx.ActorOf(t.runID, taskAllocator(
			t.logCtx, ar, t.db, t.rm, t.taskLogger,
		))
		return nil
	}

	if err := t.addTask(); err != nil {
		return err
	}

	t.runID++

	ar := sproto.AllocateRequest{
		AllocationID:      model.AllocationID(fmt.Sprintf("%s.%d", t.taskID, t.runID)),
		TaskID:            t.taskID,
		JobID:             t.jobID,
		JobSubmissionTime: t.jobSubmissionTime,
		IsUserVisible:     true,
		Name:              name,
		AllocationRef:     ctx.Self(),
		Group:             ctx.Self().Parent(),

		SlotsNeeded:  t.config.Resources().SlotsPerTrial(),
		Label:        t.config.Resources().AgentLabel(),
		ResourcePool: t.config.Resources().ResourcePool(),
		FittingRequirements: sproto.FittingRequirements{
			SingleAgent: false,
		},

		Preemptible: true,
	}

	ctx.Log().
		WithField("allocation-id", ar.AllocationID).
		Debugf("starting new trial allocation")

	prom.AssociateJobExperiment(t.jobID, strconv.Itoa(t.experimentID), t.config.Labels())
	t.allocation, _ = ctx.ActorOf(t.runID, taskAllocator(t.logCtx, ar, t.db, t.rm, t.taskLogger))
	ctx.Ask(t.allocation, actor.Ping{}).Get()

	return nil
}

const (
	// InvalidHPKillDelay the delay before we forcibly kill a trial that said it had an invalid HP.
	InvalidHPKillDelay = 10 * time.Second
)

func (t *trial) handleUserInitiatedStops(ctx *actor.Context, msg userInitiatedEarlyExit) error {
	switch msg.reason {
	case model.InvalidHP, model.InitInvalidHP:
		t.userInitiatedExit = &msg.reason
		// After a short time, force us to clean up if we're still handling messages.
		actors.NotifyAfter(ctx, InvalidHPKillDelay, model.StoppingKilledState)
		return nil
	case model.UserRequestedStop, model.Errored:
		return fmt.Errorf("should not report special exit reason %s to the master", msg.reason)
	default:
		return actor.ErrUnexpectedMessage(ctx)
	}
}

func (t *trial) addTask() error {
	return t.db.AddTask(&model.Task{
		TaskID:     t.taskID,
		TaskType:   model.TaskTypeTrial,
		StartTime:  t.jobSubmissionTime,
		JobID:      &t.jobID,
		LogVersion: model.CurrentTaskLogVersion,
	})
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
		t.logCtx = logger.MergeContexts(t.logCtx, logger.Context{"trial-id": t.id})
		ctx.AddLabel("trial-id", t.id)
		t.rm.SetAllocationName(ctx, sproto.SetAllocationName{
			Name:          fmt.Sprintf("Trial %d (Experiment %d)", t.id, t.experimentID),
			AllocationRef: t.allocation,
		})
		ctx.Tell(ctx.Self().Parent(), trialCreated{requestID: t.searcher.Create.RequestID})
	}

	if err := t.db.UpdateTrialRunID(t.id, t.runID); err != nil {
		return tasks.TaskSpec{}, errors.Wrap(err, "failed to save trial run ID")
	}

	var stepsCompleted int
	latestCheckpoint, err := t.db.LatestCheckpointForTrial(t.id)
	switch {
	case err != nil:
		return tasks.TaskSpec{}, errors.Wrapf(err, "failed to query latest checkpoint for trial")
	case latestCheckpoint == nil:
		latestCheckpoint = t.warmStartCheckpoint
	default:
		stepsCompleted = latestCheckpoint.StepsCompleted
	}

	return tasks.TrialSpec{
		Base: *t.taskSpec,

		ExperimentID:     t.experimentID,
		TrialID:          t.id,
		TrialRunID:       t.runID,
		ExperimentConfig: schemas.Copy(t.config).(expconf.ExperimentConfig),
		HParams:          t.searcher.Create.Hparams,
		TrialSeed:        t.searcher.Create.TrialSeed,
		StepsCompleted:   stepsCompleted,
		LatestCheckpoint: latestCheckpoint,
	}.ToTaskSpec(t.generatedKeys), nil
}

// allocationExited cleans up after an allocation exit and exits permanently or reallocates.
func (t *trial) allocationExited(ctx *actor.Context, exit *task.AllocationExited) error {
	if err := t.allocation.AwaitTermination(); err != nil {
		ctx.Log().WithError(err).Error("trial allocation failed")
	}
	t.allocation = nil

	prom.DisassociateJobExperiment(t.jobID, strconv.Itoa(t.experimentID), t.config.Labels())

	// Decide if this is permanent.
	switch {
	case model.StoppingStates[t.state]:
		if exit.Err != nil {
			return t.transition(ctx, model.StateWithReason{
				State: model.ErrorState,
				InformationalReason: fmt.Sprintf(
					"trial allocation exited with an error while trial was stopping %v", exit.Err),
			})
		}
		return t.transition(ctx, model.StateWithReason{
			State:               model.StoppingToTerminalStates[t.state],
			InformationalReason: "trial stopped",
		})
	case t.searcher.Complete && t.searcher.Closed:
		if exit.Err != nil {
			return t.transition(ctx, model.StateWithReason{
				State: model.ErrorState,
				InformationalReason: fmt.Sprintf(
					"trial allocation exited with an error but hp search was complete %v", exit.Err),
			})
		}
		return t.transition(ctx, model.StateWithReason{
			State:               model.CompletedState,
			InformationalReason: "hp search is finished",
		})
	case exit.Err != nil && sproto.IsUnrecoverableSystemError(exit.Err):
		ctx.Log().
			WithError(exit.Err).
			Errorf("trial encountered unrecoverable failure")
		return t.transition(ctx, model.StateWithReason{
			State: model.ErrorState,
			InformationalReason: fmt.Sprintf(
				"trial allocation exited with unrecoverable failure %v", exit.Err),
		})
	case exit.Err != nil && isNonRetryableError(exit.Err):
		// These are errors that no matter how many times we retry, the outcome will
		// be the same, so don't bother retrying. Fail right away to allow the user
		// to make any adjustments to the experiment and try again.
		return t.transition(ctx, model.StateWithReason{
			State: model.ErrorState,
			InformationalReason: fmt.Sprintf(
				"trial allocation exited with unrecoverable failure %v", exit.Err),
		})
	case exit.Err != nil && sproto.IsTransientSystemError(exit.Err):
		ctx.Log().
			WithError(exit.Err).
			Errorf("trial encountered transient system error")
	case exit.Err != nil && !sproto.IsTransientSystemError(exit.Err):
		ctx.Log().
			WithError(exit.Err).
			Errorf("trial failed (restart %d/%d)", t.restarts, t.config.MaxRestarts())
		t.restarts++
		if err := t.db.UpdateTrialRestarts(t.id, t.restarts); err != nil {
			return err
		}
		if t.restarts > t.config.MaxRestarts() {
			return t.transition(ctx, model.StateWithReason{
				State:               model.ErrorState,
				InformationalReason: "trial exceeded max restarts",
			})
		}
	case exit.UserRequestedStop:
		ctx.Tell(ctx.Self().Parent(), trialReportEarlyExit{
			requestID: t.searcher.Create.RequestID,
			reason:    model.UserRequestedStop,
		})
		return t.transition(ctx, model.StateWithReason{
			State:               model.CompletedState,
			InformationalReason: "trial exited early due to a user requested stop",
		})
	case t.userInitiatedExit != nil:
		ctx.Tell(ctx.Self().Parent(), trialReportEarlyExit{
			requestID: t.searcher.Create.RequestID,
			reason:    *t.userInitiatedExit,
		})
		return t.transition(ctx, model.StateWithReason{
			State: model.CompletedState,
			InformationalReason: fmt.Sprintf(
				"trial exited early with reason: %v", *t.userInitiatedExit),
		})
	}

	// Maybe reschedule.
	return errors.Wrap(t.maybeAllocateTask(ctx), "failed to reschedule trial")
}

// patchState decide if the state patch is valid. If so, we'll transition the trial.
func (t *trial) patchState(ctx *actor.Context, s model.StateWithReason) error {
	switch {
	case model.TerminalStates[t.state]:
		ctx.Log().Infof("ignoring transition in terminal state (%s -> %s)", t.state, s.State)
		return nil
	case model.TerminalStates[s.State]:
		ctx.Log().Infof("ignoring patch to terminal state %s", s.State)
		return nil
	case t.state == s.State: // Order is important, else below will prevent re-sending kills.
		ctx.Log().Infof("resending actions for transition for %s", t.state)
		return t.transition(ctx, s)
	case model.StoppingStates[t.state] && !model.TrialTransitions[t.state][s.State]:
		ctx.Log().Infof("ignoring patch to less severe stopping state (%s)", s.State)
		return nil
	default:
		ctx.Log().Debugf("patching state after request (%s)", s.State)
		return t.transition(ctx, s)
	}
}

// transition the trial by rectifying the desired state with our actual state to determined
// a target state, and then propogating the appropriate signals to the allocation if there is any.
func (t *trial) transition(ctx *actor.Context, s model.StateWithReason) error {
	if t.state != s.State {
		ctx.Log().Infof("trial changed from state %s to %s", t.state, s.State)
		if t.idSet {
			if err := t.db.UpdateTrial(t.id, s.State); err != nil {
				return errors.Wrap(err, "updating trial with end state")
			}
		}
		t.state = s.State
	}

	// Rectify our state and the allocation state with the transition.
	switch {
	case t.state == model.ActiveState:
		return t.maybeAllocateTask(ctx)
	case t.state == model.PausedState:
		if t.allocation != nil {
			ctx.Log().Infof("decided to %s trial due to pause", sproto.TerminateAllocation)
			ctx.Tell(t.allocation, sproto.AllocationSignalWithReason{
				AllocationSignal:    sproto.TerminateAllocation,
				InformationalReason: s.InformationalReason,
			})
		}
	case model.StoppingStates[t.state]:
		switch {
		case t.allocation == nil:
			ctx.Log().Info("stopping trial before resources are requested")
			return t.transition(ctx, model.StateWithReason{
				State:               model.StoppingToTerminalStates[t.state],
				InformationalReason: s.InformationalReason,
			})
		default:
			if action, ok := map[model.State]sproto.AllocationSignal{
				model.StoppingCanceledState: sproto.TerminateAllocation,
				model.StoppingKilledState:   sproto.KillAllocation,
				model.StoppingErrorState:    sproto.KillAllocation,
			}[t.state]; ok {
				ctx.Log().Infof("decided to %s trial", action)
				ctx.Tell(t.allocation, sproto.AllocationSignalWithReason{
					AllocationSignal:    action,
					InformationalReason: s.InformationalReason,
				})
			}
		}
	case model.TerminalStates[t.state]:
		switch t.state {
		case model.ErrorState:
			ctx.Tell(ctx.Self().Parent(), trialReportEarlyExit{
				requestID: t.searcher.Create.RequestID,
				reason:    model.Errored,
			})
		case model.CanceledState:
			ctx.Tell(ctx.Self().Parent(), trialReportEarlyExit{
				requestID: t.searcher.Create.RequestID,
				reason:    model.UserCanceled,
			})
		}
		ctx.Self().Stop()
	default:
		panic(fmt.Errorf("unmatched state in transition %s", t.state))
	}
	return nil
}

func (t *trial) enrichTaskLog(log model.TaskLog) (model.TaskLog, error) {
	if !t.idSet {
		return model.TaskLog{}, fmt.Errorf(
			"cannot handle trial log before ID is set: %v", log)
	}
	log.TaskID = string(t.taskID)

	if log.Timestamp == nil {
		log.Timestamp = ptrs.Ptr(time.Now().UTC())
	}
	if log.Level == nil {
		log.Level = ptrs.Ptr("INFO")
	}
	if log.Source == nil {
		log.Source = ptrs.Ptr("master")
	}
	if log.StdType == nil {
		log.StdType = ptrs.Ptr("stdout")
	}

	log.Log += "\n"

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

func (t *trial) maybeRestoreAllocation(ctx *actor.Context) (*model.Allocation, error) {
	if !t.restored || !config.IsReattachEnabled() {
		return nil, nil
	}

	// Do we have an open allocation?
	var allocations []model.Allocation
	err := db.Bun().NewSelect().Model(&allocations).
		Where("task_id = ?", t.taskID).
		Where("start_time IS NOT NULL").
		Where("end_time IS NULL").
		Scan(context.TODO())
	if err != nil {
		return nil, err
	}

	openAllocs := len(allocations)
	switch {
	case openAllocs == 0:
		return nil, nil
	case openAllocs == 1:
		allocation := &allocations[0]
		if !config.IsReattachEnabledForRP(allocation.ResourcePool) {
			return nil, nil
		}

		return allocation, nil
	case openAllocs > 1:
		const maxAllocsToLog int = 3
		allocIDs := make([]string, 0, maxAllocsToLog)
		for _, alloc := range allocations[0:mathx.Min(len(allocations), maxAllocsToLog)] {
			allocIDs = append(allocIDs, alloc.AllocationID.String())
		}
		return nil, errors.New(
			fmt.Sprintf(
				"discovered %d open allocations on restore: %s",
				len(allocations),
				strings.Join(allocIDs, " "),
			),
		)
	default:
		return nil, errors.New(
			fmt.Sprintf(
				"discovered %d open allocations on restore",
				len(allocations),
			),
		)
	}
}
