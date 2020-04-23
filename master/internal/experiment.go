package internal

import (
	"encoding/json"
	"fmt"

	"github.com/pkg/errors"
	log "github.com/sirupsen/logrus"

	"github.com/determined-ai/determined/master/internal/db"
	"github.com/determined-ai/determined/master/internal/scheduler"
	"github.com/determined-ai/determined/master/internal/telemetry"
	"github.com/determined-ai/determined/master/pkg/actor"
	"github.com/determined-ai/determined/master/pkg/archive"
	"github.com/determined-ai/determined/master/pkg/model"
	"github.com/determined-ai/determined/master/pkg/searcher"
)

// Experiment-specific actor messages.
type (
	trialCreated struct {
		create  searcher.Create
		trialID int
	}
	getProgress    struct{}
	getTrial       struct{ trialID int }
	restoreTrials  struct{}
	trialsRestored struct{}
	killExperiment struct{}

	// doneProcessingSearcherOperations message is only used during master restart, to ensure that
	// all the searcher operations created by a given event (experiment created / trial created /
	// workload completed) are fully handled before passing another event to the actor system. This
	// ensures we do not pass a workload completed event to a trial which either a) does not exist
	// yet, or b) has not yet seen that workload request.
	//
	// TODO(ryan): Rework the trial/experiment interface to remove the need for this level of
	// synchronization as part of DET-675, which would put the WorkloadSequencer alongside the
	// SearchMethod in the experiment actor instead of the trial actor. With that change, all of
	// the restorable state would be in a single actor and the complex replay synchronization would
	// be eliminated.
	doneProcessingSearcherOperations struct{}
)

const (
	// TrialCreatedEventType is the event type in the database for a searcher.TrialCreatedEvent.
	TrialCreatedEventType = "TrialCreated"
	// TrialClosedEventType is the event type in the database for a searcher.TrialClosedEvent.
	TrialClosedEventType = "TrialClosed"
	// WorkloadCompletedEventType is the event type in the database for a workload.CompletedMessage.
	WorkloadCompletedEventType = "WorkloadCompleted"

	// searcherEventBuffer is the maximum number of SearcherEvents that can be buffered before
	// writing to the database.  In reality, it is much more likely flushing the buffer happens
	// due to the contents of the SearcherEvents than the number of them; see the comment in
	// convertSearcherEvent()
	searcherEventBuffer = 1000
)

type experiment struct {
	*model.Experiment
	modelDefinition     archive.Archive
	cluster             *actor.Ref
	trialLogger         *actor.Ref
	db                  *db.PgDB
	searcher            *searcher.Searcher
	warmStartCheckpoint *model.Checkpoint
	bestValidation      *float64
	replaying           bool

	pendingEvents []*model.SearcherEvent

	agentUserGroup *model.AgentUserGroup
}

// Create a new experiment object from the given model experiment object, along with its searcher
// and log. If the input object has no ID set, also create a new experiment in the database and set
// the returned object's ID appropriately.
func newExperiment(master *Master, expModel *model.Experiment) (*experiment, error) {
	conf := expModel.Config
	method := searcher.NewSearchMethod(conf.Searcher)
	search := searcher.NewSearcher(
		conf.Reproducibility.ExperimentSeed, method, conf.Hyperparameters)

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
		cluster:             master.cluster,
		trialLogger:         master.trialLogger,
		db:                  master.db,
		searcher:            search,
		warmStartCheckpoint: checkpoint,
		pendingEvents:       make([]*model.SearcherEvent, 0, searcherEventBuffer),

		agentUserGroup: agentUserGroup,
	}, nil
}

// marshalInto marshals a generic JSON object into the content of obj.
func marshalInto(unmarshaled interface{}, obj interface{}) error {
	bytes, err := json.Marshal(unmarshaled)
	if err != nil {
		return errors.Wrapf(err, "failed to marshal from %T", unmarshaled)
	}
	if err = json.Unmarshal(bytes, obj); err != nil {
		return errors.Wrapf(err, "failed to unmarshal into %T", obj)
	}
	return nil
}

// newSearcherEventCallback returns a closure replays SearcherEvents to restore in-progress
// experiments during Master restart. The SearcherEvent log can become tens of GB for a large
// experiment when loaded into memory, and this lets us avoid asking the database to pass us all
// the rows at once.
func newSearcherEventCallback(master *Master, ref *actor.Ref) func(model.SearcherEvent) error {
	requestIDs := make(map[int]searcher.RequestID)

	return func(event model.SearcherEvent) error {
		switch event.EventType {
		case TrialCreatedEventType:
			log.Debugf("\x1b[32mrestore: trial created\x1b[m %v %v",
				event.Content["request_id"], event.Content["trial_id"])

			// Convert the JSON representation of the create operation into an actual operation object.
			obj := event.Content["operation"].(map[string]interface{})["Create"]
			create := searcher.Create{}
			if err := marshalInto(obj, &create); err != nil {
				return errors.Wrap(err, "failed to process create operation")
			}

			trialID := int(event.Content["trial_id"].(float64))
			requestIDs[trialID] = create.RequestID

			// We pass the TrialCreated event to the trial so that it knows its TrialID from
			// before, and the trial will pass the TrialCreated to the experiment before we get a
			// response from this Ask.
			master.system.AskAt(ref.Address().Child(create.RequestID),
				trialCreated{create: create, trialID: trialID}).Get()

			// Wait for the experiment to handle any searcher operations due to the created trial.
			master.system.Ask(ref, doneProcessingSearcherOperations{}).Get()

		case WorkloadCompletedEventType:
			{
				w := event.Content["msg"].(map[string]interface{})["workload"].(map[string]interface{})
				log.Debugf("\x1b[32mrestore workload\x1b[m: %d %v %v %s",
					event.ID, w["trial_id"], w["step_id"], w["kind"])
			}
			// Convert the JSON representation of the message to an actual message object.
			obj := event.Content["msg"]
			var msg searcher.CompletedMessage
			if err := marshalInto(obj, &msg); err != nil {
				return errors.Wrap(err, "failed to process completed message")
			}

			// Pass the workload completed message to the Trial. It will pass the event along to
			// the experiment before this Ask gets a response.
			master.system.AskAt(ref.Address().Child(requestIDs[msg.Workload.TrialID]), msg).Get()

			// Wait for the experiment to handle any searcher operations due to the completed
			// workload.
			master.system.Ask(ref, doneProcessingSearcherOperations{}).Get()

		case TrialClosedEventType:
			// Ignore these events; the trial actors' closing will notify the experiment naturally.
		}
		return nil
	}
}

func restoreExperiment(master *Master, expModel *model.Experiment) error {
	// Experiments which were trying to stop need to be marked as terminal in the database.
	if terminal, ok := model.StoppingToTerminalStates[expModel.State]; ok {
		if err := master.db.TerminateExperimentInRestart(expModel.ID, terminal); err != nil {
			return errors.Wrapf(err, "terminating experiment %d", expModel.ID)
		}
		expModel.State = terminal
		telemetry.ReportExperimentStateChanged(master.system, master.db, *expModel)
		return nil
	} else if _, ok := model.RunningStates[expModel.State]; !ok {
		return errors.Errorf(
			"cannot restore experiment %d from state %v", expModel.ID, expModel.State,
		)
	}

	e, err := newExperiment(master, expModel)
	if err != nil {
		return errors.Wrapf(err, "failed to create experiment %d from model", expModel.ID)
	}

	log := log.WithField("experiment", e.ID)

	log.Info("restoring experiment")
	e.replaying = true

	ref, _ := master.system.ActorOf(actor.Addr("experiments", e.ID), e)

	// Wait for the experiment to handle any initial searcher operations.
	master.system.Ask(ref, doneProcessingSearcherOperations{}).Get()

	if err = e.db.ForEachSearcherEvent(e.ID, newSearcherEventCallback(master, ref)); err != nil {
		return errors.Wrapf(err, "failed to get searcher events")
	}

	// We have the experiment ask all the trials to restore (since we don't know all of the trial
	// actor children) and wait here for them to finish. Since the trials might ask things of the
	// experiment while restoring, we can't have the experiment itself wait for the trials.
	trialResponses := master.system.Ask(ref, restoreTrials{}).Get()

	// If the experiment failed during the replay we may receive a nil response.
	if trialResponses == nil {
		return errors.Errorf("experiment %v did not respond to 'restoreTrials' message", e.ID)
	}

	for range trialResponses.(actor.Responses) {
	}

	// Now notify the experiment that the trials are done and wait for a response, so that this
	// function doesn't exit before the experiment and trials are fully caught up.
	master.system.Ask(ref, trialsRestored{}).Get()

	return nil
}

func (e *experiment) Receive(ctx *actor.Context) error {
	switch msg := ctx.Message().(type) {
	// Searcher-related messages.
	case actor.PreStart:
		telemetry.ReportExperimentCreated(ctx.Self().System(), *e.Experiment)

		ctx.Tell(e.cluster, scheduler.SetMaxSlots{MaxSlots: e.Config.Resources.MaxSlots})
		ctx.Tell(e.cluster, scheduler.SetWeight{Weight: e.Config.Resources.Weight})
		ops, err := e.searcher.InitialOperations()
		e.processOperations(ctx, ops, err)
	case trialCreated:
		ops, err := e.searcher.TrialCreated(msg.create, msg.trialID)
		e.processOperations(ctx, ops, err)
	case searcher.CompletedMessage:
		ops, err := e.searcher.WorkloadCompleted(msg)
		e.processOperations(ctx, ops, err)
		if msg.Workload.Kind == searcher.ComputeValidationMetrics {
			ctx.Respond(e.isBestValidation(msg))
		}
		progress := e.searcher.Progress()
		if err = e.db.SaveExperimentProgress(e.ID, &progress); err != nil {
			ctx.Log().WithError(err).Error("failed to save experiment progress")
		}
	case actor.ChildFailed:
		ctx.Log().WithError(msg.Error).Error("trial failed unexpectedly")
		e.updateState(ctx, model.StoppingErrorState)
		if e.canTerminate(ctx) {
			ctx.Self().Stop()
		}
	case actor.ChildStopped:
		requestID := searcher.MustParse(msg.Child.Address().Local())
		ops, err := e.searcher.TrialClosed(requestID)
		e.processOperations(ctx, ops, err)
		if e.canTerminate(ctx) {
			ctx.Self().Stop()
		}
	case getProgress:
		progress := e.searcher.Progress()
		ctx.Respond(&progress)

	case getTrial:
		requestID, ok := e.searcher.RequestID(msg.trialID)
		ref := ctx.Child(requestID)
		if ok && ref != nil {
			ctx.Respond(ref)
		}

	// Restoration-related messages.
	case doneProcessingSearcherOperations:
		// This is just a synchronization tool for master restarts; the actor system's default
		// response is fine.
	case restoreTrials:
		ctx.Respond(ctx.AskAll(restoreTrial{}, ctx.Children()...))
	case trialsRestored:
		e.replaying = false

	// Patch experiment messages.
	case model.State:
		e.updateState(ctx, msg)
	case scheduler.SetMaxSlots:
		e.Config.Resources.MaxSlots = msg.MaxSlots
		ctx.Tell(e.cluster, msg)
	case scheduler.SetWeight:
		e.Config.Resources.Weight = msg.Weight
		ctx.Tell(e.cluster, msg)

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

		// Flush any remaining searcher logs
		if err := e.db.AddSearcherEvents(e.pendingEvents); err != nil {
			ctx.Log().Error(err)
			e.updateState(ctx, model.StoppingErrorState)
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
			cluster:        e.cluster,
			db:             e.db,
			experiment:     e.Experiment,
		})

		// Discard searcher events for all terminal experiments (even failed ones).
		// This is safe because we never try to restore the state of the searcher for
		// terminated experiments.
		if err := e.db.DeleteSearcherEvents(e.Experiment.ID); err != nil {
			ctx.Log().WithError(err).Errorf(
				"failure to delete searcher events for experiment: %d", e.Experiment.ID)
		}

		ctx.Log().Info("experiment shut down successfully")
	}
	return nil
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

	trialOperations := make(map[searcher.RequestID][]searcher.Operation)
	for _, operation := range ops {
		ctx.Log().Debugf("handling searcher op: %v", operation)
		switch op := operation.(type) {
		case searcher.Create:
			checkpoint := e.warmStartCheckpoint
			// If the Create specifies a checkpoint, ignore the experiment-wide one.
			if op.Checkpoint != nil {
				trialID, ok := e.searcher.TrialID(op.Checkpoint.RequestID)
				if !ok {
					ctx.Log().Error(errors.Errorf(
						"invalid request ID in Create operation: %d", op.Checkpoint.RequestID))
					e.updateState(ctx, model.StoppingErrorState)
					return
				}
				checkpointModel, err := checkpointFromTrialIDOrUUID(e.db, &trialID, nil)
				if err != nil {
					ctx.Log().Error(errors.Wrap(err, "checkpoint not found"))
					e.updateState(ctx, model.StoppingErrorState)
					return
				}
				checkpoint = checkpointModel
			}
			ctx.ActorOf(op.RequestID, newTrial(e, op, checkpoint))
		case searcher.WorkloadOperation:
			trialOperations[op.RequestID] = append(trialOperations[op.RequestID], op)
		case searcher.Close:
			trialOperations[op.RequestID] = append(trialOperations[op.RequestID], op)
		case searcher.Shutdown:
			e.updateState(ctx, model.StoppingCompletedState)
		default:
			panic(fmt.Sprintf("unexpected operation: %v", op))
		}
	}
	for requestID, ops := range trialOperations {
		ctx.Tell(ctx.Child(requestID), ops)
	}

	// Commit new searcher events to the database.
	events := e.searcher.UncommittedEvents()
	if !e.replaying {
		flushEvents := false
		for _, event := range events {
			modelEvent, flush, err := convertSearcherEvent(e.ID, event)
			if err != nil {
				ctx.Log().Error(err)
				e.updateState(ctx, model.StoppingErrorState)
				return
			}
			flushEvents = flushEvents || flush
			e.pendingEvents = append(e.pendingEvents, modelEvent)
		}
		// Flush events to the database if either we have enough to be efficient or if the most
		// recent event is important for the consistency of the searcher state and the database
		// state. See comment in convertSearcherEvent().
		//
		// TODO(ryan): This keeps the experiment actor's inbox much smaller under heavy loads,
		// which results in a much more performant system, since things like `det e list` or the
		// webui have to Ask() the experiment for its state. However, chunking like this may not
		// be strictly valid, which is non-ideal, but Searcher Reload (DET-816) is the "real" fix.
		if flushEvents || len(e.pendingEvents) > searcherEventBuffer {
			if err := e.db.AddSearcherEvents(e.pendingEvents); err != nil {
				ctx.Log().Error(err)
				e.updateState(ctx, model.StoppingErrorState)
				return
			}
			e.pendingEvents = e.pendingEvents[:0]
		}
	}
}

func (e *experiment) isBestValidation(msg searcher.CompletedMessage) bool {
	metricName := e.Config.Searcher.Metric
	validation, err := msg.ValidationMetrics.Metric(metricName)
	if err != nil {
		// TODO: Better error handling here.
		return false
	}
	if e.bestValidation == nil {
		e.bestValidation = &validation
		return true
	}
	smallerIsBetter := e.Config.Searcher.SmallerIsBetter
	if smallerIsBetter && validation < *e.bestValidation {
		e.bestValidation = &validation
		return true
	} else if validation > *e.bestValidation {
		e.bestValidation = &validation
		return true
	}
	return false
}

func (e *experiment) updateState(ctx *actor.Context, state model.State) {
	if wasPatched, err := e.Transition(state); err != nil {
		ctx.Log().Error(err)
		return
	} else if !wasPatched {
		return
	}
	telemetry.ReportExperimentStateChanged(ctx.Self().System(), e.db, *e.Experiment)

	ctx.Log().Infof("experiment state changed to %s", state)
	for _, child := range ctx.Children() {
		ctx.Tell(child, state)
	}
	if err := e.db.SaveExperimentState(e.Experiment); err != nil {
		ctx.Log().Error(err)
	}
	if e.canTerminate(ctx) {
		ctx.Self().Stop()
	}
}

func (e *experiment) canTerminate(ctx *actor.Context) bool {
	return model.StoppingStates[e.State] && len(ctx.Children()) == 0
}
