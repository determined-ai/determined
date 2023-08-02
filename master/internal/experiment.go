package internal

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"

	"github.com/determined-ai/determined/master/internal/job/jobservice"
	"golang.org/x/sync/errgroup"

	"github.com/shopspring/decimal"
	log "github.com/sirupsen/logrus"
	"github.com/uptrace/bun"

	"github.com/google/uuid"
	"github.com/pkg/errors"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
	"google.golang.org/protobuf/types/known/timestamppb"

	"github.com/determined-ai/determined/master/internal/api"
	"github.com/determined-ai/determined/master/internal/config"
	"github.com/determined-ai/determined/master/internal/rm"
	"github.com/determined-ai/determined/master/internal/rm/rmerrors"
	"github.com/determined-ai/determined/master/internal/user"

	"github.com/determined-ai/determined/master/internal/db"
	"github.com/determined-ai/determined/master/internal/sproto"
	"github.com/determined-ai/determined/master/internal/telemetry"
	"github.com/determined-ai/determined/master/internal/webhooks"
	"github.com/determined-ai/determined/master/internal/workspace"
	"github.com/determined-ai/determined/master/pkg/actor"
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
	"github.com/determined-ai/determined/proto/pkg/jobv1"
)

const (
	maxConcurrentTrialOps = 16
)

// Experiment-specific actor messages.
type (
	// Searcher-related messages.
	trialCreated struct {
		requestID model.RequestID
	}
	trialCompleteOperation struct {
		requestID model.RequestID
		op        searcher.ValidateAfter
		metric    interface{}
	}
	trialReportEarlyExit struct {
		requestID model.RequestID
		reason    model.ExitedReason
	}
	trialReportProgress struct {
		requestID model.RequestID
		progress  searcher.PartialUnits
	}
	trialGetSearcherState struct {
		requestID model.RequestID
	}

	// trialClosed is used to replay closes missed when the master dies between when a trial closing
	// in its actor.PostStop and when the experiment snapshots the trial closed.
	trialClosed struct {
		requestID model.RequestID
	}

	// userInitiatedEarlyExit is a user-injected message, provided through the early exit API. It
	// _should_ indicate the user is exiting, but in the event they don't, we will clean them up.
	userInitiatedEarlyExit struct {
		requestID model.RequestID
		reason    model.ExitedReason
	}

	// UnwatchEvents is initiated from the get searcher events API. It deletes the watcher with the
	// given ID.
	UnwatchEvents struct {
		id uuid.UUID
	}
)

type (
	trialSearcherState struct {
		Create   searcher.Create
		Op       searcher.ValidateAfter
		Complete bool
		Closed   bool
	}

	experimentState struct {
		SearcherState      json.RawMessage                        `json:"searcher_state"`
		TrialSearcherState map[model.RequestID]trialSearcherState `json:"trial_searcher_state"`
	}

	experiment struct {
		experimentState

		trials map[model.RequestID]*trial

		*model.Experiment
		activeConfig        expconf.ExperimentConfig
		db                  *db.PgDB
		rm                  rm.ResourceManager
		searcher            *searcher.Searcher
		warmStartCheckpoint *model.Checkpoint

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
) (*experiment, []command.LaunchWarning, error) {
	resources := activeConfig.Resources()
	workspaceModel, err := workspace.WorkspaceByProjectID(context.TODO(), expModel.ProjectID)
	if err != nil && errors.Cause(err) != sql.ErrNoRows {
		return nil, nil, err
	}
	workspaceID := resolveWorkspaceID(workspaceModel)
	poolName, err := m.rm.ResolveResourcePool(
		m.system, resources.ResourcePool(), workspaceID, resources.SlotsPerTrial(),
	)
	if err != nil {
		return nil, nil, fmt.Errorf("cannot create an experiment: %w", err)
	}

	var launchWarnings []command.LaunchWarning
	if expModel.ID == 0 {
		err := m.rm.ValidateResources(m.system, poolName, resources.SlotsPerTrial(), false)
		if err != nil {
			return nil, nil, fmt.Errorf("validating resources: %v", err)
		}
		launchWarnings, err = m.rm.ValidateResourcePoolAvailability(
			m.system,
			poolName,
			resources.SlotsPerTrial(),
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

	agentUserGroup, err := user.GetAgentUserGroup(*expModel.OwnerID, expModel)
	if err != nil {
		return nil, launchWarnings, err
	}

	taskSpec.AgentUserGroup = agentUserGroup

	generatedKeys, err := ssh.GenerateKey(taskSpec.SSHRsaSize, nil)
	if err != nil {
		return nil, nil, errors.Wrap(err, "generating ssh keys for trials")
	}

	return &experiment{
		Experiment:          expModel,
		activeConfig:        activeConfig,
		db:                  m.db,
		rm:                  m.rm,
		searcher:            search,
		warmStartCheckpoint: checkpoint,

		trials: map[model.RequestID]*trial{},

		taskSpec:      taskSpec,
		generatedKeys: generatedKeys,

		faultToleranceEnabled: true,

		experimentState: experimentState{
			TrialSearcherState: map[model.RequestID]trialSearcherState{},
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
) (*experiment, []command.LaunchWarning, error) {
	expModel.State = model.PausedState
	expModel.Unmanaged = true

	if err := db.AddExperimentTx(ctx, idb, expModel, activeConfig, true); err != nil {
		return nil, nil, err
	}
	telemetry.ReportExperimentCreated(expModel.ID, activeConfig)

	// Will only have the model, nothing required for the experiment actor.
	return &experiment{
		Experiment: expModel,
	}, nil, nil
}

func (e *experiment) Receive(ctx *actor.Context) error {
	switch msg := ctx.Message().(type) {
	// Searcher-related messages.
	case actor.PreStart:
		ctx.AddLabels(e.logCtx)
		e.rm.SetGroupMaxSlots(ctx, sproto.SetGroupMaxSlots{
			MaxSlots: e.activeConfig.Resources().MaxSlots(),
			Handler:  ctx.Self(),
		})
		if err := e.setWeight(ctx, e.activeConfig.Resources().Weight()); err != nil {
			e.updateState(ctx, model.StateWithReason{
				State:               model.StoppingErrorState,
				InformationalReason: err.Error(),
			})
			return err
		}
		if err := e.setPriority(ctx, e.activeConfig.Resources().Priority(), true); err != nil {
			e.updateState(ctx, model.StateWithReason{
				State:               model.StoppingErrorState,
				InformationalReason: err.Error(),
			})
			return err
		}

		jobservice.Default.RegisterJob(e.JobID, ctx.Self())

		if e.restored {
			j, err := e.db.JobByID(e.JobID)
			if err != nil {
				e.updateState(ctx, model.StateWithReason{
					State:               model.StoppingErrorState,
					InformationalReason: err.Error(),
				})
				return err
			}

			if j.QPos.GreaterThan(decimal.Zero) {
				e.rm.RecoverJobPosition(ctx, sproto.RecoverJobPosition{
					JobID:        e.JobID,
					JobPosition:  j.QPos,
					ResourcePool: e.activeConfig.Resources().ResourcePool(),
				})
			}

			e.restoreTrials(ctx)
			return nil
		}

		ops, err := e.searcher.InitialOperations()
		if err != nil {
			err = errors.Wrap(err, "failed to generate initial operations")
			e.updateState(ctx, model.StateWithReason{
				State:               model.StoppingErrorState,
				InformationalReason: err.Error(),
			})
			return err
		}
		e.processOperations(ctx, ops, nil)

	case trialCreated:
		ops, err := e.searcher.TrialCreated(msg.requestID)
		e.processOperations(ctx, ops, err)
	case trialCompleteOperation:
		state, ok := e.TrialSearcherState[msg.op.RequestID]
		switch {
		case !ok:
			ctx.Respond(api.AsValidationError("no such trial"))
			return nil
		case msg.op != state.Op:
			ctx.Respond(api.AsValidationError("expected op %v but received op %v", state.Op, msg.op))
			return nil
		case state.Complete:
			ctx.Respond(api.AsValidationError("received op %v which was previously completed", msg.op))
			return nil
		}

		state.Complete = true
		e.TrialSearcherState[msg.op.RequestID] = state

		t, ok := e.trials[msg.op.RequestID]
		if !ok {
			ctx.Log().Warnf("missing trial to propogate complete op: %s", msg.op.RequestID)
			return nil
		}

		err := t.PatchSearcherState(state)
		if err != nil {
			ctx.Log().WithError(err).Error("patching trial search state")
			ctx.Tell(ctx.Self(), trialClosed{requestID: msg.op.RequestID})
			return nil
		}

		ops, err := e.searcher.ValidationCompleted(msg.requestID, msg.metric, msg.op)
		e.processOperations(ctx, ops, err)
	case trialReportEarlyExit:
		state, ok := e.TrialSearcherState[msg.requestID]
		if !ok {
			ctx.Respond(api.AsValidationError("trial has no state"))
			return nil
		}
		state.Complete = true
		state.Closed = true
		e.TrialSearcherState[msg.requestID] = state

		t, ok := e.trials[msg.requestID]
		if !ok {
			ctx.Log().Warnf("missing trial to propogate complete op: %s", msg.requestID)
			return nil
		}

		err := t.PatchSearcherState(state)
		if err != nil {
			ctx.Log().WithError(err).Error("patching trial search state")
			ctx.Tell(ctx.Self(), trialClosed{requestID: msg.requestID})
			return nil
		}

		ops, err := e.searcher.TrialExitedEarly(msg.requestID, msg.reason)
		e.processOperations(ctx, ops, err)
	case trialReportProgress:
		e.searcher.SetTrialProgress(msg.requestID, msg.progress)
		progress := e.searcher.Progress()
		if err := e.db.SaveExperimentProgress(e.ID, &progress); err != nil {
			ctx.Log().WithError(err).Error("failed to save experiment progress")
		}
	case trialGetSearcherState:
		state, ok := e.TrialSearcherState[msg.requestID]
		if !ok {
			ctx.Respond(api.AsErrNotFound("trial has no state"))
			return nil
		}
		ctx.Respond(state)
	case trialClosed:
		e.trialClosed(ctx, msg.requestID)
	case userInitiatedEarlyExit:
		ref, ok := e.trials[msg.requestID]
		if !ok {
			ctx.Respond(fmt.Errorf("no such trial: %s", msg.requestID))
			return nil
		}

		err := ref.SetUserInitiatedEarlyExit(msg)
		if err != nil {
			ctx.Log().WithError(err).Error("setting user initiated early exit")
			ctx.Tell(ctx.Self(), trialClosed{requestID: msg.requestID})
			ctx.Respond(err)
			return nil
		}

	// Patch experiment messages.
	case model.StateWithReason:
		e.updateState(ctx, msg)
	case model.State:
		e.updateState(ctx, model.StateWithReason{State: msg})
	case config.ExperimentConfigPatch:
		e.activeConfig.SetName(expconf.Name{RawString: msg.Name})
	case sproto.SetGroupMaxSlots:
		resources := e.activeConfig.Resources()
		resources.SetMaxSlots(msg.MaxSlots)
		e.activeConfig.SetResources(resources)
		msg.Handler = ctx.Self()
		e.rm.SetGroupMaxSlots(ctx, msg)
	case sproto.NotifyRMPriorityChange:
		err := e.setPriority(ctx, &msg.Priority, false)
		if err != nil {
			ctx.Log().WithError(err).Info("setting experiment job priority")
		}
		if ctx.ExpectingResponse() {
			ctx.Respond(err)
		}
	case sproto.SetGroupWeight:
		err := e.setWeight(ctx, msg.Weight)
		if err != nil {
			ctx.Log().WithError(err).Info("setting experiment job weight")
		}
		if ctx.ExpectingResponse() {
			ctx.Respond(err)
		}
	case sproto.SetGroupPriority:
		err := e.setPriority(ctx, &msg.Priority, true)
		if err != nil {
			ctx.Log().WithError(err).Info("setting experiment job priority")
		}
		if ctx.ExpectingResponse() {
			ctx.Respond(err)
		}
	case sproto.GetJob:
		j, err := e.toV1Job()
		if err != nil && err != sql.ErrNoRows {
			// FIXME: DET-9563 workspace and/or project is deleted.
			ctx.Respond(err)
		} else {
			ctx.Respond(j)
		}

	case sproto.SetResourcePool:
		if err := e.setRP(ctx, msg); err != nil {
			ctx.Respond(err)
		}

	case sproto.RegisterJobPosition:
		err := e.db.UpdateJobPosition(msg.JobID, msg.JobPosition)
		if err != nil {
			ctx.Log().WithError(err).Errorf("persisting position for job %s failed", msg.JobID)
		}

	// Experiment shutdown logic.
	case actor.PostStop:
		if e.State == model.CompletedState || e.State == model.StoppingCompletedState {
			if err := e.db.SaveExperimentProgress(e.ID, ptrs.Ptr(1.0)); err != nil {
				ctx.Log().Error(err)
			}
		}
		jobservice.Default.UnregisterJob(e.JobID)
		state := model.StoppingToTerminalStates[e.State]
		if wasPatched, err := e.Transition(state); err != nil {
			return err
		} else if !wasPatched {
			return errors.New("experiment is already in a terminal state")
		}
		telemetry.ReportExperimentStateChanged(e.db, e.Experiment)
		if err := webhooks.ReportExperimentStateChanged(
			context.TODO(), *e.Experiment, e.activeConfig,
		); err != nil {
			log.WithError(err).Error("failed to send experiment state change webhook")
		}

		if err := e.db.SaveExperimentState(e.Experiment); err != nil {
			return err
		}
		ctx.Log().Infof("experiment state changed to %s", e.State)

		checkpoints, err := e.db.ExperimentCheckpointsToGCRaw(
			e.Experiment.ID,
			e.activeConfig.CheckpointStorage().SaveExperimentBest(),
			e.activeConfig.CheckpointStorage().SaveTrialBest(),
			e.activeConfig.CheckpointStorage().SaveTrialLatest(),
		)
		if err != nil {
			ctx.Log().WithError(err).Error("")
		}

		taskSpec := *e.taskSpec

		// May be no checkpoints to gc, if so skip
		if len(checkpoints) > 0 {
			taskID := model.TaskID(fmt.Sprintf("%d.%s", e.ID, uuid.New()))
			go func() {
				err := runCheckpointGCTask(
					ctx.Self().System(), e.rm, e.db, taskID, e.JobID, e.StartTime, taskSpec,
					e.Experiment.ID, e.activeConfig.AsLegacy(), checkpoints, []string{fullDeleteGlob},
					false, taskSpec.AgentUserGroup, taskSpec.Owner, e.logCtx,
				)
				if err != nil {
					ctx.Log().WithError(err).Error("failed to GC experiment checkpoints")
				}
			}()
		}

		if err := e.db.DeleteSnapshotsForExperiment(e.Experiment.ID); err != nil {
			ctx.Log().WithError(err).Errorf(
				"failure to delete snapshots for experiment: %d", e.Experiment.ID)
		}

		if err := e.db.DeleteUserSessionByToken(taskSpec.UserSessionToken); err != nil {
			ctx.Log().WithError(err).Errorf(
				"failure to delete user session for experiment: %d", e.Experiment.ID)
		}

		ctx.Log().Info("experiment shut down successfully")

	case *apiv1.PostSearcherOperationsRequest:
		queue, err := e.searcher.GetCustomSearcherEventQueue()
		if err != nil {
			ctx.Respond(status.Error(codes.Internal, err.Error()))
			return nil
		}
		var ops []searcher.Operation
		for _, searcherOp := range msg.SearcherOperations {
			switch concreteOperation := searcherOp.GetUnion().(type) {
			case *experimentv1.SearcherOperation_CreateTrial:
				op, err := searcher.CreateFromProto(concreteOperation, model.TrialWorkloadSequencerType)
				if err != nil {
					ctx.Log().Error(err)
				} else {
					ops = append(ops, *op)
				}
			case *experimentv1.SearcherOperation_ShutDown:
				op, err := searcher.ShutdownFromProto(concreteOperation)
				if err != nil {
					ctx.Log().Error(err)
				} else {
					ops = append(ops, *op)
				}
			case *experimentv1.SearcherOperation_TrialOperation:
				switch sub := concreteOperation.TrialOperation.GetUnion().(type) {
				case *experimentv1.TrialOperation_ValidateAfter:
					op, err := searcher.ValidateAfterFromProto(sub)
					if err != nil {
						ctx.Log().Error(err)
					} else {
						ops = append(ops, *op)
					}
				}
			case *experimentv1.SearcherOperation_CloseTrial:
				op, err := searcher.CloseFromProto(concreteOperation)
				if err != nil {
					ctx.Log().Error(err)
				} else {
					ops = append(ops, *op)
				}
			case *experimentv1.SearcherOperation_SetSearcherProgress:
				ops = append(ops, searcher.SetSearcherProgressFromProto(concreteOperation))
			default:
				ctx.Log().Errorf("unimplemented op %+v", concreteOperation)
			}
		}
		ctx.Log().Infof("processing searcher operations %+v", ops)

		// Remove newly processed events from queue.
		if err := queue.RemoveUpTo(int(msg.TriggeredByEvent.Id)); err != nil {
			ctx.Respond(status.Error(codes.Internal, "failed to remove events from queue"))
		} else {
			e.searcher.Record(ops)
			e.processOperations(ctx, ops, nil)
			ctx.Respond(&apiv1.PostSearcherOperationsResponse{})
		}

	case *apiv1.GetSearcherEventsRequest:
		if queue, err := e.searcher.GetCustomSearcherEventQueue(); err != nil {
			ctx.Respond(status.Error(codes.Internal, err.Error()))
		} else {
			if w, err := queue.Watch(); err != nil {
				ctx.Respond(err)
			} else {
				ctx.Respond(w)
			}
		}

	case UnwatchEvents:
		if queue, err := e.searcher.GetCustomSearcherEventQueue(); err != nil {
			ctx.Respond(status.Error(codes.Internal, err.Error()))
		} else {
			queue.Unwatch(msg.id)
		}

	case *apiv1.ActivateExperimentRequest:
		switch ok := e.updateState(ctx, model.StateWithReason{
			State:               model.ActiveState,
			InformationalReason: "user requested activation",
		}); ok {
		case true:
			ctx.Respond(&apiv1.ActivateExperimentResponse{})
		default:
			ctx.Respond(status.Errorf(codes.FailedPrecondition,
				"experiment in incompatible state %s", e.State))
		}

	case *apiv1.PauseExperimentRequest:
		switch ok := e.updateState(ctx, model.StateWithReason{
			State:               model.PausedState,
			InformationalReason: "user requested pause",
		}); ok {
		case true:
			ctx.Respond(&apiv1.PauseExperimentResponse{})
		default:
			ctx.Respond(status.Errorf(codes.FailedPrecondition,
				"experiment in incompatible state %s", e.State))
		}

	case *apiv1.CancelExperimentRequest:
		switch {
		case model.StoppingStates[e.State] || model.TerminalStates[e.State]:
			ctx.Respond(&apiv1.CancelExperimentResponse{})
		default:
			switch ok := e.updateState(ctx, model.StateWithReason{
				State:               model.StoppingCanceledState,
				InformationalReason: "user requested cancellation",
			}); ok {
			case true:
				ctx.Respond(&apiv1.CancelExperimentResponse{})
			default:
				ctx.Respond(status.Errorf(codes.FailedPrecondition,
					"experiment in incompatible state %s", e.State,
				))
			}
		}

	case *apiv1.KillExperimentRequest:
		switch {
		case e.State == model.StoppingKilledState || model.TerminalStates[e.State]:
			ctx.Respond(&apiv1.KillExperimentResponse{})
		default:
			switch ok := e.updateState(ctx, model.StateWithReason{
				State:               model.StoppingKilledState,
				InformationalReason: "user requested kill",
			}); ok {
			case true:
				ctx.Respond(&apiv1.KillExperimentResponse{})
			default:
				ctx.Respond(status.Errorf(codes.FailedPrecondition,
					"experiment in incompatible state %s", e.State,
				))
			}
		}

	case sproto.InvalidResourcesRequestError:
		e.updateState(ctx, model.StateWithReason{
			State:               model.StoppingErrorState,
			InformationalReason: msg.Cause.Error(),
		})

	default:
		return status.Errorf(codes.InvalidArgument, "unknown message type %T", msg)
	}

	return nil
}

func (e *experiment) trialClosed(ctx *actor.Context, requestID model.RequestID) {
	ops, err := e.searcher.TrialClosed(requestID)
	e.processOperations(ctx, ops, err)
	if e.canTerminate(ctx) {
		ctx.Self().Stop()
	}
}

// restoreTrialsFromStates from the operations that were snapshotted with the
// last experiment checkpoint.
func (e *experiment) restoreTrials(ctx *actor.Context) {
	for _, state := range e.TrialSearcherState {
		checkpoint, err := e.checkpointForCreate(state.Create)
		if err != nil {
			e.updateState(ctx, model.StateWithReason{
				State:               model.StoppingErrorState,
				InformationalReason: fmt.Sprintf("failed getting checkpoint to restore with error %v", err),
			})
			ctx.Log().Error(err)
			return
		}
		e.restoreTrial(ctx, checkpoint, state)
	}
}

func (e *experiment) processOperations(
	ctx *actor.Context, ops []searcher.Operation, err error,
) {
	if _, ok := model.StoppingStates[e.State]; ok {
		return
	}
	if err != nil {
		ctx.Log().Error(err)
		e.updateState(ctx, model.StateWithReason{
			State:               model.StoppingErrorState,
			InformationalReason: fmt.Sprintf("encountered error %v", err),
		})
		return
	}

	defer e.snapshotAndSave(ctx)

	updatedTrials := make(map[model.RequestID]bool)
	for _, operation := range ops {
		ctx.Log().Debugf("handling searcher op: %v", operation)
		switch op := operation.(type) {
		case searcher.Create:
			checkpoint, err := e.checkpointForCreate(op)
			if err != nil {
				e.updateState(ctx, model.StateWithReason{
					State: model.StoppingErrorState,
					InformationalReason: fmt.Sprintf(
						"hp search unable to get checkpoint for new trial with error %v", err),
				})
				ctx.Log().Error(err)
				return
			}
			config := schemas.Copy(e.activeConfig)
			state := trialSearcherState{Create: op, Complete: true}
			e.TrialSearcherState[op.RequestID] = state
			t, err := newTrial(
				e.logCtx, trialTaskID(e.ID, op.RequestID), e.JobID, e.StartTime, e.ID, e.State,
				state, e.rm, e.db, config, checkpoint, e.taskSpec, e.generatedKeys, false,
				nil, false, ctx.Self().System(), ctx.Self(),
			)
			if err != nil {
				ctx.Log().WithError(err).Error("failed to create trial")
				ctx.Tell(ctx.Self(), trialClosed{requestID: op.RequestID})
				continue
			}
			e.trials[op.RequestID] = t
		case searcher.ValidateAfter:
			state := e.TrialSearcherState[op.RequestID]
			state.Op = op
			state.Complete = false
			e.TrialSearcherState[op.RequestID] = state
			updatedTrials[op.RequestID] = true
		case searcher.SetSearcherProgress:
			if err := e.searcher.SetCustomSearcherProgress(op.Progress); err != nil {
				ctx.Respond(status.Error(codes.Internal, err.Error()))
			}

		case searcher.Close:
			state := e.TrialSearcherState[op.RequestID]
			state.Closed = true
			e.TrialSearcherState[op.RequestID] = state
			updatedTrials[op.RequestID] = true
		case searcher.Shutdown:
			switch {
			case op.Failure:
				e.updateState(ctx, model.StateWithReason{
					State:               model.StoppingErrorState,
					InformationalReason: "hp search failed",
				})
			case op.Cancel:
				e.updateState(ctx, model.StateWithReason{
					State:               model.StoppingCanceledState,
					InformationalReason: "hp search canceled",
				})
			default:
				e.updateState(ctx, model.StateWithReason{
					State:               model.StoppingCompletedState,
					InformationalReason: "hp search completed",
				})
			}
		default:
			panic(fmt.Sprintf("unexpected operation: %v", op))
		}
	}

	for requestID := range updatedTrials {
		t, ok := e.trials[requestID]
		if !ok {
			ctx.Log().Errorf("invalid request ID: %v", requestID)
			continue
		}

		err := t.PatchSearcherState(e.TrialSearcherState[requestID])
		if err != nil {
			ctx.Log().WithError(err).Error("updating trial search state")
			ctx.Tell(ctx.Self(), trialClosed{requestID: requestID})
			continue
		}
	}
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

func (e *experiment) checkpointForCreate(op searcher.Create) (*model.Checkpoint, error) {
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

func (e *experiment) updateState(ctx *actor.Context, state model.StateWithReason) bool {
	if wasPatched, err := e.Transition(state.State); err != nil {
		ctx.Log().Errorf("error transitioning experiment state: %s", err)
		return false
	} else if !wasPatched {
		return true
	}
	telemetry.ReportExperimentStateChanged(e.db, e.Experiment)
	if err := webhooks.ReportExperimentStateChanged(
		context.TODO(), *e.Experiment, e.activeConfig,
	); err != nil {
		log.WithError(err).Error("failed to send experiment state change webhook")
	}

	ctx.Log().Infof("experiment state changed to %s", state.State)

	var g errgroup.Group
	g.SetLimit(maxConcurrentTrialOps)
	for rID, t := range e.trials {
		rID, t := rID, t
		g.Go(func() error {
			err := t.PatchState(state)
			if err != nil {
				ctx.Log().WithError(err).Error("patching trial state")
				ctx.Tell(ctx.Self(), trialClosed{requestID: rID})
			}
			return nil
		})
	}
	_ = g.Wait() // Errors are handled in g.Go.

	if err := e.db.SaveExperimentState(e.Experiment); err != nil {
		ctx.Log().Errorf("error saving experiment state: %s", err)
	}
	if e.canTerminate(ctx) {
		ctx.Self().Stop()
	}
	// The database error is explicitly ignored.
	return true
}

func (e *experiment) canTerminate(ctx *actor.Context) bool {
	return model.StoppingStates[e.State] && len(ctx.Children()) == 0
}

func (e *experiment) Snapshot() (json.RawMessage, error) {
	searcherSnapshot, err := e.searcher.Snapshot()
	if err != nil {
		return nil, errors.Wrap(err, "failed to snapshot searcher")
	}
	e.SearcherState = searcherSnapshot
	experimentSnapshot, err := json.Marshal(e.experimentState)
	return experimentSnapshot, errors.Wrap(err, "failed to marshal experiment")
}

func (e *experiment) Restore(experimentSnapshot json.RawMessage) error {
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

func (e *experiment) setPriority(ctx *actor.Context, priority *int, forward bool) (err error) {
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
		switch err := e.rm.SetGroupPriority(ctx, sproto.SetGroupPriority{
			Priority: *priority,
			Handler:  ctx.Self(),
		}).(type) {
		case nil:
		case rmerrors.ErrUnsupported:
			ctx.Log().WithError(err).Debug("ignoring unsupported call to set group priority")
		default:
			return errors.Wrapf(err, "setting experiment %d priority", e.ID)
		}
	}

	return nil
}

func (e *experiment) setWeight(ctx *actor.Context, weight float64) error {
	resources := e.activeConfig.Resources()
	oldWeight := resources.Weight()
	resources.SetWeight(weight)
	e.activeConfig.SetResources(resources)
	if err := e.db.SaveExperimentConfig(e.ID, e.activeConfig); err != nil {
		resources.SetWeight(oldWeight)
		e.activeConfig.SetResources(resources)
		return fmt.Errorf("setting experiment %d weight: %w", e.ID, err)
	}

	switch err := e.rm.SetGroupWeight(ctx, sproto.SetGroupWeight{
		Weight:  weight,
		Handler: ctx.Self(),
	}).(type) {
	case nil:
	case rmerrors.ErrUnsupported:
		ctx.Log().WithError(err).Debug("ignoring unsupported call to set group weight")
	default:
		resources.SetWeight(oldWeight)
		e.activeConfig.SetResources(resources)
		return fmt.Errorf("setting experiment %d weight: %w", e.ID, err)
	}
	return nil
}

func (e *experiment) setRP(ctx *actor.Context, msg sproto.SetResourcePool) error {
	resources := e.activeConfig.Resources()
	oldRP := resources.ResourcePool()
	workspaceModel, err := workspace.WorkspaceByProjectID(context.TODO(), e.ProjectID)
	if err != nil && errors.Cause(err) != sql.ErrNoRows {
		return err
	}
	workspaceID := resolveWorkspaceID(workspaceModel)
	rp, err := e.rm.ResolveResourcePool(
		ctx.Self().System(), msg.ResourcePool, workspaceID, e.activeConfig.Resources().SlotsPerTrial(),
	)
	switch {
	case err != nil:
		return fmt.Errorf("invalid resource pool name %s", msg.ResourcePool)
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

	// TODO revert the change like the other setters
	// also change to ask all?
	// for rID, t := range e.trials
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

func (e *experiment) toV1Job() (*jobv1.Job, error) {
	workspace, err := workspace.WorkspaceByProjectID(context.TODO(), e.ProjectID)
	if err != nil {
		return nil, err
	}

	j := jobv1.Job{
		JobId:          e.JobID.String(),
		EntityId:       fmt.Sprint(e.ID),
		Type:           jobv1.Type_TYPE_EXPERIMENT,
		SubmissionTime: timestamppb.New(e.StartTime),
		Username:       e.Username,
		UserId:         int32(*e.OwnerID),
		Progress:       float32(e.searcher.Progress()),
		Name:           e.activeConfig.Name().String(),
		WorkspaceId:    int32(workspace.ID),
	}

	j.IsPreemptible = config.ReadRMPreemptionStatus(j.ResourcePool)
	j.Priority = int32(config.ReadPriority(j.ResourcePool, &e.activeConfig))
	j.Weight = config.ReadWeight(j.ResourcePool, &e.activeConfig)

	j.ResourcePool = e.activeConfig.Resources().ResourcePool()

	return &j, nil
}
