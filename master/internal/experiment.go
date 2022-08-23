package internal

import (
	"encoding/json"
	"fmt"

	"github.com/shopspring/decimal"

	"github.com/google/uuid"
	"github.com/pkg/errors"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
	"google.golang.org/protobuf/types/known/timestamppb"

	"github.com/determined-ai/determined/master/internal/api"
	"github.com/determined-ai/determined/master/internal/config"
	"github.com/determined-ai/determined/master/internal/rm"
	"github.com/determined-ai/determined/master/internal/task"

	"github.com/determined-ai/determined/master/internal/db"
	"github.com/determined-ai/determined/master/internal/hpimportance"
	"github.com/determined-ai/determined/master/internal/sproto"
	"github.com/determined-ai/determined/master/internal/telemetry"
	"github.com/determined-ai/determined/master/pkg/actor"
	"github.com/determined-ai/determined/master/pkg/logger"
	"github.com/determined-ai/determined/master/pkg/model"
	"github.com/determined-ai/determined/master/pkg/ptrs"
	"github.com/determined-ai/determined/master/pkg/schemas"
	"github.com/determined-ai/determined/master/pkg/schemas/expconf"
	"github.com/determined-ai/determined/master/pkg/searcher"
	"github.com/determined-ai/determined/master/pkg/tasks"
	"github.com/determined-ai/determined/proto/pkg/apiv1"
	"github.com/determined-ai/determined/proto/pkg/jobv1"
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
		metric    float64
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

		*model.Experiment
		taskLogger          *task.Logger
		hpImportance        *actor.Ref
		db                  *db.PgDB
		rm                  rm.ResourceManager
		searcher            *searcher.Searcher
		warmStartCheckpoint *model.Checkpoint

		taskSpec *tasks.TaskSpec

		faultToleranceEnabled bool
		restored              bool

		logCtx logger.Context
	}
)

// Create a new experiment object from the given model experiment object, along with its searcher
// and log. If the input object has no ID set, also create a new experiment in the database and set
// the returned object's ID appropriately.
func newExperiment(m *Master, expModel *model.Experiment, taskSpec *tasks.TaskSpec) (
	*experiment, error,
) {
	conf := &expModel.Config

	resources := conf.Resources()
	poolName, err := m.rm.ResolveResourcePool(
		m.system, resources.ResourcePool(), resources.SlotsPerTrial(), false,
	)
	if err != nil {
		return nil, fmt.Errorf("cannot create an experiment: %w", err)
	}

	resources.SetResourcePool(poolName)
	conf.SetResources(resources)

	method := searcher.NewSearchMethod(conf.Searcher())
	search := searcher.NewSearcher(
		conf.Reproducibility().ExperimentSeed(), method, conf.Hyperparameters(),
	)

	// Retrieve the warm start checkpoint, if provided.
	checkpoint, err := checkpointFromTrialIDOrUUID(
		m.db, conf.Searcher().SourceTrialID(), conf.Searcher().SourceCheckpointUUID())
	if err != nil {
		return nil, err
	}

	if expModel.ID == 0 {
		if err = m.db.AddExperiment(expModel); err != nil {
			return nil, err
		}
		telemetry.ReportExperimentCreated(m.system, expModel)
	}

	agentUserGroup, err := m.db.AgentUserGroup(*expModel.OwnerID)
	if err != nil {
		return nil, err
	}

	if agentUserGroup == nil {
		agentUserGroup = &m.config.Security.DefaultTask
	}
	taskSpec.AgentUserGroup = agentUserGroup

	return &experiment{
		Experiment:          expModel,
		taskLogger:          m.taskLogger,
		hpImportance:        m.hpImportance,
		db:                  m.db,
		rm:                  m.rm,
		searcher:            search,
		warmStartCheckpoint: checkpoint,

		taskSpec: taskSpec,

		faultToleranceEnabled: true,

		experimentState: experimentState{
			TrialSearcherState: map[model.RequestID]trialSearcherState{},
		},

		logCtx: logger.Context{
			"job-id":        expModel.JobID,
			"experiment-id": expModel.ID,
		},
	}, nil
}

func (e *experiment) Receive(ctx *actor.Context) error {
	switch msg := ctx.Message().(type) {
	// Searcher-related messages.
	case actor.PreStart:
		ctx.AddLabels(e.logCtx)
		e.rm.SetGroupMaxSlots(ctx, sproto.SetGroupMaxSlots{
			MaxSlots: e.Config.Resources().MaxSlots(),
			Handler:  ctx.Self(),
		})
		if err := e.setWeight(ctx, e.Config.Resources().Weight()); err != nil {
			e.updateState(ctx, model.StateWithReason{
				State:               model.StoppingErrorState,
				InformationalReason: err.Error(),
			})
			return err
		}
		if err := e.setPriority(ctx, e.Config.Resources().Priority(), true); err != nil {
			e.updateState(ctx, model.StateWithReason{
				State:               model.StoppingErrorState,
				InformationalReason: err.Error(),
			})
			return err
		}

		ctx.Self().System().TellAt(sproto.JobsActorAddr, sproto.RegisterJob{
			JobID:    e.JobID,
			JobActor: ctx.Self(),
		})

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
					ResourcePool: e.Config.Resources().ResourcePool(),
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
		ctx.Tell(e.hpImportance, hpimportance.ExperimentCreated{ID: e.ID})

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
		ctx.Tell(ctx.Child(msg.op.RequestID), state)
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
		ctx.Tell(ctx.Child(msg.requestID), state)
		ops, err := e.searcher.TrialExitedEarly(msg.requestID, msg.reason)
		e.processOperations(ctx, ops, err)
	case trialReportProgress:
		e.searcher.SetTrialProgress(msg.requestID, msg.progress)
		progress := e.searcher.Progress()
		if err := e.db.SaveExperimentProgress(e.ID, &progress); err != nil {
			ctx.Log().WithError(err).Error("failed to save experiment progress")
		}
		ctx.Tell(e.hpImportance, hpimportance.ExperimentProgress{ID: e.ID, Progress: progress})
	case trialGetSearcherState:
		state, ok := e.TrialSearcherState[msg.requestID]
		if !ok {
			ctx.Respond(api.AsErrNotFound("trial has no state"))
			return nil
		}
		ctx.Respond(state)
	case actor.ChildFailed:
		ctx.Log().WithError(msg.Error).Error("trial failed unexpectedly")
		e.trialClosed(ctx, model.MustParseRequestID(msg.Child.Address().Local()))
	case actor.ChildStopped:
		e.trialClosed(ctx, model.MustParseRequestID(msg.Child.Address().Local()))
	case trialClosed:
		e.trialClosed(ctx, msg.requestID)

	// Patch experiment messages.
	case model.StateWithReason:
		e.updateState(ctx, msg)
	case model.State:
		e.updateState(ctx, model.StateWithReason{State: msg})
	case config.ExperimentConfigPatch:
		e.Config.SetName(expconf.Name{RawString: msg.Name})
	case sproto.SetGroupMaxSlots:
		resources := e.Config.Resources()
		resources.SetMaxSlots(msg.MaxSlots)
		e.Config.SetResources(resources)
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
		ctx.Respond(e.toV1Job())

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

		ctx.Self().System().TellAt(sproto.JobsActorAddr, sproto.UnregisterJob{
			JobID: e.JobID,
		})

		state := model.StoppingToTerminalStates[e.State]
		if wasPatched, err := e.Transition(state); err != nil {
			return err
		} else if !wasPatched {
			return errors.New("experiment is already in a terminal state")
		}
		telemetry.ReportExperimentStateChanged(ctx.Self().System(), e.db, *e.Experiment)

		if err := e.db.SaveExperimentState(e.Experiment); err != nil {
			return err
		}
		ctx.Log().Infof("experiment state changed to %s", e.State)
		addr := actor.Addr(fmt.Sprintf("experiment-%d-checkpoint-gc", e.ID))

		checkpoints, err := e.db.ExperimentCheckpointsToGCRaw(
			e.Experiment.ID,
			e.Config.CheckpointStorage().SaveExperimentBest(),
			e.Config.CheckpointStorage().SaveTrialBest(),
			e.Config.CheckpointStorage().SaveTrialLatest(),
		)
		if err != nil {
			ctx.Log().WithError(err).Error("")
		}

		taskSpec := *e.taskSpec

		// May be no checkpoints to gc, if so skip
		if len(checkpoints) > 0 {
			taskID := model.TaskID(fmt.Sprintf("%d.%s", e.ID, uuid.New()))
			ckptGCTask := newCheckpointGCTask(
				e.rm, e.db, e.taskLogger, taskID, e.JobID, e.StartTime, taskSpec, e.Experiment.ID,
				e.Config.AsLegacy(), checkpoints, false, taskSpec.AgentUserGroup, taskSpec.Owner,
				e.logCtx,
			)
			ctx.Self().System().ActorOf(addr, ckptGCTask)
		}

		if e.State == model.CompletedState {
			ctx.Tell(e.hpImportance, hpimportance.ExperimentCompleted{ID: e.ID})
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
		case model.StoppingStates[e.State] || model.TerminalStates[e.State]:
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
			config := schemas.Copy(e.Config).(expconf.ExperimentConfig)
			state := trialSearcherState{Create: op, Complete: true}
			e.TrialSearcherState[op.RequestID] = state
			ctx.ActorOf(op.RequestID, newTrial(
				e.logCtx, trialTaskID(e.ID, op.RequestID), e.JobID, e.StartTime, e.ID, e.State,
				state, e.taskLogger, e.rm, e.db, config, checkpoint, e.taskSpec, false,
			))
		case searcher.ValidateAfter:
			state := e.TrialSearcherState[op.RequestID]
			state.Op = op
			state.Complete = false
			e.TrialSearcherState[op.RequestID] = state
			updatedTrials[op.RequestID] = true
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
		ctx.Tell(ctx.Child(requestID), e.TrialSearcherState[requestID])
	}
}

func trialTaskID(eID int, rID model.RequestID) model.TaskID {
	return model.TaskID(fmt.Sprintf("%d.%s", eID, rID))
}

func (e *experiment) checkpointForCreate(op searcher.Create) (*model.Checkpoint, error) {
	checkpoint := e.warmStartCheckpoint
	// If the Create specifies a checkpoint, ignore the experiment-wide one.
	if op.Checkpoint != nil {
		trial, err := e.db.TrialByExperimentAndRequestID(e.ID, op.Checkpoint.RequestID)
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
	telemetry.ReportExperimentStateChanged(ctx.Self().System(), e.db, *e.Experiment)

	ctx.Log().Infof("experiment state changed to %s", state.State)
	ctx.TellAll(state, ctx.Children()...)
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
	oldPriority := rm.DefaultSchedulingPriority
	var oldPriorityPtr *int
	resources := e.Config.Resources()
	if resources.Priority() != nil {
		oldPriority = *resources.Priority()
		oldPriorityPtr = &oldPriority
	}
	resources.SetPriority(priority)
	e.Config.SetResources(resources)

	defer func() {
		if err != nil {
			resources.SetPriority(oldPriorityPtr)
			e.Config.SetResources(resources)
			err = e.db.SaveExperimentConfig(e.Experiment)
			if err != nil {
				return
			}
		}
	}()

	if err := e.db.SaveExperimentConfig(e.Experiment); err != nil {
		return errors.Wrapf(err, "setting experiment %d priority", e.ID)
	}

	if forward {
		switch err := e.rm.SetGroupPriority(ctx, sproto.SetGroupPriority{
			Priority: *priority,
			Handler:  ctx.Self(),
		}).(type) {
		case nil:
		case rm.ErrUnsupported:
			ctx.Log().WithError(err).Debug("ignoring unsupported call to set group priority")
		default:
			return errors.Wrapf(err, "setting experiment %d priority", e.ID)
		}
	}

	return nil
}

func (e *experiment) setWeight(ctx *actor.Context, weight float64) error {
	resources := e.Config.Resources()
	oldWeight := resources.Weight()
	resources.SetWeight(weight)
	e.Config.SetResources(resources)
	if err := e.db.SaveExperimentConfig(e.Experiment); err != nil {
		resources.SetWeight(oldWeight)
		e.Config.SetResources(resources)
		return fmt.Errorf("setting experiment %d weight: %w", e.ID, err)
	}

	switch err := e.rm.SetGroupWeight(ctx, sproto.SetGroupWeight{
		Weight:  weight,
		Handler: ctx.Self(),
	}).(type) {
	case nil:
	case rm.ErrUnsupported:
		ctx.Log().WithError(err).Debug("ignoring unsupported call to set group weight")
	default:
		resources.SetWeight(oldWeight)
		e.Config.SetResources(resources)
		return fmt.Errorf("setting experiment %d weight: %w", e.ID, err)
	}
	return nil
}

func (e *experiment) setRP(ctx *actor.Context, msg sproto.SetResourcePool) error {
	resources := e.Config.Resources()
	oldRP := resources.ResourcePool()
	rp, err := e.rm.ResolveResourcePool(
		ctx, msg.ResourcePool, e.Config.Resources().SlotsPerTrial(), false,
	)
	switch {
	case err != nil:
		return fmt.Errorf("invalid resource pool name %s", msg.ResourcePool)
	case oldRP == rp:
		return fmt.Errorf("resource pool is unchanged (%s == %s)", oldRP, rp)
	}

	resources.SetResourcePool(rp)
	e.Config.SetResources(resources)

	if err := e.db.SaveExperimentConfig(e.Experiment); err != nil {
		resources.SetResourcePool(oldRP)
		e.Config.SetResources(resources)
		return errors.Wrapf(err, "setting experiment %d RP to %s", e.ID, rp)
	}

	// TODO revert the change like the other setters
	// also change to ask all?
	ctx.TellAll(sproto.ChangeRP{ResourcePool: rp}, ctx.Children()...)

	return nil
}

func (e *experiment) toV1Job() *jobv1.Job {
	j := jobv1.Job{
		JobId:          e.JobID.String(),
		EntityId:       fmt.Sprint(e.ID),
		Type:           jobv1.Type_TYPE_EXPERIMENT,
		SubmissionTime: timestamppb.New(e.StartTime),
		Username:       e.Username,
		UserId:         int32(*e.OwnerID),
		Progress:       float32(e.searcher.Progress()),
		Name:           e.Config.Name().String(),
	}

	j.IsPreemptible = config.ReadRMPreemptionStatus(j.ResourcePool)
	j.Priority = int32(config.ReadPriority(j.ResourcePool, &e.Config))
	j.Weight = config.ReadWeight(j.ResourcePool, &e.Config)

	j.ResourcePool = e.Config.Resources().ResourcePool()

	return &j
}
