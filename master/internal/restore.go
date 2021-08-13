package internal

import (
	"encoding/json"
	"fmt"

	"github.com/pkg/errors"
	log "github.com/sirupsen/logrus"

	"github.com/determined-ai/determined/master/internal/db"
	"github.com/determined-ai/determined/master/internal/sproto"
	"github.com/determined-ai/determined/master/internal/telemetry"
	"github.com/determined-ai/determined/master/pkg/actor"
	"github.com/determined-ai/determined/master/pkg/model"
	"github.com/determined-ai/determined/master/pkg/schemas"
	"github.com/determined-ai/determined/master/pkg/schemas/expconf"
	"github.com/determined-ai/determined/master/pkg/searcher"
)

// The current experiment snapshot version. Once this is incremented, older versions should be
// shimmed. Experiment and trial snapshots share a version currently.
const experimentSnapshotVersion = 4

// Restore works by restoring from distributed consistent snapshots taken through the course
// of an experiment. Snapshots within the system flow from the bottom up, starting with the
// trial workload sequencer, to the trial, and finally to the experiment. Any event that the
// trial or trial workload sequencer processes that would trigger a change to the state of the
// experiment is:
//   1. Propagated atomically, within a single message, to ensure the experiment handles it all
//      or nothing
//   2. With a snapshot affixed to it, to mark that it should trigger a snapshot
// Upon receipt, the experiment handles the event entirely, snapshots its state and saves it
// along with all the snapshots it has received, atomically.
//
// This is technically equivalent to the Chandy-Lamport distributed snapshotting algorithm where
// the trial workload sequencer initiates the snapshots and propagates them along with trialCreated
// and trialCompletedWorkload messages, with the snapshot itself acting as the marker. Each actor we
// care about and know has updated state in the system snapshots it state when it receives this
// marker and propagates it through the system. Even though all trials are part of each distributed
// snapshot, we do not propagate snapshots to trials that didn't initiate the snapshot; that a trial
// sends a snapshot with each state change implies if it did not send the snapshot, it does not have
// a state change [1]. We can also be sure there are no pre-snapshot messages floating in the ether,
// since any message we care about snapshotting is itself a snapshot.
//
// To restore properly, any message the experiment would send as a result of snapshot-able event
// must be replayed on restore. In our case, these are the new operations created by the searcher
// as a result a trialCreated or trialCompletedWorkload message. We restore trials and experiment
// directly from their snapshots and tell the trial all its operations on creation.
//
// [1] This is fine, but _also_ we have to make sure that nothing else alters trial and experiment
// state that doesn't flow from trial to experiment; experiments can never push non-ephemeral state
// updates to trials without special consideration. searcher.Operations are an example of this (and
// the experiment snapshots them and re-sends them).
func (m *Master) restoreExperiment(expModel *model.Experiment) error {
	// Experiments which were trying to stop need to be marked as terminal in the database.
	if terminal, ok := model.StoppingToTerminalStates[expModel.State]; ok {
		if err := m.db.TerminateExperimentInRestart(expModel.ID, terminal); err != nil {
			return errors.Wrapf(err, "terminating experiment %d", expModel.ID)
		}
		expModel.State = terminal
		telemetry.ReportExperimentStateChanged(m.system, m.db, *expModel)
		return nil
	} else if _, ok := model.RunningStates[expModel.State]; !ok {
		return errors.Errorf(
			"cannot restore experiment %d from state %v", expModel.ID, expModel.State,
		)
	} else if err := expModel.Config.Searcher().AssertCurrent(); err != nil {
		return errors.Errorf(
			"cannot restore experiment %d with legacy searcher", expModel.ID,
		)
	}

	poolName, err := sproto.GetResourcePool(
		m.system,
		expModel.Config.Resources().ResourcePool(),
		expModel.Config.Resources().SlotsPerTrial(),
		false,
	)
	if err != nil {
		return errors.Wrap(err, "invalid resource configuration")
	}

	taskContainerDefaults := m.getTaskContainerDefaults(poolName)
	taskSpec := *m.taskSpec
	taskSpec.TaskContainerDefaults = taskContainerDefaults

	log.WithField("experiment", expModel.ID).Debug("restoring experiment")
	snapshot, err := m.retrieveExperimentSnapshot(expModel)
	if err != nil {
		return errors.Wrapf(err, "failed to restore experiment %d", expModel.ID)
	}
	e, err := newExperiment(m, expModel, &taskSpec)
	if err != nil {
		return errors.Wrapf(err, "failed to create experiment %d from model", expModel.ID)
	}
	if snapshot != nil {
		if err := e.Restore(snapshot); err != nil {
			return errors.Wrap(err, "failed to restore experiment")
		}
		e.restored = true
	}

	m.system.ActorOf(actor.Addr("experiments", e.ID), e)
	return nil
}

// restoreTrial takes the a searcher.Create and attempts to restore the trial that would be
// associated with it. On failure, the trial is just reset to the start and errors are logged.
func (e *experiment) restoreTrial(
	ctx *actor.Context, ckpt *model.Checkpoint, searcher trialSearcherState,
) {
	l := ctx.Log().WithField("request-id", searcher.Create.RequestID)
	l.Debug("restoring trial")

	var trialID *int
	var terminal bool
	switch trial, err := e.db.TrialByExperimentAndRequestID(e.ID, searcher.Create.RequestID); {
	case errors.Cause(err) == db.ErrNotFound:
		l.Debug("trial was never previously allocated")
	case err != nil:
		// This is the only place we _have_ to error, because if the trial did previously exist
		// and we failed to retrieve it, continuing will result in an invalid state (we'll get a
		// new trial in the trials table with the same (experiment_id, request_id).
		l.WithError(err).Error("failed to retrieve trial, aborting restore")
		terminal = true
	default:
		trialID = &trial.ID
		l = l.WithField("trial-id", trial.ID)
		if model.TerminalStates[trial.State] {
			l.Debugf("trial was in terminal state in restore: %s", trial.State)
			terminal = true
		} else if !model.RunningStates[trial.State] {
			l.Debugf("cannot restore trial in state: %s", trial.State)
			terminal = true
		}
	}

	// In the event a trial is terminal and is not recorded in the searcher, replay the close.
	if terminal {
		if !e.searcher.TrialsClosed[searcher.Create.RequestID] {
			ctx.Tell(ctx.Self(), trialClosed{requestID: searcher.Create.RequestID})
		}
		return
	}

	config := schemas.Copy(e.Config).(expconf.ExperimentConfig)
	t := newTrial(
		trialTaskID(e.ID, searcher.Create.RequestID), e.ID, e.State, searcher, e.rm,
		e.trialLogger, e.db, config, ckpt, e.taskSpec, e.modelDefinition,
	)
	if trialID != nil {
		t.id = *trialID
		t.idSet = true
		if _, ok := e.searcher.TrialsCreated[searcher.Create.RequestID]; !ok {
			ctx.Tell(ctx.Self(), trialCreated{
				requestID: searcher.Create.RequestID,
			})
		}
	}
	ctx.ActorOf(searcher.Create.RequestID, t)
	l.Debug("restored trial")
}

// retrieveExperimentSnapshot retrieves a snapshot in from database if it exists.
func (m *Master) retrieveExperimentSnapshot(expModel *model.Experiment) ([]byte, error) {
	switch snapshot, version, err := m.db.ExperimentSnapshot(expModel.ID); {
	case snapshot == nil:
		log.WithField("experiment-id", expModel.ID).Debug("no snapshot found")
		return nil, nil
	case err != nil:
		return nil, errors.Wrap(err, "failed to retrieve experiment snapshot")
	default:
		if snapshot, err = shimExperimentSnapshot(snapshot, version); err != nil {
			return nil, errors.Wrap(err, "failed to shim trial snapshot")
		}
		return snapshot, nil
	}
}

func (e *experiment) snapshotAndSave(ctx *actor.Context) {
	es, err := e.Snapshot()
	if err != nil {
		e.faultToleranceEnabled = false
		ctx.Log().WithError(err).Errorf("failed to snapshot experiment, fault tolerance is lost")
		return
	}
	err = e.db.SaveSnapshot(e.ID, experimentSnapshotVersion, es)
	if err != nil {
		e.faultToleranceEnabled = false
		ctx.Log().WithError(err).Errorf("failed to persist experiment snapshot, fault tolerance is lost")
		return
	}
}

// experimentSnapshotShims maps a version to the shim that bumps that version.
var experimentSnapshotShims = map[int]snapshotShimFunc{
	0: shimExperimentSnapshotV0,
	1: shimExperimentSnapshotV1,
	2: shimExperimentSnapshotV2,
}

// shimExperimentSnapshot shims a trial snapshot to the version required by the master,
// returning an error in the event the shim fails or the snapshot version is greater
// than the current version (which could happen in a downgrade).
func shimExperimentSnapshot(snapshot []byte, version int) ([]byte, error) {
	return shimSnapshot(experimentSnapshotShims, snapshot, version)
}

func shimSnapshot(shims map[int]snapshotShimFunc, snapshot []byte, version int) ([]byte, error) {
	if version > experimentSnapshotVersion {
		return nil, fmt.Errorf("cannot shim from %d to %d", experimentSnapshotVersion, version)
	}
	var err error
	for version < experimentSnapshotVersion {
		shim, ok := shims[version]
		if !ok {
			return nil, fmt.Errorf("missing shim from %d to %d", experimentSnapshotVersion, version)
		}
		if snapshot, err = shim(snapshot); err != nil {
			return nil, errors.Wrapf(err, "failed to shim snapshot")
		}
		version++
	}
	return snapshot, nil
}

// snapshotShimFunc is a shimming function.
type snapshotShimFunc func([]byte) ([]byte, error)

// Version 0 => 1 shims

// shimExperimentSnapshotV0 shims a v0 experiment snapshot to a v1 experiment snapshot.
// From v0 to v1, the searcher checkpoint operations were removed. Because of this, all checkpoint
// operations are removed from the operations requested of trials and any PBT operations
// which are awaiting a particular checkpoint to finish added to the queue of trial operations,
// since by other invariants in the system we can guarantee this checkpoint exists.
func shimExperimentSnapshotV0(snapshot []byte) ([]byte, error) {
	var experimentSnapshotV0 map[string]interface{}
	if err := json.Unmarshal(snapshot, &experimentSnapshotV0); err != nil {
		return nil, err
	}

	searcherState := experimentSnapshotV0["searcher_state"].(map[string]interface{})
	if searcherState["search_method_state"] == nil {
		return snapshot, nil
	}
	searchMethodState := searcherState["search_method_state"].(map[string]interface{})
	// Get the waiting operations from PBT.
	waitingCheckpoints, ok := searchMethodState["waiting_checkpoints"]
	if !ok {
		// If `waiting_checkpoints` is missing, this isn't PBT and this shim is only to shim PBT.
		return snapshot, nil
	}

	// And queue them to be sent to trials immediately.
	for _, ops := range waitingCheckpoints.(map[string]interface{}) {
		searcherState["trial_operations"] = append(searcherState["trial_operations"].([]interface{}),
			ops.([]interface{})...)
	}
	delete(searchMethodState, "waiting_checkpoints")

	// Then filter out any checkpoints PBT instructed the trial to run.
	var filteredOps []interface{}
	for _, op := range searcherState["trial_operations"].([]interface{}) {
		// Remove checkpoints, that used to be OperationType = 3.
		// Any numeric types parsed from JSON into an interface{} will be a float64.
		if op.(map[string]interface{})["OperationType"].(float64) == 3 {
			continue
		}
		filteredOps = append(filteredOps, op)
	}
	searcherState["trial_operations"] = filteredOps
	return json.Marshal(experimentSnapshotV0)
}

// Version 1 => 2 shims

// shimExperimentSnapshotV1 shims a v1 snapshot to a v2 snapshot. From v1 to v2,
// progress was rewritten so that individual trial progress was tracked and aggregated
// instead of just maintaining the total units completed.
func shimExperimentSnapshotV1(snapshot []byte) ([]byte, error) {
	var experimentSnapshotV1 map[string]interface{}
	if err := json.Unmarshal(snapshot, &experimentSnapshotV1); err != nil {
		return nil, err
	}

	searcherState := experimentSnapshotV1["searcher_state"].(map[string]interface{})

	delete(searcherState, "total_units_completed")
	searcherState["trial_progress"] = searcherState["units_completed_by_trial"]
	delete(searcherState, "units_completed_by_trial")

	return json.Marshal(experimentSnapshotV1)
}

// Version 2 => 3 shims

// shimExperimentSnapshotV2 shims a v2 snapshot to a v3 snapshot. From v2 to v3,
// Train and Validate operations were merged into a single ValidateAfter operation
// that indicates to the trial the total units to train before reporting a validation
// to the searcher.
func shimExperimentSnapshotV2(snapshot []byte) ([]byte, error) {
	var experimentSnapshotV2 map[string]interface{}
	if err := json.Unmarshal(snapshot, &experimentSnapshotV2); err != nil {
		return nil, err
	}

	searcherState := experimentSnapshotV2["searcher_state"].(map[string]interface{})
	operationsList := searcherState["trial_operations"].([]interface{})

	totalUnitsForTrial := map[string]float64{} // string is model.RequestID
	var newOperationsList []map[string]interface{}
	for _, iOp := range operationsList {
		op := iOp.(map[string]interface{})
		switch searcher.OperationType(op["OperationType"].(float64)) {
		case searcher.TrainOperation:
			op := op["Operation"].(map[string]interface{})
			requestID := op["RequestID"].(string)
			length := op["Length"].(map[string]interface{})
			for unit, units := range length {
				totalUnitsForTrial[requestID] += units.(float64)
				newOperationsList = append(newOperationsList, map[string]interface{}{
					"OperationType": searcher.ValidateAfterOperation,
					"Operation": map[string]interface{}{
						"RequestID": requestID,
						"Length": map[string]interface{}{
							unit: totalUnitsForTrial[requestID],
						},
					},
				})
			}
		case searcher.ValidateOperation:
			continue
		default:
			newOperationsList = append(newOperationsList, op)
		}
	}

	searcherState["trial_operations"] = newOperationsList

	return json.Marshal(experimentSnapshotV2)
}
