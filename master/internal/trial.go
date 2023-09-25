package internal

import (
	"context"
	"fmt"
	"regexp"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/pkg/errors"
	"github.com/sirupsen/logrus"

	"github.com/determined-ai/determined/master/internal/db"
	"github.com/determined-ai/determined/master/internal/prom"
	"github.com/determined-ai/determined/master/internal/rm"
	"github.com/determined-ai/determined/master/internal/sproto"
	"github.com/determined-ai/determined/master/internal/task"
	"github.com/determined-ai/determined/master/pkg/actor"
	"github.com/determined-ai/determined/master/pkg/logger"
	"github.com/determined-ai/determined/master/pkg/mathx"
	"github.com/determined-ai/determined/master/pkg/model"
	"github.com/determined-ai/determined/master/pkg/ptrs"
	"github.com/determined-ai/determined/master/pkg/schemas"
	"github.com/determined-ai/determined/master/pkg/schemas/expconf"
	"github.com/determined-ai/determined/master/pkg/ssh"
	"github.com/determined-ai/determined/master/pkg/syncx/waitgroupx"
	"github.com/determined-ai/determined/master/pkg/tasks"
)

const (
	// InvalidHPKillDelay the delay before we forcibly kill a trial that said it had an invalid HP.
	InvalidHPKillDelay = 10 * time.Second
)

// A list of errors for which we don't want to attempt any retries of the experiment.
// These are errors that no matter how many times we retry, the outcome will still result
// in the same error.
var nonRetryableErrors = []*regexp.Regexp{
	// This error is typically seen when you request resources that SLURM is not able to satisfy.
	regexp.MustCompile("sbatch: error: Batch job submission failed"),
}

type trialExitCallback func(model.RequestID, *model.ExitedReason)

// A trial is a task actor which is responsible for handling:
//   - messages from the resource manager,
//   - messages from the experiment,
//   - messages from the trial container(s), and
//   - keeping the trial table of the database up-to-date.
//
// The trial's desired state is dictated to it by the experiment, searcher and user; they push
// it to states like 'ACTIVE', 'PAUSED' and kill or wake it when more work is available. It takes
// this information and works with the resource manager, allocation, etc, to push us towards
// a terminal state, by requesting resources, managing them and restarting them on failures.
type trial struct {
	mu sync.Mutex
	wg waitgroupx.Group

	id                int
	taskID            model.TaskID
	jobID             model.JobID
	jobSubmissionTime time.Time
	idSet             bool
	experimentID      int
	restored          bool

	// System dependencies.
	db     db.DB
	rm     rm.ResourceManager
	syslog *logrus.Entry
	system *actor.System
	parent *actor.Ref

	// Fields that are essentially configuration for the trial.
	config              expconf.ExperimentConfig
	taskSpec            *tasks.TaskSpec
	generatedKeys       ssh.PrivateAndPublicKeys
	warmStartCheckpoint *model.Checkpoint

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
	allocationID *model.AllocationID
	// a note of the user initated exit reason, if any.
	userInitiatedExit *model.ExitedReason

	logCtx logger.Context

	exitCallback trialExitCallback
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
	rm rm.ResourceManager,
	pgDB db.DB,
	config expconf.ExperimentConfig,
	warmStartCheckpoint *model.Checkpoint,
	taskSpec *tasks.TaskSpec,
	generatedKeys ssh.PrivateAndPublicKeys,
	restored bool,
	id *int,
	continueFromTrialID *int,
	system *actor.System,
	parent *actor.Ref,
	exitCallback trialExitCallback,
) (t *trial, err error) {
	t = &trial{
		wg: waitgroupx.WithContext(context.Background()),

		taskID:            taskID,
		jobID:             jobID,
		jobSubmissionTime: jobSubmissionTime,
		experimentID:      experimentID,
		state:             initialState,
		searcher:          searcher,
		parent:            parent,

		db:     pgDB,
		rm:     rm,
		syslog: logrus.WithField("component", "trial"),
		system: system,

		config:              config,
		taskSpec:            taskSpec,
		generatedKeys:       generatedKeys,
		warmStartCheckpoint: warmStartCheckpoint,

		logCtx: logger.MergeContexts(logCtx, logger.Context{
			"task-id":   taskID,
			"task-type": model.TaskTypeTrial,
		}),
		restored: restored,

		exitCallback: exitCallback,
	}
	switch {
	case id != nil:
		t.id = *id
		t.idSet = true
		if err := t.recover(); err != nil {
			return nil, fmt.Errorf("recovering trial in prestart: %w", err)
		}
	case continueFromTrialID != nil:
		if err := t.continueSetup(continueFromTrialID); err != nil {
			return nil, fmt.Errorf("continue trial in prestart: %w", err)
		}

	default:
		if err := t.create(); err != nil {
			return nil, fmt.Errorf("persisting trial in prestart: %w", err)
		}
	}

	t.logCtx = logger.MergeContexts(t.logCtx, logger.Context{
		"trial-id":     t.id,
		"trial-run-id": t.runID,
	})

	err = t.maybeAllocateTask()
	if err != nil {
		return nil, fmt.Errorf("initial allocation: %w", err)
	}
	return t, nil
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

func (t *trial) exit(reason *model.ExitedReason) {
	if err := t.close(); err != nil {
		t.syslog.WithError(err).Error("error closing trial")
	}
	go t.exitCallback(t.searcher.Create.RequestID, reason)
}

func (t *trial) close() error {
	t.wg.Close()
	if !t.idSet {
		return nil
	}
	if !model.TerminalStates[t.state] {
		if t.allocationID != nil {
			err := task.DefaultService.Signal(*t.allocationID, task.KillAllocation, "trial crashed")
			if err == nil {
				task.DefaultService.AwaitTermination(*t.allocationID)
			}
		}
		return t.transition(model.StateWithReason{
			State:               model.ErrorState,
			InformationalReason: "trial did not finish properly",
		})
	}
	return nil
}

func (t *trial) PatchState(req model.StateWithReason) error {
	t.mu.Lock()
	defer t.mu.Unlock()

	return t.patchState(req)
}

func (t *trial) PatchSearcherState(req trialSearcherState) error {
	t.mu.Lock()
	defer t.mu.Unlock()

	t.searcher = req
	switch {
	case !t.searcher.Complete:
		return t.maybeAllocateTask()
	case t.searcher.Complete && t.searcher.Closed:
		return t.patchState(model.StateWithReason{
			State:               model.StoppingCompletedState,
			InformationalReason: "hp search is finished",
		})
	}
	return nil
}

func (t *trial) PatchRP(rp string) {
	t.mu.Lock()
	defer t.mu.Unlock()

	resources := t.config.Resources()
	resources.SetResourcePool(rp)
	t.config.SetResources(resources)
	if t.allocationID != nil {
		err := task.DefaultService.Signal(
			*t.allocationID,
			task.TerminateAllocation,
			"allocation resource pool changed",
		)
		if err != nil {
			t.syslog.WithError(err).Warn("could not preempt allocation to change rp")
		}
	}
}

func (t *trial) SetUserInitiatedEarlyExit(req userInitiatedEarlyExit) error {
	t.mu.Lock()
	defer t.mu.Unlock()

	switch req.reason {
	case model.InvalidHP, model.InitInvalidHP:
		t.userInitiatedExit = &req.reason
		// After a short time, force us to clean up if we're still handling messages.
		t.wg.Go(func(ctx context.Context) {
			tmr := time.NewTimer(InvalidHPKillDelay)
			defer tmr.Stop()

			select {
			case <-tmr.C:
				err := t.PatchState(model.StateWithReason{
					State:               model.StoppingKilledState,
					InformationalReason: "timeout after user initiated early exit",
				})
				if err != nil {
					t.syslog.WithError(err).Error("error patching state")
				}
			case <-ctx.Done():
			}
		})
		return nil
	case model.UserRequestedStop, model.Errored:
		return fmt.Errorf("should not report special exit reason %s to the master", req.reason)
	default:
		return fmt.Errorf("unhandled early exit reason: %s", req.reason)
	}
}

func (t *trial) create() error {
	m := model.NewTrial(
		t.state,
		t.searcher.Create.RequestID,
		t.experimentID,
		model.JSONObj(t.searcher.Create.Hparams),
		t.warmStartCheckpoint,
		int64(t.searcher.Create.TrialSeed),
	)

	err := t.addTask()
	if err != nil {
		return err
	}

	err = db.AddTrial(context.TODO(), m, t.taskID)
	if err != nil {
		return errors.Wrap(err, "failed to save trial to database")
	}

	t.id = m.ID
	t.idSet = true
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

// / continueSetup sets trial state up so that it will continue training.
func (t *trial) continueSetup(continueFromTrialID *int) error {
	if continueFromTrialID == nil {
		return fmt.Errorf("continueFromTrialID is nil trial %+v", t)
	}

	t.id = *continueFromTrialID
	t.idSet = true

	if err := t.recover(); err != nil {
		return fmt.Errorf("recovering trial state: %w", err)
	}

	trialIDTaskIDs, err := db.TrialTaskIDsByTrialID(context.TODO(), t.id)
	if err != nil {
		return fmt.Errorf("getting previous task IDs for trial: %w", err)
	}

	t.taskID = model.TaskID(fmt.Sprintf("%s-%d", t.taskID, len(trialIDTaskIDs)))

	err = t.addTask()
	if err != nil {
		return err
	}
	if _, err := db.Bun().
		NewInsert().
		Model(&model.TrialTaskID{TrialID: t.id, TaskID: t.taskID}).
		Exec(context.TODO()); err != nil {
		return fmt.Errorf("adding trial ID task ID relationship: %w", err)
	}
	return nil
}

// maybeAllocateTask checks if the trial should allocate state and allocates it if so.
func (t *trial) maybeAllocateTask() error {
	if !(t.allocationID == nil &&
		!t.searcher.Complete &&
		t.state == model.ActiveState) {
		t.syslog.WithFields(logrus.Fields{
			"allocation-id":    t.allocationID,
			"sercher-complete": t.searcher.Complete,
			"trial-state":      t.state,
		}).Trace("decided not to allocate trial")
		return nil
	}

	name := fmt.Sprintf("Trial %d (Experiment %d)", t.id, t.experimentID)
	t.syslog.Info("decided to allocate trial")

	restoredAllocation, err := t.maybeRestoreAllocation()
	if err != nil {
		t.syslog.WithError(err).Warn("failed to restore trial allocation")
	} else if restoredAllocation != nil {
		specifier, err := t.buildTaskSpecifier()
		if err != nil {
			return err
		}

		ar := sproto.AllocateRequest{
			AllocationID:      restoredAllocation.AllocationID,
			TaskID:            t.taskID,
			JobID:             t.jobID,
			JobSubmissionTime: t.jobSubmissionTime,
			RequestTime:       time.Now().UTC(),
			IsUserVisible:     true,
			Name:              name,
			Group:             t.parent,
			SlotsNeeded:       t.config.Resources().SlotsPerTrial(),
			ResourcePool:      t.config.Resources().ResourcePool(),
			FittingRequirements: sproto.FittingRequirements{
				SingleAgent: false,
			},

			Preemptible: true,
			Restore:     true,
			ProxyPorts: sproto.NewProxyPortConfig(
				tasks.TrialSpecProxyPorts(t.taskSpec, t.config), t.taskID),
		}
		t.syslog.
			WithField("allocation-id", ar.AllocationID).
			Infof("starting restored trial allocation")
		err = task.DefaultService.StartAllocation(
			t.logCtx, ar, t.db, t.rm, specifier, t.system,
			t.AllocationExitedCallback,
		)
		if err != nil {
			return err
		}
		t.allocationID = &ar.AllocationID
		return nil
	}

	t.runID++
	t.logCtx = logger.MergeContexts(t.logCtx, logger.Context{"trial-run-id": t.runID})
	t.syslog = t.syslog.WithFields(t.logCtx.Fields())

	specifier, err := t.buildTaskSpecifier()
	if err != nil {
		return err
	}

	ar := sproto.AllocateRequest{
		AllocationID:      model.AllocationID(fmt.Sprintf("%s.%d", t.taskID, t.runID)),
		TaskID:            t.taskID,
		JobID:             t.jobID,
		RequestTime:       time.Now().UTC(),
		JobSubmissionTime: t.jobSubmissionTime,
		IsUserVisible:     true,
		Name:              name,
		Group:             t.parent,

		SlotsNeeded:  t.config.Resources().SlotsPerTrial(),
		ResourcePool: t.config.Resources().ResourcePool(),
		FittingRequirements: sproto.FittingRequirements{
			SingleAgent: false,
		},

		Preemptible: true,
		ProxyPorts:  sproto.NewProxyPortConfig(tasks.TrialSpecProxyPorts(t.taskSpec, t.config), t.taskID),
	}

	t.syslog.
		WithField("allocation-id", ar.AllocationID).
		Debugf("starting new trial allocation")

	prom.AssociateJobExperiment(t.jobID, strconv.Itoa(t.experimentID), t.config.Labels())
	err = task.DefaultService.StartAllocation(
		t.logCtx, ar, t.db, t.rm, specifier, t.system,
		t.AllocationExitedCallback,
	)
	if err != nil {
		return err
	}
	t.allocationID = &ar.AllocationID
	return nil
}

func (t *trial) addTask() error {
	return t.db.AddTask(&model.Task{
		TaskID:     t.taskID,
		TaskType:   model.TaskTypeTrial,
		StartTime:  t.jobSubmissionTime, // TODO: Why is this the job submission time..?
		JobID:      &t.jobID,
		LogVersion: model.CurrentTaskLogVersion,
	})
}

func (t *trial) buildTaskSpecifier() (*tasks.TrialSpec, error) {
	if err := t.db.UpdateTrialRunID(t.id, t.runID); err != nil {
		return nil, errors.Wrap(err, "failed to save trial run ID")
	}

	var stepsCompleted int
	latestCheckpoint, err := t.db.LatestCheckpointForTrial(t.id)
	switch {
	case err != nil:
		return nil, errors.Wrapf(err, "failed to query latest checkpoint for trial")
	case latestCheckpoint == nil:
		latestCheckpoint = t.warmStartCheckpoint
	default:
		stepsCompleted = latestCheckpoint.StepsCompleted
	}

	return &tasks.TrialSpec{
		Base: *t.taskSpec,

		ExperimentID:     t.experimentID,
		TrialID:          t.id,
		TrialRunID:       t.runID,
		ExperimentConfig: schemas.Copy(t.config),
		HParams:          t.searcher.Create.Hparams,
		TrialSeed:        t.searcher.Create.TrialSeed,
		StepsCompleted:   stepsCompleted,
		LatestCheckpoint: latestCheckpoint,

		Keys: t.generatedKeys,
	}, nil
}

// AllocationExitedCallback cleans up after an allocation exit and exits permanently or reallocates.
func (t *trial) AllocationExitedCallback(exit *task.AllocationExited) {
	t.mu.Lock()
	defer t.mu.Unlock()

	err := t.handleAllocationExit(exit)
	if err != nil {
		t.syslog.WithError(err).Error("handling allocation exit")
	}
}

func (t *trial) handleAllocationExit(exit *task.AllocationExited) error {
	if exit.Err != nil {
		t.syslog.WithError(exit.Err).Error("trial allocation failed")
	}
	t.allocationID = nil

	prom.DisassociateJobExperiment(t.jobID, strconv.Itoa(t.experimentID), t.config.Labels())

	// Decide if this is permanent.
	switch {
	case model.StoppingStates[t.state]:
		if exit.Err != nil {
			return t.transition(model.StateWithReason{
				State: model.ErrorState,
				InformationalReason: fmt.Sprintf(
					"trial allocation exited with an error while trial was stopping %v", exit.Err),
			})
		}
		return t.transition(model.StateWithReason{
			State:               model.StoppingToTerminalStates[t.state],
			InformationalReason: "trial stopped",
		})
	case t.searcher.Complete && t.searcher.Closed:
		if exit.Err != nil {
			return t.transition(model.StateWithReason{
				State: model.ErrorState,
				InformationalReason: fmt.Sprintf(
					"trial allocation exited with an error but hp search was complete %v", exit.Err),
			})
		}
		return t.transition(model.StateWithReason{
			State:               model.CompletedState,
			InformationalReason: "hp search is finished",
		})
	case exit.Err != nil && sproto.IsUnrecoverableSystemError(exit.Err):
		t.syslog.
			WithError(exit.Err).
			Errorf("trial encountered unrecoverable failure")
		return t.transition(model.StateWithReason{
			State: model.ErrorState,
			InformationalReason: fmt.Sprintf(
				"trial allocation exited with unrecoverable failure %v", exit.Err),
		})
	case exit.Err != nil && isNonRetryableError(exit.Err):
		// These are errors that no matter how many times we retry, the outcome will
		// be the same, so don't bother retrying. Fail right away to allow the user
		// to make any adjustments to the experiment and try again.
		return t.transition(model.StateWithReason{
			State: model.ErrorState,
			InformationalReason: fmt.Sprintf(
				"trial allocation exited with unrecoverable failure %v", exit.Err),
		})
	case exit.Err != nil && sproto.IsTransientSystemError(exit.Err):
		t.syslog.
			WithError(exit.Err).
			Errorf("trial encountered transient system error")
	case exit.Err != nil && !sproto.IsTransientSystemError(exit.Err):
		t.syslog.
			WithError(exit.Err).
			Errorf("trial failed (restart %d/%d)", t.restarts, t.config.MaxRestarts())
		t.restarts++
		if err := t.db.UpdateTrialRestarts(t.id, t.restarts); err != nil {
			return err
		}
		if t.restarts > t.config.MaxRestarts() {
			return t.transition(model.StateWithReason{
				State:               model.ErrorState,
				InformationalReason: "trial exceeded max restarts",
			})
		}
	case exit.UserRequestedStop:
		return t.transition(model.StateWithReason{
			State:               model.CompletedState,
			InformationalReason: "trial exited early due to a user requested stop",
		})
	case t.userInitiatedExit != nil:
		return t.transition(model.StateWithReason{
			State: model.CompletedState,
			InformationalReason: fmt.Sprintf(
				"trial exited early with reason: %v", *t.userInitiatedExit),
		})
	}

	// Maybe reschedule.
	return errors.Wrap(t.maybeAllocateTask(), "failed to reschedule trial")
}

// patchState decide if the state patch is valid. If so, we'll transition the trial.
func (t *trial) patchState(s model.StateWithReason) error {
	switch {
	case model.TerminalStates[t.state]:
		t.syslog.Infof("ignoring transition in terminal state (%s -> %s)", t.state, s.State)
		return nil
	case model.TerminalStates[s.State]:
		t.syslog.Infof("ignoring patch to terminal state %s", s.State)
		return nil
	case t.state == s.State: // Order is important, else below will prevent re-sending kills.
		t.syslog.Infof("resending actions for transition for %s", t.state)
		return t.transition(s)
	case model.StoppingStates[t.state] && !model.TrialTransitions[t.state][s.State]:
		t.syslog.Infof("ignoring patch to less severe stopping state (%s)", s.State)
		return nil
	default:
		t.syslog.Debugf("patching state after request (%s)", s.State)
		return t.transition(s)
	}
}

// transition the trial by rectifying the desired state with our actual state to determined
// a target state, and then propogating the appropriate signals to the allocation if there is any.
func (t *trial) transition(s model.StateWithReason) error {
	if t.state != s.State {
		t.syslog.Infof("trial changed from state %s to %s", t.state, s.State)
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
		return t.maybeAllocateTask()
	case t.state == model.PausedState:
		if t.allocationID != nil {
			t.syslog.Info("decided to terminate trial due to pause")
			err := task.DefaultService.Signal(
				*t.allocationID,
				task.TerminateAllocation,
				s.InformationalReason,
			)
			if err != nil {
				t.syslog.WithError(err).Warn("could not terminate allocation after pause")
			}
		}
	case model.StoppingStates[t.state]:
		switch {
		case t.allocationID == nil:
			t.syslog.Info("stopping trial before resources are requested")
			return t.transition(model.StateWithReason{
				State:               model.StoppingToTerminalStates[t.state],
				InformationalReason: s.InformationalReason,
			})
		default:
			if action, ok := map[model.State]task.AllocationSignal{
				model.StoppingCanceledState: task.TerminateAllocation,
				model.StoppingKilledState:   task.KillAllocation,
				model.StoppingErrorState:    task.KillAllocation,
			}[t.state]; ok {
				t.syslog.Infof("decided to %s trial", action)
				err := task.DefaultService.Signal(*t.allocationID, action, s.InformationalReason)
				if err != nil {
					t.syslog.WithError(err).Warnf("could not %s allocation during stop", action)
				}
			}
		}
	case model.TerminalStates[t.state]:
		switch t.state {
		case model.ErrorState:
			t.exit(ptrs.Ptr(model.Errored))
		case model.CanceledState:
			t.exit(ptrs.Ptr(model.UserCanceled))
		default:
			t.exit(nil)
		}
	default:
		panic(fmt.Errorf("unmatched state in transition %s", t.state))
	}
	return nil
}

func (t *trial) maybeRestoreAllocation() (*model.Allocation, error) {
	if !t.restored {
		return nil, nil
	}

	var allocations []model.Allocation
	selectQuery := db.Bun().NewSelect().Model(&allocations).
		Where("task_id = ?", t.taskID).
		Where("end_time IS NULL").
		Where("state != ?", model.AllocationStateTerminated)

	if t.rm.IsReattachableOnlyAfterStarted(t.system) {
		selectQuery.Where("start_time IS NOT NULL")
	}

	// Do we have an open allocation?
	err := selectQuery.Scan(context.TODO())
	if err != nil {
		return nil, err
	}

	openAllocs := len(allocations)
	switch {
	case openAllocs == 0:
		return nil, nil
	case openAllocs == 1:
		allocation := &allocations[0]
		return allocation, nil
	case openAllocs > 1:
		const maxAllocsToLog int = 3
		allocIDs := make([]string, 0, maxAllocsToLog)
		for _, alloc := range allocations[0:mathx.Min(len(allocations), maxAllocsToLog)] {
			allocIDs = append(allocIDs, alloc.AllocationID.String())
		}
		return nil, fmt.Errorf(
			"discovered %d open allocations on restore: %s",
			len(allocations),
			strings.Join(allocIDs, " "),
		)
	default:
		return nil, fmt.Errorf(
			"discovered %d open allocations on restore",
			len(allocations),
		)
	}
}
