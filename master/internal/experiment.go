package internal

import (
	"encoding/json"
	"fmt"

	"github.com/pkg/errors"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"

	"github.com/determined-ai/determined/master/internal/db"
	"github.com/determined-ai/determined/master/internal/hpimportance"
	"github.com/determined-ai/determined/master/internal/sproto"
	"github.com/determined-ai/determined/master/internal/telemetry"
	"github.com/determined-ai/determined/master/pkg/actor"
	"github.com/determined-ai/determined/master/pkg/archive"
	"github.com/determined-ai/determined/master/pkg/model"
	"github.com/determined-ai/determined/master/pkg/searcher"
	"github.com/determined-ai/determined/master/pkg/tasks"
	"github.com/determined-ai/determined/master/pkg/workload"
	"github.com/determined-ai/determined/proto/pkg/apiv1"
)

// Experiment-specific actor messages.
type (
	trialCreated struct {
		create searcher.Create
		trialSnapshot
	}
	trialReportValidation struct {
		metric float64
		trialSnapshot
	}
	trialReportEarlyExit struct {
		reason workload.ExitedReason
		trialSnapshot
	}
	trialReportProgress struct {
		requestID model.RequestID
		progress  model.PartialUnits
	}
	trialQueryIsBestValidation struct {
		validationMetrics workload.ValidationMetrics
	}

	// Searcher-related messages.
	trialTrainUntilReq struct {
		trialID int
	}
	trialTrainUntilResp struct {
		finished bool
		length   model.Length
	}

	// trialClosed is used to replay closes missed when the master dies between when a trial closing in
	// its actor.PostStop and when the experiment snapshots the trial closed.
	trialClosed struct {
		requestID model.RequestID
	}
	// TODO(brad): This message is redundant.
	getTrial       struct{ trialID int }
	killExperiment struct{}
)

type trialSnapshot struct {
	requestID model.RequestID
	trialID   int
	snapshot  []byte
}

type trialSnapshotCarrier interface {
	getSnapshot() trialSnapshot
}

func (t trialSnapshot) getSnapshot() trialSnapshot {
	return t
}

type (
	experimentState struct {
		SearcherState  json.RawMessage `json:"searcher_state"`
		BestValidation *float64        `json:"best_validation"`
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

		agentUserGroup *model.AgentUserGroup
		taskSpec       *tasks.TaskSpec

		TrialCurrentOperation map[model.RequestID]searcher.ValidateAfter

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
	conf := expModel.Config

	// Validate the ResourcePool setting.  The reason to do it now and not in postExperiment like
	// all the other validations is that the resource pool should be revalidated every time the
	// master restarts.
	if err := sproto.ValidateRP(master.system, conf.Resources.ResourcePool); err != nil {
		return nil, err
	}
	// If the resource pool isn't set, fill in the default.
	if expModel.Config.Resources.ResourcePool == "" {
		if expModel.Config.Resources.SlotsPerTrial == 0 {
			expModel.Config.Resources.ResourcePool = sproto.GetDefaultCPUResourcePool(master.system)
		} else {
			expModel.Config.Resources.ResourcePool = sproto.GetDefaultGPUResourcePool(master.system)
		}
	}

	method := searcher.NewSearchMethod(conf.Searcher)
	search := searcher.NewSearcher(conf.Reproducibility.ExperimentSeed, method, conf.Hyperparameters)

	// Call InitialOperations which adds operations to the record in the Searcher. These
	// will be sent back to their respective trials in experiment prestart. This allows them to
	// be discarded if we Restore from a snapshot (since they will already exist in the snapshot
	// and have been accounted for).
	if _, err := search.InitialOperations(); err != nil {
		return nil, errors.Wrap(err, "failed to generate initial operations")
	}

	// Retrieve the warm start checkpoint, if provided.
	checkpoint, err := checkpointFromTrialIDOrUUID(
		master.db, conf.Searcher.SourceTrialID, conf.Searcher.SourceCheckpointUUID)
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

	return &experiment{
		Experiment:          expModel,
		modelDefinition:     modelDefinition,
		rm:                  master.rm,
		trialLogger:         master.trialLogger,
		hpImportance:        master.hpImportance,
		db:                  master.db,
		searcher:            search,
		warmStartCheckpoint: checkpoint,

		agentUserGroup: agentUserGroup,
		taskSpec:       taskSpec,

		TrialCurrentOperation: map[model.RequestID]searcher.ValidateAfter{},

		faultToleranceEnabled: true,
	}, nil
}

func (e *experiment) Receive(ctx *actor.Context) error {
	if msg, ok := ctx.Message().(trialSnapshotCarrier); ok && e.faultToleranceEnabled {
		defer e.snapshotAndSave(ctx, msg.getSnapshot())
	}
	switch msg := ctx.Message().(type) {
	// Searcher-related messages.
	case actor.PreStart:
		telemetry.ReportExperimentCreated(ctx.Self().System(), *e.Experiment)

		ctx.Tell(e.rm, sproto.SetGroupMaxSlots{
			MaxSlots: e.Config.Resources.MaxSlots,
			Handler:  ctx.Self(),
		})
		ctx.Tell(e.rm, sproto.SetGroupWeight{Weight: e.Config.Resources.Weight, Handler: ctx.Self()})
		ctx.Tell(e.rm, sproto.SetGroupPriority{
			Priority: e.Config.Resources.Priority,
			Handler:  ctx.Self(),
		})

		if e.restored {
			e.restoreTrialsFromPriorOperations(ctx, e.searcher.TrialOperations)
		} else {
			e.processOperations(ctx, e.searcher.TrialOperations, nil)
			ctx.Tell(e.hpImportance, hpimportance.ExperimentCreated{ID: e.ID})
		}
		// Since e.searcher.TrialOperations should have all trials that were previously
		// allocated, we can stop trying to restore new trials after processing these.
		e.restored = false
	case trialCreated:
		ops, err := e.searcher.TrialCreated(msg.create, msg.trialID)
		e.processOperations(ctx, ops, err)
	case trialReportValidation:
		ops, err := e.searcher.ValidationCompleted(msg.trialID, msg.metric)
		e.processOperations(ctx, ops, err)
		if ctx.ExpectingResponse() {
			ctx.Respond(nil)
		}
	case trialReportEarlyExit:
		ops, err := e.searcher.TrialExitedEarly(msg.trialID, msg.reason)
		e.processOperations(ctx, ops, err)
	case trialQueryIsBestValidation:
		ctx.Respond(e.isBestValidation(msg.validationMetrics))
	case trialReportProgress:
		e.searcher.SetTrialProgress(msg.requestID, msg.progress)
		progress := e.searcher.Progress()
		if err := e.db.SaveExperimentProgress(e.ID, &progress); err != nil {
			ctx.Log().WithError(err).Error("failed to save experiment progress")
		}
		ctx.Tell(e.hpImportance, hpimportance.ExperimentProgress{ID: e.ID, Progress: progress})
		if ctx.ExpectingResponse() {
			ctx.Respond(nil)
		}
	case trialTrainUntilReq:
		requestID, ok := e.searcher.RequestID(msg.trialID)
		if !ok {
			ctx.Respond(errors.New("trial not found"))
		}
		if op, ok := e.TrialCurrentOperation[requestID]; ok {
			ctx.Respond(trialTrainUntilResp{
				finished: false,
				length:   op.Length,
			})
		} else {
			ctx.Respond(trialTrainUntilResp{
				finished: true,
			})
		}
	case sendNextWorkload:
		// Pass this back to the trial; this message is just used to allow the trial to synchronize
		// with the searcher.
		ctx.Tell(ctx.Sender(), msg)
	case actor.ChildFailed:
		ctx.Log().WithError(msg.Error).Error("trial failed unexpectedly")
		e.trialClosed(ctx, model.MustParseRequestID(msg.Child.Address().Local()))
	case actor.ChildStopped:
		e.trialClosed(ctx, model.MustParseRequestID(msg.Child.Address().Local()))
	case trialClosed:
		e.trialClosed(ctx, msg.requestID)

	case getTrial:
		requestID, ok := e.searcher.RequestID(msg.trialID)
		ref := ctx.Child(requestID)
		if ok && ref != nil {
			ctx.Respond(ref)
		}

	// Patch experiment messages.
	case model.State:
		e.updateState(ctx, msg)
	case sproto.SetGroupMaxSlots:
		e.Config.Resources.MaxSlots = msg.MaxSlots
		msg.Handler = ctx.Self()
		ctx.Tell(e.rm, msg)
	case sproto.SetGroupWeight:
		e.Config.Resources.Weight = msg.Weight
		msg.Handler = ctx.Self()
		ctx.Tell(e.rm, msg)

	case killExperiment:
		if _, running := model.RunningStates[e.State]; running {
			e.updateState(ctx, model.StoppingCanceledState)
		}

		for _, child := range ctx.Children() {
			ctx.Tell(child, killTrial{})
		}

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
		ctx.Self().System().ActorOf(addr, &checkpointGCTask{
			agentUserGroup: e.agentUserGroup,
			taskSpec:       e.taskSpec,
			rm:             e.rm,
			db:             e.db,
			experiment:     e.Experiment,
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
			switch ok := e.updateState(ctx, model.StoppingCanceledState); ok {
			case true:
				ctx.Respond(&apiv1.CancelExperimentResponse{})
				for _, child := range ctx.Children() {
					ctx.Tell(child, killTrial{})
				}
			default:
				ctx.Respond(status.Errorf(codes.FailedPrecondition,
					"experiment in incompatible state %s", e.State))
			}
		}

	case *apiv1.KillExperimentRequest:
		switch {
		case model.StoppingStates[e.State] || model.TerminalStates[e.State]:
			ctx.Respond(&apiv1.KillExperimentResponse{})
		default:
			switch ok := e.updateState(ctx, model.StoppingCanceledState); ok {
			case true:
				ctx.Respond(&apiv1.KillExperimentResponse{})
				for _, child := range ctx.Children() {
					ctx.Tell(child, killTrial{})
				}
			default:
				ctx.Respond(status.Errorf(codes.FailedPrecondition,
					"experiment in incompatible state %s", e.State))
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

// restoreTrialsFromPriorOperations from the operations that were snapshotted with the
// last experiment checkpoint.
func (e *experiment) restoreTrialsFromPriorOperations(
	ctx *actor.Context, ops []searcher.Operation,
) {
	// Previous implementations had a nice property that: since trials were restored in the order
	// they were requested, the trial running on failure was the first restarted. Using this ordered
	// list keeps that property.
	var requestIDs []model.RequestID
	trialOpsByRequestID := make(map[model.RequestID][]searcher.Operation)
	for _, op := range ops {
		ctx.Log().Debugf("restoring searcher op: %v", op)
		switch op := op.(type) {
		case searcher.Create:
			requestIDs = append(requestIDs, op.RequestID)
			trialOpsByRequestID[op.RequestID] = append(trialOpsByRequestID[op.RequestID], op)
		case searcher.ValidateAfter:
			trialOpsByRequestID[op.GetRequestID()] = append(trialOpsByRequestID[op.GetRequestID()], op)
			e.TrialCurrentOperation[op.GetRequestID()] = op
		case searcher.Close:
			trialOpsByRequestID[op.GetRequestID()] = append(trialOpsByRequestID[op.GetRequestID()], op)
			delete(e.TrialCurrentOperation, op.GetRequestID())
		}
	}

	for _, requestID := range requestIDs {
		ops := trialOpsByRequestID[requestID]
		op, ok := ops[0].(searcher.Create)
		if !ok {
			panic(fmt.Sprintf("encountered trial without a create: %s", requestID))
		}
		checkpoint, err := e.checkpointForCreate(op)
		if err != nil {
			e.updateState(ctx, model.StoppingErrorState)
			ctx.Log().Error(err)
			return
		}
		terminal := e.restoreTrial(ctx, op, checkpoint, ops[1:])
		// In the event a trial is terminal and is not recorded in the searcher, replay the close.
		if terminal && !e.searcher.TrialsClosed[op.RequestID] {
			ctx.Tell(ctx.Self(), trialClosed{requestID: op.RequestID})
		}
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

	trialOperations := make(map[model.RequestID][]searcher.Operation)
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
			ctx.ActorOf(op.RequestID, newTrial(e, op, checkpoint))
		case searcher.ValidateAfter:
			trialOperations[op.GetRequestID()] = append(trialOperations[op.GetRequestID()], op)
			e.TrialCurrentOperation[op.GetRequestID()] = op
		case searcher.Close:
			trialOperations[op.GetRequestID()] = append(trialOperations[op.GetRequestID()], op)
			delete(e.TrialCurrentOperation, op.GetRequestID())
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
	for requestID, ops := range trialOperations {
		ctx.Tell(ctx.Child(requestID), ops)
	}
}

func (e *experiment) checkpointForCreate(op searcher.Create) (*model.Checkpoint, error) {
	checkpoint := e.warmStartCheckpoint
	// If the Create specifies a checkpoint, ignore the experiment-wide one.
	if op.Checkpoint != nil {
		trialID, ok := e.searcher.TrialID(op.Checkpoint.RequestID)
		if !ok {
			return nil, errors.Errorf(
				"invalid request ID in Create operation: %d", op.Checkpoint.RequestID)
		}
		checkpointModel, err := checkpointFromTrialIDOrUUID(e.db, &trialID, nil)
		if err != nil {
			return nil, errors.Wrap(err, "checkpoint not found")
		}
		checkpoint = checkpointModel
	}
	return checkpoint, nil
}

func (e *experiment) isBestValidation(metrics workload.ValidationMetrics) bool {
	metricName := e.Config.Searcher.Metric
	validation, err := metrics.Metric(metricName)
	if err != nil {
		// TODO: Better error handling here.
		return false
	}
	smallerIsBetter := e.Config.Searcher.SmallerIsBetter
	isBest := (e.BestValidation == nil) ||
		(smallerIsBetter && validation <= *e.BestValidation) ||
		(!smallerIsBetter && validation >= *e.BestValidation)
	if isBest {
		e.BestValidation = &validation
	}
	return isBest
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
	for _, child := range ctx.Children() {
		ctx.Tell(child, state)
	}
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
