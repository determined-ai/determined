package internal

import (
	"encoding/json"
	"fmt"

	"github.com/google/uuid"

	"github.com/pkg/errors"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"

	"github.com/determined-ai/determined/master/internal/api"

	"github.com/determined-ai/determined/master/internal/db"
	"github.com/determined-ai/determined/master/internal/hpimportance"
	"github.com/determined-ai/determined/master/internal/sproto"
	"github.com/determined-ai/determined/master/internal/telemetry"
	"github.com/determined-ai/determined/master/pkg/actor"
	"github.com/determined-ai/determined/master/pkg/archive"
	"github.com/determined-ai/determined/master/pkg/model"
	"github.com/determined-ai/determined/master/pkg/schemas"
	"github.com/determined-ai/determined/master/pkg/schemas/expconf"
	"github.com/determined-ai/determined/master/pkg/searcher"
	"github.com/determined-ai/determined/master/pkg/tasks"
	"github.com/determined-ai/determined/master/pkg/workload"
	"github.com/determined-ai/determined/proto/pkg/apiv1"
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
		reason    workload.ExitedReason
	}
	trialReportProgress struct {
		requestID model.RequestID
		progress  model.PartialUnits
	}
	trialGetSearcherState struct {
		requestID model.RequestID
	}
	// trialClosed is used to replay closes missed when the master dies between when a trial closing in
	// its actor.PostStop and when the experiment snapshots the trial closed.
	trialClosed struct {
		requestID model.RequestID
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
		modelDefinition     archive.Archive
		rm                  *actor.Ref
		trialLogger         *actor.Ref
		hpImportance        *actor.Ref
		db                  *db.PgDB
		searcher            *searcher.Searcher
		warmStartCheckpoint *model.Checkpoint

		taskSpec *tasks.TaskSpec

		faultToleranceEnabled bool
		restored              bool
	}
)

// Create a new experiment object from the given model experiment object, along with its searcher
// and log. If the input object has no ID set, also create a new experiment in the database and set
// the returned object's ID appropriately.
func newExperiment(master *Master, expModel *model.Experiment, taskSpec *tasks.TaskSpec) (
	*experiment, error,
) {
	conf := &expModel.Config

	resources := conf.Resources()
	poolName, err := sproto.GetResourcePool(
		master.system, resources.ResourcePool(), resources.SlotsPerTrial(), false)
	if err != nil {
		return nil, errors.Wrap(err, "cannot create an experiment")
	}

	resources.SetResourcePool(poolName)
	conf.SetResources(resources)

	method := searcher.NewSearchMethod(conf.Searcher())
	search := searcher.NewSearcher(
		conf.Reproducibility().ExperimentSeed(), method, conf.Hyperparameters(),
	)

	// Retrieve the warm start checkpoint, if provided.
	checkpoint, err := checkpointFromTrialIDOrUUID(
		master.db, conf.Searcher().SourceTrialID(), conf.Searcher().SourceCheckpointUUID())
	if err != nil {
		return nil, err
	}

	// Decompress the model definition from .tar.gz into an Archive.
	modelDefinition, err := archive.FromTarGz(expModel.ModelDefinitionBytes)
	if err != nil {
		return nil, err
	}

	if expModel.ID == 0 {
		if err = master.db.AddExperiment(expModel); err != nil {
			return nil, err
		}
	}

	agentUserGroup, err := master.db.AgentUserGroup(*expModel.OwnerID)
	if err != nil {
		return nil, err
	}

	if agentUserGroup == nil {
		agentUserGroup = &master.config.Security.DefaultTask
	}
	taskSpec.AgentUserGroup = agentUserGroup

	return &experiment{
		Experiment:          expModel,
		modelDefinition:     modelDefinition,
		rm:                  master.rm,
		trialLogger:         master.trialLogger,
		hpImportance:        master.hpImportance,
		db:                  master.db,
		searcher:            search,
		warmStartCheckpoint: checkpoint,

		taskSpec: taskSpec,

		faultToleranceEnabled: true,

		experimentState: experimentState{
			TrialSearcherState: map[model.RequestID]trialSearcherState{},
		},
	}, nil
}

func (e *experiment) Receive(ctx *actor.Context) error {
	switch msg := ctx.Message().(type) {
	// Searcher-related messages.
	case actor.PreStart:
		telemetry.ReportExperimentCreated(ctx.Self().System(), *e.Experiment)

		ctx.Tell(e.rm, sproto.SetGroupMaxSlots{
			MaxSlots: e.Config.Resources().MaxSlots(),
			Handler:  ctx.Self(),
		})
		ctx.Tell(e.rm, sproto.SetGroupWeight{Weight: e.Config.Resources().Weight(), Handler: ctx.Self()})
		ctx.Tell(e.rm, sproto.SetGroupPriority{
			Priority: e.Config.Resources().Priority(),
			Handler:  ctx.Self(),
		})

		if e.restored {
			e.restoreTrials(ctx)
			return nil
		}

		ops, err := e.searcher.InitialOperations()
		if err != nil {
			return errors.Wrap(err, "failed to generate initial operations")
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
	case model.State:
		e.updateState(ctx, msg)
	case sproto.SetGroupMaxSlots:
		resources := e.Config.Resources()
		resources.SetMaxSlots(msg.MaxSlots)
		e.Config.SetResources(resources)
		msg.Handler = ctx.Self()
		ctx.Tell(e.rm, msg)
	case sproto.SetGroupWeight:
		resources := e.Config.Resources()
		resources.SetWeight(msg.Weight)
		e.Config.SetResources(resources)
		msg.Handler = ctx.Self()
		ctx.Tell(e.rm, msg)
	case sproto.SetGroupPriority:
		resources := e.Config.Resources()
		resources.SetPriority(msg.Priority)
		e.Config.SetResources(resources)
		msg.Handler = ctx.Self()
		ctx.Tell(e.rm, msg)

	// Experiment shutdown logic.
	case actor.PostStop:
		if err := e.db.SaveExperimentProgress(e.ID, nil); err != nil {
			ctx.Log().Error(err)
		}

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
			true,
		)
		if err != nil {
			ctx.Log().WithError(err).Error("")
		}

		taskSpec := *e.taskSpec
		ctx.Self().System().ActorOf(addr, &checkpointGCTask{
			taskID: model.TaskID(fmt.Sprintf("%d.%s", e.ID, uuid.New())),
			GCCkptSpec: tasks.GCCkptSpec{
				Base:         taskSpec,
				ExperimentID: e.Experiment.ID,
				LegacyConfig: e.Config.AsLegacy(),
				ToDelete:     checkpoints,
			},

			rm: e.rm,
			db: e.db,
		})

		if e.State == model.CompletedState {
			ctx.Tell(e.hpImportance, hpimportance.ExperimentCompleted{ID: e.ID})
		}

		if err := e.db.DeleteSnapshotsForExperiment(e.Experiment.ID); err != nil {
			ctx.Log().WithError(err).Errorf(
				"failure to delete snapshots for experiment: %d", e.Experiment.ID)
		}

		ctx.Log().Info("experiment shut down successfully")

	case *apiv1.ActivateExperimentRequest:
		switch ok := e.updateState(ctx, model.ActiveState); ok {
		case true:
			ctx.Respond(&apiv1.ActivateExperimentResponse{})
		default:
			ctx.Respond(status.Errorf(codes.FailedPrecondition,
				"experiment in incompatible state %s", e.State))
		}

	case *apiv1.PauseExperimentRequest:
		switch ok := e.updateState(ctx, model.PausedState); ok {
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
			if ok := e.updateState(ctx, model.StoppingCanceledState); !ok {
				ctx.Respond(status.Errorf(codes.FailedPrecondition,
					"experiment in incompatible state %s", e.State,
				))
			} else {
				ctx.Respond(&apiv1.CancelExperimentResponse{})
			}
		}

	case *apiv1.KillExperimentRequest:
		switch {
		case model.StoppingStates[e.State] || model.TerminalStates[e.State]:
			ctx.Respond(&apiv1.KillExperimentResponse{})
		default:
			if ok := e.updateState(ctx, model.StoppingKilledState); !ok {
				ctx.Respond(status.Errorf(codes.FailedPrecondition,
					"experiment in incompatible state %s", e.State,
				))
			} else {
				ctx.Respond(&apiv1.KillExperimentResponse{})
			}
		}
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
			e.updateState(ctx, model.StoppingErrorState)
			ctx.Log().Error(err)
			return
		}
		e.restoreTrial(ctx, checkpoint, state)
	}
}

func (e *experiment) processOperations(
	ctx *actor.Context, ops []searcher.Operation, err error) {
	if _, ok := model.StoppingStates[e.State]; ok {
		return
	}
	if err != nil {
		ctx.Log().Error(err)
		e.updateState(ctx, model.StoppingErrorState)
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
				e.updateState(ctx, model.StoppingErrorState)
				ctx.Log().Error(err)
				return
			}
			config := schemas.Copy(e.Config).(expconf.ExperimentConfig)
			state := trialSearcherState{Create: op, Complete: true}
			e.TrialSearcherState[op.RequestID] = state
			ctx.ActorOf(op.RequestID, newTrial(
				trialTaskID(e.ID, op.RequestID), e.ID, e.State, state, e.rm, e.trialLogger, e.db,
				config, checkpoint, e.taskSpec, e.modelDefinition,
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
			if op.Failure {
				e.updateState(ctx, model.StoppingErrorState)
			} else {
				e.updateState(ctx, model.StoppingCompletedState)
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

func (e *experiment) updateState(ctx *actor.Context, state model.State) bool {
	if wasPatched, err := e.Transition(state); err != nil {
		ctx.Log().Errorf("error transitioning experiment state: %s", err)
		return false
	} else if !wasPatched {
		return true
	}
	telemetry.ReportExperimentStateChanged(ctx.Self().System(), e.db, *e.Experiment)

	ctx.Log().Infof("experiment state changed to %s", state)
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
