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
	"github.com/determined-ai/determined/master/internal/config"
	"github.com/determined-ai/determined/master/internal/db"
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
	"github.com/determined-ai/determined/proto/pkg/apiv1"
	"github.com/determined-ai/determined/proto/pkg/experimentv1"
)

const (
	maxConcurrentTrialOps = 16
)

type (
	experimentState struct {
		SearcherState      json.RawMessage                                   `json:"searcher_state"`
		TrialSearcherState map[model.RequestID]experiment.TrialSearcherState `json:"trial_searcher_state"`
	}

	internalExperiment struct {
		mu sync.Mutex

		experimentState

		trials map[model.RequestID]*trial

		*model.Experiment
		activeConfig        expconf.ExperimentConfig
		db                  *db.PgDB
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
	activeConfig expconf.ExperimentConfig,
	taskSpec *tasks.TaskSpec,
) (*internalExperiment, []command.LaunchWarning, error) {
	resources := activeConfig.Resources()
	workspaceModel, err := workspace.WorkspaceByProjectID(context.TODO(), expModel.ProjectID)
	if err != nil && errors.Cause(err) != sql.ErrNoRows {
		return nil, nil, err
	}
	workspaceID := resolveWorkspaceID(workspaceModel)
	poolName, err := m.rm.ResolveResourcePool(
		resources.ResourcePool(), workspaceID, resources.SlotsPerTrial(),
	)
	if err != nil {
		return nil, nil, fmt.Errorf("cannot create an experiment: %w", err)
	}

	var launchWarnings []command.LaunchWarning
	if expModel.ID == 0 {
		err := m.rm.ValidateResources(poolName, resources.SlotsPerTrial(), false)
		if err != nil {
			return nil, nil, fmt.Errorf("validating resources: %v", err)
		}
		launchWarnings, err = m.rm.ValidateResourcePoolAvailability(
			&sproto.ValidateResourcePoolAvailabilityRequest{
				Name:  poolName,
				Slots: resources.SlotsPerTrial(),
			},
		)
		if err != nil {
			return nil, launchWarnings, fmt.Errorf("getting resource availability: %w", err)
		}
		if m.config.ResourceManager.AgentRM != nil && m.config.LaunchError && len(launchWarnings) > 0 {
			return nil, nil, errors.New("slots requested exceeds cluster capacity")
		}
	}
	resources.SetResourcePool(poolName)

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
		if err = m.db.AddExperiment(expModel, activeConfig); err != nil {
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

		trials: map[model.RequestID]*trial{},

		taskSpec:      taskSpec,
		generatedKeys: generatedKeys,

		faultToleranceEnabled: true,

		experimentState: experimentState{
			TrialSearcherState: map[model.RequestID]experiment.TrialSearcherState{},
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
	m *Master,
	expModel *model.Experiment,
	activeConfig expconf.ExperimentConfig,
	taskSpec *tasks.TaskSpec,
) (*internalExperiment, []command.LaunchWarning, error) {
	expModel.State = model.PausedState
	expModel.Unmanaged = true

	if err := db.AddExperimentTx(ctx, idb, expModel, activeConfig, true); err != nil {
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
		j, err := e.db.JobByID(e.JobID)
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
		return nil
	}

	ops, err := e.searcher.InitialOperations()
	if err != nil {
		err = errors.Wrap(err, "failed to generate initial operations")
		e.updateState(model.StateWithReason{
			State:               model.StoppingErrorState,
			InformationalReason: err.Error(),
		})
		return err
	}
	e.processOperations(ops, nil)

	return nil
}

func (e *internalExperiment) TrialCompleteOperation(msg experiment.TrialCompleteOperation) error {
	e.mu.Lock()
	defer e.mu.Unlock()

	state, ok := e.TrialSearcherState[msg.Op.RequestID]
	switch {
	case !ok:
		return api.AsValidationError("no such trial")
	case msg.Op != state.Op:
		return api.AsValidationError("expected op %v but received op %v", state.Op, msg.Op)
	case state.Complete:
		return api.AsValidationError("received op %v which was previously completed", msg.Op)
	}

	defer func() {
		ops, err := e.searcher.ValidationCompleted(msg.RequestID, msg.Metric, msg.Op)
		e.processOperations(ops, err)
	}()

	state.Complete = true
	e.TrialSearcherState[msg.Op.RequestID] = state

	t, ok := e.trials[msg.Op.RequestID]
	if !ok {
		return api.AsErrNotFound("trial not found")
	}

	err := t.PatchSearcherState(state)
	if err != nil {
		e.syslog.WithError(err).Error("patching trial search state")
		return err
	}

	return nil
}

func (e *internalExperiment) TrialReportProgress(msg experiment.TrialReportProgress) error {
	e.mu.Lock()
	defer e.mu.Unlock()

	e.searcher.SetTrialProgress(msg.RequestID, msg.Progress)
	progress := e.searcher.Progress()
	if err := e.db.SaveExperimentProgress(e.ID, &progress); err != nil {
		e.syslog.WithError(err).Error("failed to save experiment progress")
	}
	return nil
}

func (e *internalExperiment) TrialGetSearcherState(requestID model.RequestID) (experiment.TrialSearcherState, error) {
	e.mu.Lock()
	defer e.mu.Unlock()

	state, ok := e.TrialSearcherState[requestID]
	if !ok {
		return state, api.AsErrNotFound("trial has no state")
	}
	return state, nil
}

func (e *internalExperiment) UserInitiatedEarlyTrialExit(msg experiment.UserInitiatedEarlyTrialExit) error {
	e.mu.Lock()
	defer e.mu.Unlock()

	ref, ok := e.trials[msg.RequestID]
	if !ok {
		return api.AsErrNotFound("trial not found")
	}
	if err := ref.SetUserInitiatedEarlyExit(msg); err != nil {
		return err
	}
	return nil
}

func (e *internalExperiment) PatchTrialState(msg experiment.PatchTrialState) error {
	e.mu.Lock()
	defer e.mu.Unlock()

	ref, ok := e.trials[msg.RequestID]
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

	checkpoints, err := e.db.ExperimentCheckpointsToGCRaw(
		e.Experiment.ID,
		e.activeConfig.CheckpointStorage().SaveExperimentBest(),
		e.activeConfig.CheckpointStorage().SaveTrialBest(),
		e.activeConfig.CheckpointStorage().SaveTrialLatest(),
	)
	if err != nil {
		e.syslog.WithError(err).Error("")
	}

	taskSpec := *e.taskSpec

	// May be no checkpoints to gc, if so skip
	if len(checkpoints) > 0 {
		taskID := model.TaskID(fmt.Sprintf("%d.%s", e.ID, uuid.New()))
		go func() {
			err := runCheckpointGCTask(
				e.rm, e.db, taskID, e.JobID, e.StartTime, taskSpec,
				e.Experiment.ID, e.activeConfig.AsLegacy(), checkpoints, []string{fullDeleteGlob},
				false, taskSpec.AgentUserGroup, taskSpec.Owner, e.logCtx,
			)
			if err != nil {
				e.syslog.WithError(err).Error("failed to GC experiment checkpoints")
			}
		}()
	}

	if err := e.db.DeleteSnapshotsForExperiment(e.Experiment.ID); err != nil {
		e.syslog.WithError(err).Errorf(
			"failure to delete snapshots for experiment: %d", e.Experiment.ID)
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

func (e *internalExperiment) PerformSearcherOperations(msg *apiv1.PostSearcherOperationsRequest) error {
	e.mu.Lock()
	defer e.mu.Unlock()

	queue, err := e.searcher.GetCustomSearcherEventQueue()
	if err != nil {
		return status.Error(codes.Internal, err.Error())
	}
	var ops []searcher.Operation
	for _, searcherOp := range msg.SearcherOperations {
		switch concreteOperation := searcherOp.GetUnion().(type) {
		case *experimentv1.SearcherOperation_CreateTrial:
			op, err := searcher.CreateFromProto(concreteOperation, model.TrialWorkloadSequencerType)
			if err != nil {
				e.syslog.Error(err)
			} else {
				ops = append(ops, *op)
			}
		case *experimentv1.SearcherOperation_ShutDown:
			op, err := searcher.ShutdownFromProto(concreteOperation)
			if err != nil {
				e.syslog.Error(err)
			} else {
				ops = append(ops, *op)
			}
		case *experimentv1.SearcherOperation_TrialOperation:
			switch sub := concreteOperation.TrialOperation.GetUnion().(type) {
			case *experimentv1.TrialOperation_ValidateAfter:
				op, err := searcher.ValidateAfterFromProto(sub)
				if err != nil {
					e.syslog.Error(err)
				} else {
					ops = append(ops, *op)
				}
			}
		case *experimentv1.SearcherOperation_CloseTrial:
			op, err := searcher.CloseFromProto(concreteOperation)
			if err != nil {
				e.syslog.Error(err)
			} else {
				ops = append(ops, *op)
			}
		case *experimentv1.SearcherOperation_SetSearcherProgress:
			ops = append(ops, searcher.SetSearcherProgressFromProto(concreteOperation))
		default:
			e.syslog.Errorf("unimplemented op %+v", concreteOperation)
		}
	}
	e.syslog.Infof("processing searcher operations %+v", ops)

	// Remove newly processed events from queue.
	if err := queue.RemoveUpTo(int(msg.TriggeredByEvent.Id)); err != nil {
		return status.Error(codes.Internal, "failed to remove events from queue")
	}
	e.searcher.Record(ops)
	e.processOperations(ops, nil)
	return nil
}

func (e *internalExperiment) GetSearcherEventsWatcher() (*searcher.EventsWatcher, error) {
	e.mu.Lock()
	defer e.mu.Unlock()

	queue, err := e.searcher.GetCustomSearcherEventQueue()
	if err != nil {
		return nil, status.Error(codes.Internal, err.Error())
	}
	watcher, err := queue.Watch()
	return &watcher, err
}

func (e *internalExperiment) UnwatchEvents(id uuid.UUID) error {
	e.mu.Lock()
	defer e.mu.Unlock()

	queue, err := e.searcher.GetCustomSearcherEventQueue()
	if err != nil {
		return status.Error(codes.Internal, err.Error())
	}
	queue.Unwatch(id)
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

func (e *internalExperiment) TrialClosed(requestID model.RequestID, reason *model.ExitedReason) {
	e.mu.Lock()
	defer e.mu.Unlock()

	e.trialClosed(requestID, reason)
}

func (e *internalExperiment) trialClosed(requestID model.RequestID, reason *model.ExitedReason) {
	if reason != nil {
		e.trialReportEarlyExit(requestID, *reason)
	}
	delete(e.trials, requestID)

	ops, err := e.searcher.TrialClosed(requestID)
	e.processOperations(ops, err)
	if e.canTerminate() {
		if err := e.stop(); err != nil {
			e.syslog.WithError(err).Error("failed to stop experiment on trial closed")
		}
	}
}

func (e *internalExperiment) trialReportEarlyExit(requestID model.RequestID, reason model.ExitedReason) {
	e.syslog.WithField("requestId", requestID).Info("experiment received trial early exit")
	state, ok := e.TrialSearcherState[requestID]
	if !ok {
		e.syslog.WithField("requestID", requestID).Error("trial has no searcher state on early exit")
		return
	}

	defer func() {
		ops, err := e.searcher.TrialExitedEarly(requestID, reason)
		e.processOperations(ops, err)
	}()

	state.Complete = true
	state.Closed = true
	e.TrialSearcherState[requestID] = state

	t, ok := e.trials[requestID]
	if !ok {
		e.syslog.WithField("requestID", requestID).Warnf("missing trial to patch on early exit")
		return
	}

	err := t.PatchSearcherState(state)
	if err != nil {
		e.syslog.WithError(err).Error("patching trial search state")
	}
}

func (e *internalExperiment) trialCreated(t *trial) {
	requestID := t.searcher.Create.RequestID
	if !e.searcher.TrialIsCreated(requestID) {
		ops, err := e.searcher.TrialCreated(requestID)
		e.processOperations(ops, err)
	}
	e.trials[requestID] = t
}

// restoreTrialsFromStates from the operations that were snapshotted with the
// last experiment checkpoint.
func (e *internalExperiment) restoreTrials() {
	for _, state := range e.TrialSearcherState {
		checkpoint, err := e.checkpointForCreate(state.Create)
		if err != nil {
			e.updateState(model.StateWithReason{
				State:               model.StoppingErrorState,
				InformationalReason: fmt.Sprintf("failed getting checkpoint to restore with error %v", err),
			})
			e.syslog.Error(err)
			return
		}
		e.restoreTrial(checkpoint, state)
	}
}

func (e *internalExperiment) handleContinueExperiment(reqID model.RequestID) (*int, bool) {
	var continueFromTrialID *int
	if e.continueTrials {
		switch trial, err := db.TrialByExperimentAndRequestID(context.TODO(), e.ID, reqID); {
		case errors.Is(err, sql.ErrNoRows):
		// Trial doesn't exist, don't do anything
		case err != nil:
			e.updateState(model.StateWithReason{
				State: model.StoppingErrorState,
				InformationalReason: fmt.Sprintf(
					"hp search unable to get trial for the Request ID %v with error %v", reqID, err),
			})
			e.syslog.Error(err)
			return nil, true
		case err == nil:
			if trial.State != model.CompletedState {
				continueFromTrialID = &trial.ID
			} else {
				e.trialClosed(reqID, nil)
				return nil, true
			}
		}
	}
	return continueFromTrialID, false
}

func (e *internalExperiment) processOperations(
	ops []searcher.Operation, err error,
) {
	if _, ok := model.StoppingStates[e.State]; ok {
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

	updatedTrials := make(map[model.RequestID]bool)
	for _, operation := range ops {
		e.syslog.Debugf("handling searcher op: %v", operation)
		switch op := operation.(type) {
		case searcher.Create:
			_, ok := e.trials[op.RequestID]
			if ok {
				e.syslog.Errorf("trial %s already exists", op.RequestID)
				continue
			}

			continueFromTrialID, closed := e.handleContinueExperiment(op.RequestID)
			if closed {
				continue
			}

			checkpoint, err := e.checkpointForCreate(op)
			if err != nil {
				e.updateState(model.StateWithReason{
					State: model.StoppingErrorState,
					InformationalReason: fmt.Sprintf(
						"hp search unable to get checkpoint for new trial with error %v", err),
				})
				e.syslog.Error(err)
				continue
			}
			config := schemas.Copy(e.activeConfig)
			state := experiment.TrialSearcherState{Create: op, Complete: true}
			e.TrialSearcherState[op.RequestID] = state
			t, err := newTrial(
				e.logCtx, trialTaskID(e.ID, op.RequestID), e.JobID, e.StartTime, e.ID, e.State,
				state, e.rm, e.db, config, checkpoint, e.taskSpec, e.generatedKeys, false,
				nil, continueFromTrialID, e.TrialClosed,
			)
			if err != nil {
				e.syslog.WithError(err).Error("failed to create trial")
				e.trialClosed(op.RequestID, ptrs.Ptr(model.Errored))
				continue
			}
			e.trialCreated(t)
		case searcher.ValidateAfter:
			state := e.TrialSearcherState[op.RequestID]
			state.Op = op
			state.Complete = false
			e.TrialSearcherState[op.RequestID] = state
			updatedTrials[op.RequestID] = true
		case searcher.SetSearcherProgress:
			if err := e.searcher.SetCustomSearcherProgress(op.Progress); err != nil {
				e.syslog.WithError(err).Error("failed to set searcher progress")
			}

		case searcher.Close:
			state := e.TrialSearcherState[op.RequestID]
			state.Closed = true
			e.TrialSearcherState[op.RequestID] = state
			updatedTrials[op.RequestID] = true
		case searcher.Shutdown:
			e.syslog.WithField("op", operation).Info("searcher shutdown")
			switch {
			case op.Failure:
				e.updateState(model.StateWithReason{
					State:               model.StoppingErrorState,
					InformationalReason: "hp search failed",
				})
			case op.Cancel:
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
			panic(fmt.Sprintf("unexpected operation: %v", op))
		}
	}

	var g errgroup.Group
	g.SetLimit(maxConcurrentTrialOps)
	for requestID := range updatedTrials {
		requestID := requestID
		syslog := e.syslog.WithField("requestID", requestID)
		t, ok := e.trials[requestID]
		if !ok {
			syslog.Errorf("processOperations invalid requestID")
			continue
		}
		g.Go(func() error {
			err := t.PatchSearcherState(e.TrialSearcherState[requestID])
			if err != nil {
				syslog.WithError(err).Error("processOperations updating trial search state")
			}
			return nil
		})
	}
	_ = g.Wait() // Errors are handled in g.Go.
}

func trialTaskID(eID int, rID model.RequestID) model.TaskID {
	return model.TaskID(fmt.Sprintf("%d.%s", eID, rID))
}

var errIsNotTrialTaskID = fmt.Errorf("taskID is not a trial task ID")

func experimentIDFromTrialTaskID(taskID model.TaskID) (int, error) {
	var experimentID int
	err := db.Bun().NewSelect().
		Table("trial_id_task_id").
		Column("experiment_id").
		Join("LEFT JOIN trials ON trials.id = trial_id_task_id.trial_id").
		Where("task_id = ?", taskID).
		Scan(context.TODO(), &experimentID)
	if errors.Is(err, sql.ErrNoRows) {
		return 0, errIsNotTrialTaskID
	} else if err != nil {
		return 0, fmt.Errorf("getting experiment ID from trial task ID: %w", err)
	}

	return experimentID, nil
}

func (e *internalExperiment) checkpointForCreate(op searcher.Create) (*model.Checkpoint, error) {
	checkpoint := e.warmStartCheckpoint
	// If the Create specifies a checkpoint, ignore the experiment-wide one.
	if op.Checkpoint != nil {
		trial, err := db.TrialByExperimentAndRequestID(context.TODO(), e.ID, op.Checkpoint.RequestID)
		if err != nil {
			return nil, errors.Wrapf(err,
				"invalid request ID in Create operation: %d", op.Checkpoint.RequestID)
		}
		checkpointModel, err := checkpointFromTrialIDOrUUID(e.db, &trial.ID, nil)
		if err != nil {
			return nil, errors.Wrap(err, "checkpoint not found")
		}
		checkpoint = checkpointModel
	}
	return checkpoint, nil
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

	var g errgroup.Group
	g.SetLimit(maxConcurrentTrialOps)
	for _, t := range e.trials {
		t := t
		g.Go(func() error {
			err := t.PatchState(state)
			if err != nil {
				e.syslog.WithError(err).Error("patching trial state")
			}
			return nil
		})
	}
	_ = g.Wait() // Errors are handled in g.Go.

	if err := e.db.SaveExperimentState(e.Experiment); err != nil {
		e.syslog.Errorf("error saving experiment state: %s", err)
	}
	if e.canTerminate() {
		if err := e.stop(); err != nil {
			e.syslog.WithError(err).Error("failed to stop experiment on updateState")
		}
	}
	// The database error is explicitly ignored.
	return true
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
	db *db.PgDB, trialID *int, checkpointUUIDStr *string,
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
		checkpoint, err = db.CheckpointByUUID(checkpointUUID)
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
		resourcePool, workspaceID, e.activeConfig.Resources().SlotsPerTrial(),
	)
	switch {
	case err != nil:
		return fmt.Errorf("invalid resource pool name %s", resourcePool)
	case oldRP == rp:
		return fmt.Errorf("resource pool is unchanged (%s == %s)", oldRP, rp)
	}

	resources.SetResourcePool(rp)
	e.activeConfig.SetResources(resources)

	if err := e.db.SaveExperimentConfig(e.ID, e.activeConfig); err != nil {
		resources.SetResourcePool(oldRP)
		e.activeConfig.SetResources(resources)
		return errors.Wrapf(err, "setting experiment %d RP to %s", e.ID, rp)
	}

	var g errgroup.Group
	g.SetLimit(maxConcurrentTrialOps)
	for _, t := range e.trials {
		t := t
		g.Go(func() error {
			t.PatchRP(rp)
			return nil
		})
	}
	_ = g.Wait() // Errors handled in g.Go.

	return nil
}
