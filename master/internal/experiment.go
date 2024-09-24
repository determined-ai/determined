package internal

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"sync"

	"github.com/google/uuid"
	"github.com/pkg/errors"
	"github.com/shopspring/decimal"
	"github.com/sirupsen/logrus"
	"github.com/uptrace/bun"
	"golang.org/x/sync/errgroup"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"

	"github.com/determined-ai/determined/master/internal/api"
	"github.com/determined-ai/determined/master/internal/checkpoints"
	"github.com/determined-ai/determined/master/internal/config"
	internaldb "github.com/determined-ai/determined/master/internal/db"
	"github.com/determined-ai/determined/master/internal/experiment"
	"github.com/determined-ai/determined/master/internal/job/jobservice"
	"github.com/determined-ai/determined/master/internal/rm"
	"github.com/determined-ai/determined/master/internal/rm/rmerrors"
	"github.com/determined-ai/determined/master/internal/rm/tasklist"
	"github.com/determined-ai/determined/master/internal/sproto"
	"github.com/determined-ai/determined/master/internal/telemetry"
	"github.com/determined-ai/determined/master/internal/user"
	"github.com/determined-ai/determined/master/internal/webhooks"
	"github.com/determined-ai/determined/master/internal/workspace"
	"github.com/determined-ai/determined/master/pkg/command"
	"github.com/determined-ai/determined/master/pkg/logger"
	"github.com/determined-ai/determined/master/pkg/model"
	"github.com/determined-ai/determined/master/pkg/ptrs"
	"github.com/determined-ai/determined/master/pkg/schemas"
	"github.com/determined-ai/determined/master/pkg/schemas/expconf"
	"github.com/determined-ai/determined/master/pkg/searcher"
	"github.com/determined-ai/determined/master/pkg/ssh"
	"github.com/determined-ai/determined/master/pkg/tasks"
)

const (
	maxConcurrentTrialOps = 16
)

type (
	experimentState struct {
		SearcherState    json.RawMessage                       `json:"searcher_state"`
		RunSearcherState map[int32]experiment.RunSearcherState `json:"run_searcher_state"`
	}

	//legacyExperimentState struct {
	//	SearcherState      json.RawMessage                                   `json:"searcher_state"`
	//	TrialSearcherState map[model.RequestID]experiment.TrialSearcherState `json:"trial_searcher_state"`
	//}

	internalExperiment struct {
		mu sync.Mutex

		experimentState

		trials map[int32]*trial

		*model.Experiment
		activeConfig        expconf.ExperimentConfig
		db                  *internaldb.PgDB
		rm                  rm.ResourceManager
		syslog              *logrus.Entry
		searcher            *searcher.Searcher
		warmStartCheckpoint *model.Checkpoint
		continueTrials      bool

		taskSpec      *tasks.TaskSpec
		generatedKeys ssh.PrivateAndPublicKeys

		faultToleranceEnabled bool
		restored              bool

		logCtx logger.Context
	}
)

// returns the workspace set by the user or the default workspace if none.
func resolveWorkspaceID(workspace *model.Workspace) int {
	if workspace == nil || workspace.ID == 0 {
		return 1
	}
	return workspace.ID
}

// Create a new experiment object from the given model experiment object, along with its searcher
// and log. If the input object has no ID set, also create a new experiment in the database and set
// the returned object's ID appropriately.
func newExperiment(
	m *Master,
	expModel *model.Experiment,
	modelDef []byte,
	activeConfig expconf.ExperimentConfig,
	taskSpec *tasks.TaskSpec,
) (*internalExperiment, []command.LaunchWarning, error) {
	if len(modelDef) > 0 && expModel.ID != 0 {
		return nil, nil, fmt.Errorf("experiments restoring should not provide a model def")
	}

	resources := activeConfig.Resources()
	workspaceModel, err := workspace.WorkspaceByProjectID(context.TODO(), expModel.ProjectID)
	if err != nil && errors.Cause(err) != sql.ErrNoRows {
		return nil, nil, err
	}
	workspaceID := resolveWorkspaceID(workspaceModel)
	poolName, err := m.rm.ResolveResourcePool(
		rm.ResourcePoolName(resources.ResourcePool()), workspaceID, resources.SlotsPerTrial(),
	)
	if err != nil {
		return nil, nil, fmt.Errorf("cannot create an experiment: %w", err)
	}

	var launchWarnings []command.LaunchWarning
	if expModel.ID == 0 {
		if launchWarnings, err = m.rm.ValidateResources(sproto.ValidateResourcesRequest{
			ResourcePool: poolName.String(),
			Slots:        resources.SlotsPerTrial(),
			IsSingleNode: resources.IsSingleNode() != nil && *resources.IsSingleNode(),
		}); err != nil {
			return nil, nil, fmt.Errorf("validating resources: %v", err)
		}
		if m.config.LaunchError && len(launchWarnings) > 0 {
			return nil, nil, errors.New("slots requested exceeds cluster capacity")
		}
	}
	resources.SetResourcePool(poolName.String())

	activeConfig.SetResources(resources)

	method := searcher.NewSearchMethod(activeConfig.Searcher())
	search := searcher.NewSearcher(
		activeConfig.Reproducibility().ExperimentSeed(), method, activeConfig.Hyperparameters(),
	)

	// Retrieve the warm start checkpoint, if provided.
	checkpoint, err := checkpointFromTrialIDOrUUID(
		m.db, activeConfig.Searcher().SourceTrialID(), activeConfig.Searcher().SourceCheckpointUUID())
	if err != nil {
		return nil, launchWarnings, err
	}

	if expModel.ID == 0 {
		if err = m.db.AddExperiment(expModel, modelDef, activeConfig); err != nil {
			return nil, launchWarnings, err
		}
		telemetry.ReportExperimentCreated(expModel.ID, activeConfig)
	}

	agentUserGroup, err := user.GetAgentUserGroup(context.TODO(), *expModel.OwnerID, workspaceID)
	if err != nil {
		return nil, launchWarnings, err
	}

	taskSpec.AgentUserGroup = agentUserGroup

	generatedKeys, err := ssh.GenerateKey(taskSpec.SSHRsaSize, nil)
	if err != nil {
		return nil, nil, errors.Wrap(err, "generating ssh keys for trials")
	}

	return &internalExperiment{
		Experiment:   expModel,
		activeConfig: activeConfig,
		db:           m.db,
		rm:           m.rm,
		syslog: logrus.WithFields(logrus.Fields{
			"component":     "experiment",
			"job-id":        expModel.JobID,
			"experiment-id": expModel.ID,
		},
		),
		searcher:            search,
		warmStartCheckpoint: checkpoint,

		trials: map[int32]*trial{},

		taskSpec:      taskSpec,
		generatedKeys: generatedKeys,

		faultToleranceEnabled: true,

		experimentState: experimentState{
			RunSearcherState: map[int32]experiment.RunSearcherState{},
		},

		logCtx: logger.Context{
			"job-id":        expModel.JobID,
			"experiment-id": expModel.ID,
		},
	}, launchWarnings, nil
}

func newUnmanagedExperiment(
	ctx context.Context,
	idb bun.IDB,
	expModel *model.Experiment,
	modelDef []byte,
	activeConfig expconf.ExperimentConfig,
) (*internalExperiment, []command.LaunchWarning, error) {
	expModel.State = model.PausedState
	expModel.Unmanaged = true

	if err := internaldb.AddExperimentTx(ctx, idb, expModel, modelDef, activeConfig, true); err != nil {
		return nil, nil, err
	}
	telemetry.ReportExperimentCreated(expModel.ID, activeConfig)

	// Will only have the model, nothing required for the experiment actor.
	return &internalExperiment{
		Experiment: expModel,
	}, nil, nil
}

// Start first registers the experiment and then starts synchronously.
func (e *internalExperiment) Start() error {
	e.mu.Lock()
	defer e.mu.Unlock()

	if err := e.register(); err != nil {
		return err
	}
	if err := e.start(); err != nil {
		e.unregister()
		return err
	}
	return nil
}

func (e *internalExperiment) register() error {
	return experiment.ExperimentRegistry.Add(e.ID, e)
}

func (e *internalExperiment) unregister() {
	if err := experiment.ExperimentRegistry.Delete(e.ID); err != nil {
		e.syslog.WithError(err).Error("failed to unregister experiment")
	}
}

func (e *internalExperiment) start() error {
	priorityChange := func(priority int) error {
		e.mu.Lock()
		defer e.mu.Unlock()
		return e.setPriority(&priority, false)
	}
	if err := tasklist.GroupPriorityChangeRegistry.Add(e.JobID, priorityChange); err != nil {
		return err
	}

	e.rm.SetGroupMaxSlots(sproto.SetGroupMaxSlots{
		MaxSlots:     e.activeConfig.Resources().MaxSlots(),
		ResourcePool: e.activeConfig.Resources().ResourcePool(),
		JobID:        e.JobID,
	})
	if err := e.setWeight(e.activeConfig.Resources().Weight()); err != nil {
		e.updateState(model.StateWithReason{
			State:               model.StoppingErrorState,
			InformationalReason: err.Error(),
		})
		return err
	}
	if err := e.setPriority(e.activeConfig.Resources().Priority(), true); err != nil {
		e.updateState(model.StateWithReason{
			State:               model.StoppingErrorState,
			InformationalReason: err.Error(),
		})
		return err
	}

	jobservice.DefaultService.RegisterJob(e.JobID, e)

	if e.restored {
		j, err := internaldb.JobByID(context.TODO(), e.JobID)
		if err != nil {
			e.updateState(model.StateWithReason{
				State:               model.StoppingErrorState,
				InformationalReason: err.Error(),
			})
			return err
		}

		if j.QPos.GreaterThan(decimal.Zero) {
			e.rm.RecoverJobPosition(sproto.RecoverJobPosition{
				JobID:        e.JobID,
				JobPosition:  j.QPos,
				ResourcePool: e.activeConfig.Resources().ResourcePool(),
			})
		}

		e.restoreTrials()

		// Resend stopping state to trials again so we can reregister preemption timeout and stuff.
		if model.StoppingStates[e.State] && e.State != model.StoppingCompletedState {
			e.patchTrialsState(model.StateWithReason{
				State:               e.State,
				InformationalReason: "resending stopping state signal on restore",
			})
		}
		return nil
	}

	creates, err := e.searcher.InitialRuns()
	if err != nil {
		err = errors.Wrap(err, "failed to generate initial operations")
		e.updateState(model.StateWithReason{
			State:               model.StoppingErrorState,
			InformationalReason: err.Error(),
		})
		return err
	}
	e.handleSearcherActions(creates, nil)

	return nil
}

func (e *internalExperiment) RunReportProgress(runID int32, msg experiment.RunReportProgress) error {
	e.mu.Lock()
	defer e.mu.Unlock()

	progress := float64(msg.Progress)
	e.searcher.SetRunProgress(runID, progress)

	if err := e.db.SaveExperimentProgress(e.ID, &progress); err != nil {
		e.syslog.WithError(err).Error("failed to save experiment progress")
	}
	return nil
}

func (e *internalExperiment) RunReportValidation(runID int32, metrics map[string]interface{}) error {
	e.mu.Lock()
	defer e.mu.Unlock()
	ops, err := e.searcher.ValidationCompleted(runID, metrics)
	e.handleSearcherActions(ops, err)
	return nil
}

func (e *internalExperiment) UserInitiatedEarlyRunExit(msg experiment.UserInitiatedEarlyRunExit) error {
	e.mu.Lock()
	defer e.mu.Unlock()

	ref, ok := e.trials[msg.RunID]
	if !ok {
		return api.AsErrNotFound("trial not found")
	}
	if err := ref.SetUserInitiatedEarlyExit(msg); err != nil {
		return err
	}
	return nil
}

func (e *internalExperiment) PatchRunState(msg experiment.PatchRunState) error {
	e.mu.Lock()
	defer e.mu.Unlock()

	ref, ok := e.trials[msg.RunID]
	if !ok {
		return api.AsErrNotFound("trial not found")
	}
	if err := ref.PatchState(msg.State); err != nil {
		return err
	}
	return nil
}

func (e *internalExperiment) SetGroupMaxSlots(msg sproto.SetGroupMaxSlots) {
	e.mu.Lock()
	defer e.mu.Unlock()

	resources := e.activeConfig.Resources()
	resources.SetMaxSlots(msg.MaxSlots)
	e.activeConfig.SetResources(resources)
	msg.JobID = e.JobID
	msg.ResourcePool = e.activeConfig.Resources().ResourcePool()
	e.rm.SetGroupMaxSlots(msg)
}

func (e *internalExperiment) SetGroupWeight(weight float64) error {
	e.mu.Lock()
	defer e.mu.Unlock()

	return e.setWeight(weight)
}

func (e *internalExperiment) SetGroupPriority(priority int) error {
	e.mu.Lock()
	defer e.mu.Unlock()

	return e.setPriority(&priority, true)
}

func (e *internalExperiment) stop() error {
	e.unregister()

	if err := tasklist.GroupPriorityChangeRegistry.Delete(e.JobID); err != nil {
		e.syslog.WithError(err).Error("failed to remove priority change registry")
	}
	if e.State == model.CompletedState || e.State == model.StoppingCompletedState {
		if err := e.db.SaveExperimentProgress(e.ID, ptrs.Ptr(1.0)); err != nil {
			e.syslog.Error(err)
		}
	}
	go jobservice.DefaultService.UnregisterJob(e.JobID)
	state := model.StoppingToTerminalStates[e.State]
	if state == "" {
		state = model.ErrorState
	}
	if wasPatched, err := e.Transition(state); err != nil {
		return err
	} else if !wasPatched {
		return errors.New("experiment is already in a terminal state")
	}
	telemetry.ReportExperimentStateChanged(e.db, e.Experiment)
	if err := webhooks.ReportExperimentStateChanged(
		context.TODO(), *e.Experiment, e.activeConfig,
	); err != nil {
		e.syslog.WithError(err).Error("failed to send experiment state change webhook")
	}

	if err := e.db.SaveExperimentState(e.Experiment); err != nil {
		return err
	}
	e.syslog.Infof("PostStop state changed to %s", e.State)

	taskSpec, err := e.taskSpec.Clone()
	if err != nil {
		return fmt.Errorf("cloning checkpoint gc task spec: %w", err)
	}

	checkpoints, err := experiment.ExperimentCheckpointsToGCRaw(
		context.TODO(),
		e.Experiment.ID,
		e.activeConfig.CheckpointStorage().SaveExperimentBest(),
		e.activeConfig.CheckpointStorage().SaveTrialBest(),
		e.activeConfig.CheckpointStorage().SaveTrialLatest(),
	)
	if err != nil {
		e.syslog.WithError(err).Error("")
	}

	if err := e.db.DeleteSnapshotsForExperiment(e.Experiment.ID); err != nil {
		e.syslog.WithError(err).Errorf(
			"failure to delete snapshots for experiment: %d", e.Experiment.ID)
	}

	// May be no checkpoints to gc, if so skip
	if len(checkpoints) > 0 {
		go func() {
			if err := runCheckpointGCForCheckpoints(
				e.rm, e.db, e.JobID, e.StartTime, taskSpec,
				e.Experiment.ID, e.activeConfig.AsLegacy(), checkpoints,
				[]string{fullDeleteGlob},
				false, taskSpec.AgentUserGroup, taskSpec.Owner, e.logCtx,
			); err != nil {
				e.syslog.WithError(err).Error("failed to GC experiment checkpoints")
			}
		}()
	}

	if err := user.DeleteSessionByToken(
		context.TODO(),
		taskSpec.UserSessionToken,
	); err != nil {
		e.syslog.WithError(err).Errorf(
			"failure to delete user session for experiment: %d", e.Experiment.ID)
	}

	e.syslog.Info("experiment shut down successfully")
	return nil
}

func (e *internalExperiment) ActivateExperiment() error {
	e.mu.Lock()
	defer e.mu.Unlock()

	if ok := e.updateState(model.StateWithReason{
		State:               model.ActiveState,
		InformationalReason: "user requested activation",
	}); !ok {
		return status.Errorf(codes.FailedPrecondition,
			"experiment in incompatible state %s", e.State)
	}
	return nil
}

func (e *internalExperiment) PauseExperiment() error {
	e.mu.Lock()
	defer e.mu.Unlock()

	if ok := e.updateState(model.StateWithReason{
		State:               model.PausedState,
		InformationalReason: "user requested pause",
	}); !ok {
		return status.Errorf(codes.FailedPrecondition,
			"experiment in incompatible state %s", e.State)
	}
	return nil
}

func (e *internalExperiment) CancelExperiment() error {
	e.mu.Lock()
	defer e.mu.Unlock()

	if model.StoppingStates[e.State] || model.TerminalStates[e.State] {
		return nil
	}
	if ok := e.updateState(model.StateWithReason{
		State:               model.StoppingCanceledState,
		InformationalReason: "user requested cancellation",
	}); !ok {
		return status.Errorf(codes.FailedPrecondition,
			"experiment in incompatible state %s", e.State)
	}
	return nil
}

func (e *internalExperiment) KillExperiment() error {
	e.mu.Lock()
	defer e.mu.Unlock()

	if e.State == model.StoppingKilledState || model.TerminalStates[e.State] {
		return nil
	}
	if ok := e.updateState(model.StateWithReason{
		State:               model.StoppingKilledState,
		InformationalReason: "user requested kill",
	}); !ok {
		return status.Errorf(codes.FailedPrecondition,
			"experiment in incompatible state %s", e.State,
		)
	}
	return nil
}

func (e *internalExperiment) RunClosed(runID int32, reason *model.ExitedReason) {
	e.mu.Lock()
	defer e.mu.Unlock()

	e.runClosed(runID, reason)
}

func (e *internalExperiment) runClosed(runID int32, reason *model.ExitedReason) {
	if reason != nil {
		e.trialReportEarlyExit(runID, *reason)
	}
	delete(e.trials, runID)

	ops, err := e.searcher.RunClosed(runID)
	e.handleSearcherActions(ops, err)
	if e.canTerminate() {
		if err := e.stop(); err != nil {
			e.syslog.WithError(err).Error("failed to stop experiment on trial closed")
		}
	}
}

func (e *internalExperiment) trialReportEarlyExit(runID int32, reason model.ExitedReason) {
	e.syslog.WithField("requestId", runID).Info("experiment received trial early exit")
	state, ok := e.RunSearcherState[runID]
	if !ok {
		e.syslog.WithField("runID", runID).Error("trial has no searcher state on early exit")
		return
	}

	defer func() {
		ops, err := e.searcher.RunExitedEarly(runID, reason)
		e.handleSearcherActions(ops, err)
	}()

	state.Closed = true
	e.RunSearcherState[runID] = state

	t, ok := e.trials[runID]
	if !ok {
		e.syslog.WithField("runID", runID).Warnf("missing trial to patch on early exit")
		return
	}

	err := t.PatchSearcherState(state)
	if err != nil {
		e.syslog.WithError(err).Error("patching trial search state")
	}
}

func (e *internalExperiment) trialCreated(t *trial) {
	runID := int32(t.id)
	if !e.searcher.RunIsCreated(runID) {
		ops, err := e.searcher.RunCreated(runID, t.searcher.Create)
		e.handleSearcherActions(ops, err)
	}
	state, ok := e.RunSearcherState[runID]
	if !ok {
		e.syslog.WithField("runID", t.id).Error("run has no searcher state on create")
		return
	}
	state.RunID = ptrs.Ptr(runID)
	e.RunSearcherState[runID] = state
	e.trials[runID] = t
}

// restoreTrialsFromStates from the operations that were snapshotted with the
// last experiment checkpoint.
func (e *internalExperiment) restoreTrials() {
	for _, state := range e.RunSearcherState {
		e.restoreRun(e.warmStartCheckpoint, state)
	}
}

func (e *internalExperiment) handleSearcherActions(
	actions []searcher.Action, err error,
) {
	// Only continue for experiments in stopping states if the searcher operations are all
	// type Shutdown failures.
	if _, ok := model.StoppingStates[e.State]; ok && !allSearcherShutdowns(actions) {
		return
	}

	if err != nil {
		e.syslog.Error(err)
		e.updateState(model.StateWithReason{
			State:               model.StoppingErrorState,
			InformationalReason: fmt.Sprintf("encountered error %v", err),
		})
		return
	}

	defer e.snapshotAndSave()

	updatedTrials := make(map[int32]bool)
	for _, action := range actions {
		e.syslog.Debugf("handling searcher action: %v", action)
		switch action := action.(type) {
		case searcher.Create:
			config := schemas.Copy(e.activeConfig)
			state := experiment.RunSearcherState{Create: action}

			clonedSpec, err := e.taskSpec.Clone()
			if err != nil {
				e.syslog.WithError(err).Error("failed to create trial")
				continue
			}

			t, err := newTrial(
				e.logCtx, trialTaskID(e.ID), e.JobID, e.StartTime, e.ID, e.State,
				state, e.rm, e.db, config, e.warmStartCheckpoint, clonedSpec, e.generatedKeys, false,
				nil, nil, e.RunClosed,
			)
			if err != nil {
				e.syslog.WithError(err).Error("failed to create trial")
				continue
			}
			e.trialCreated(t)
		case searcher.Stop:
			state := e.RunSearcherState[action.RunID]
			state.Stopped = true
			e.RunSearcherState[action.RunID] = state
			updatedTrials[action.RunID] = true
		case searcher.Shutdown:
			e.syslog.WithField("action", action).Info("searcher shutdown")
			switch {
			case action.Failure:
				e.updateState(model.StateWithReason{
					State:               model.StoppingErrorState,
					InformationalReason: "hp search failed",
				})
			case action.Cancel:
				e.updateState(model.StateWithReason{
					State:               model.StoppingCanceledState,
					InformationalReason: "hp search canceled",
				})
			default:
				e.updateState(model.StateWithReason{
					State:               model.StoppingCompletedState,
					InformationalReason: "hp search completed",
				})
			}
		default:
			panic(fmt.Sprintf("unexpected action: %v", action))
		}
	}

	var g errgroup.Group
	g.SetLimit(maxConcurrentTrialOps)
	for runID := range updatedTrials {
		syslog := e.syslog.WithField("runID", runID)
		t, ok := e.trials[runID]
		if !ok {
			syslog.Errorf("handleSearcherActions invalid runID")
			continue
		}
		g.Go(func() error {
			err := t.PatchSearcherState(e.RunSearcherState[runID])
			if err != nil {
				syslog.WithError(err).Error("handleSearcherActions updating trial search state")
			}
			return nil
		})
	}
	_ = g.Wait() // Errors are handled in g.Go.
}

func trialTaskID(eID int) model.TaskID {
	return model.TaskID(fmt.Sprintf("%d.%s", eID, model.NewTaskID()))
}

var errIsNotTrialTaskID = fmt.Errorf("taskID is not a trial task ID")

func experimentIDFromTrialTaskID(taskID model.TaskID) (int, error) {
	var experimentID int
	err := internaldb.Bun().NewSelect().
		Table("run_id_task_id").
		Column("experiment_id").
		Join("LEFT JOIN trials ON trials.id = run_id_task_id.run_id").
		Where("task_id = ?", taskID).
		Scan(context.TODO(), &experimentID)
	if errors.Is(err, sql.ErrNoRows) {
		return 0, errIsNotTrialTaskID
	} else if err != nil {
		return 0, fmt.Errorf("getting experiment ID from trial task ID: %w", err)
	}

	return experimentID, nil
}

func (e *internalExperiment) updateState(state model.StateWithReason) bool {
	if wasPatched, err := e.Transition(state.State); err != nil {
		e.syslog.Errorf("error transitioning experiment state: %s", err)
		return false
	} else if !wasPatched {
		return true
	}
	telemetry.ReportExperimentStateChanged(e.db, e.Experiment)
	if err := webhooks.ReportExperimentStateChanged(
		context.TODO(), *e.Experiment, e.activeConfig,
	); err != nil {
		e.syslog.WithError(err).Error("failed to send experiment state change webhook")
	}

	e.syslog.Infof("updateState changed to %s", state.State)
	e.patchTrialsState(state)

	// The database error is explicitly ignored.
	if err := e.db.SaveExperimentState(e.Experiment); err != nil {
		e.syslog.Errorf("error saving experiment state: %s", err)
	}
	if e.canTerminate() {
		if err := e.stop(); err != nil {
			e.syslog.WithError(err).Error("failed to stop experiment on updateState")
		}
	}

	return true
}

func (e *internalExperiment) patchTrialsState(state model.StateWithReason) {
	var g errgroup.Group
	g.SetLimit(maxConcurrentTrialOps)
	for _, t := range e.trials {
		g.Go(func() error {
			err := t.PatchState(state)
			if err != nil {
				e.syslog.WithError(err).Error("patching trial state")
			}
			return nil
		})
	}
	_ = g.Wait() // Errors are handled in g.Go.
}

func (e *internalExperiment) canTerminate() bool {
	return model.StoppingStates[e.State] && len(e.trials) == 0
}

func (e *internalExperiment) snapshot() (json.RawMessage, error) {
	searcherSnapshot, err := e.searcher.Snapshot()
	if err != nil {
		return nil, errors.Wrap(err, "failed to snapshot searcher")
	}
	e.SearcherState = searcherSnapshot
	experimentSnapshot, err := json.Marshal(e.experimentState)
	return experimentSnapshot, errors.Wrap(err, "failed to marshal experiment")
}

func (e *internalExperiment) restore(experimentSnapshot json.RawMessage) error {
	if err := json.Unmarshal(experimentSnapshot, &e.experimentState); err != nil {
		return errors.Wrap(err, "failed to unmarshal experiment snapshot")
	}
	if err := e.searcher.Restore(e.SearcherState); err != nil {
		return errors.Wrap(err, "failed to restore searcher snapshot")
	}
	return nil
}

func checkpointFromTrialIDOrUUID(
	db *internaldb.PgDB, trialID *int, checkpointUUIDStr *string,
) (*model.Checkpoint, error) {
	var checkpoint *model.Checkpoint
	var err error

	// Attempt to find a Checkpoint object from the given IDs.
	if trialID != nil {
		checkpoint, err = db.LatestCheckpointForTrial(*trialID)
		if err != nil {
			return nil, errors.Wrapf(err, "failed to get checkpoint for source trial %d", *trialID)
		}
		if checkpoint == nil {
			return nil, errors.Errorf("no checkpoint found for source trial %d", *trialID)
		}
	} else if checkpointUUIDStr != nil {
		checkpointUUID, err := uuid.Parse(*checkpointUUIDStr)
		if err != nil {
			return nil, errors.Wrap(err, "invalid source checkpoint UUID")
		}
		checkpoint, err = checkpoints.CheckpointByUUID(context.TODO(), checkpointUUID)
		if err != nil {
			return nil, errors.Wrapf(err, "failed to get source checkpoint %v", checkpointUUID)
		}
		if checkpoint == nil {
			return nil, errors.Errorf("no checkpoint found with UUID %v", checkpointUUID)
		}
	}
	return checkpoint, nil
}

func (e *internalExperiment) setPriority(priority *int, forward bool) (err error) {
	if priority == nil {
		return nil
	}
	oldPriority := config.DefaultSchedulingPriority
	var oldPriorityPtr *int
	resources := e.activeConfig.Resources()
	if resources.Priority() != nil {
		oldPriority = *resources.Priority()
		oldPriorityPtr = &oldPriority
	}
	resources.SetPriority(priority)
	e.activeConfig.SetResources(resources)

	defer func() {
		if err != nil {
			resources.SetPriority(oldPriorityPtr)
			e.activeConfig.SetResources(resources)
			err = e.db.SaveExperimentConfig(e.ID, e.activeConfig)
			if err != nil {
				return
			}
		}
	}()

	if err := e.db.SaveExperimentConfig(e.ID, e.activeConfig); err != nil {
		return errors.Wrapf(err, "setting experiment %d priority", e.ID)
	}

	if forward {
		switch err := e.rm.SetGroupPriority(sproto.SetGroupPriority{
			Priority:     *priority,
			ResourcePool: e.activeConfig.Resources().ResourcePool(),
			JobID:        e.JobID,
		}).(type) {
		case nil:
		case rmerrors.UnsupportedError:
			e.syslog.WithError(err).Debug("ignoring unsupported call to set group priority")
		default:
			return errors.Wrapf(err, "setting experiment %d priority", e.ID)
		}
	}

	return nil
}

func (e *internalExperiment) setWeight(weight float64) error {
	resources := e.activeConfig.Resources()
	oldWeight := resources.Weight()
	resources.SetWeight(weight)
	e.activeConfig.SetResources(resources)
	if err := e.db.SaveExperimentConfig(e.ID, e.activeConfig); err != nil {
		resources.SetWeight(oldWeight)
		e.activeConfig.SetResources(resources)
		return fmt.Errorf("setting experiment %d weight: %w", e.ID, err)
	}

	switch err := e.rm.SetGroupWeight(sproto.SetGroupWeight{
		Weight:       weight,
		ResourcePool: e.activeConfig.Resources().ResourcePool(),
		JobID:        e.JobID,
	}).(type) {
	case nil:
	case rmerrors.UnsupportedError:
		e.syslog.WithError(err).Debug("ignoring unsupported call to set group weight")
	default:
		resources.SetWeight(oldWeight)
		e.activeConfig.SetResources(resources)
		return fmt.Errorf("setting experiment %d weight: %w", e.ID, err)
	}
	return nil
}

func (e *internalExperiment) setRP(resourcePool string) error {
	resources := e.activeConfig.Resources()
	oldRP := resources.ResourcePool()
	workspaceModel, err := workspace.WorkspaceByProjectID(context.TODO(), e.ProjectID)
	if err != nil && errors.Cause(err) != sql.ErrNoRows {
		return err
	}
	workspaceID := resolveWorkspaceID(workspaceModel)
	rp, err := e.rm.ResolveResourcePool(
		rm.ResourcePoolName(resourcePool), workspaceID, e.activeConfig.Resources().SlotsPerTrial(),
	)
	switch {
	case err != nil:
		return fmt.Errorf("invalid resource pool name %s", resourcePool)
	case oldRP == rp.String():
		return fmt.Errorf("resource pool is unchanged (%s == %s)", oldRP, rp)
	}

	resources.SetResourcePool(rp.String())
	e.activeConfig.SetResources(resources)

	if err := e.db.SaveExperimentConfig(e.ID, e.activeConfig); err != nil {
		resources.SetResourcePool(oldRP)
		e.activeConfig.SetResources(resources)
		return errors.Wrapf(err, "setting experiment %d RP to %s", e.ID, rp)
	}

	var g errgroup.Group
	g.SetLimit(maxConcurrentTrialOps)
	for _, t := range e.trials {
		g.Go(func() error {
			t.PatchRP(rp.String())
			return nil
		})
	}
	_ = g.Wait() // Errors handled in g.Go.

	return nil
}

func allSearcherShutdowns(actions []searcher.Action) bool {
	for _, action := range actions {
		if _, ok := action.(searcher.Shutdown); !ok {
			return false
		}
	}
	return true
}
