package internal

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"github.com/determined-ai/determined/master/internal/db"
	"github.com/determined-ai/determined/master/internal/experiment"
	"github.com/determined-ai/determined/master/internal/rm"
	"github.com/determined-ai/determined/master/internal/sproto"
	"github.com/determined-ai/determined/master/internal/user"
	"github.com/determined-ai/determined/master/internal/workspace"
	"github.com/determined-ai/determined/master/pkg/model"
	"github.com/determined-ai/determined/master/pkg/nprand"
	"github.com/determined-ai/determined/master/pkg/ptrs"
	"github.com/determined-ai/determined/master/pkg/schemas"
	"github.com/determined-ai/determined/master/pkg/searcher"
	"github.com/pkg/errors"
	log "github.com/sirupsen/logrus"
)

// The current experiment snapshot version. Once this is incremented, older versions should be
// shimmed. Experiment and trial snapshots share a version currently.
const experimentSnapshotVersion = 6

// Restore works by restoring from distributed consistent snapshots taken through the course
// of an experiment. Snapshots within the system flow from the bottom up, starting with the
// trial workload sequencer, to the trial, and finally to the experiment. Any event that the
// trial or trial workload sequencer processes that would trigger a change to the state of the
// experiment is:
//  1. Propagated atomically, within a single message, to ensure the experiment handles it all
//     or nothing
//  2. With a snapshot affixed to it, to mark that it should trigger a snapshot
//
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
	activeConfig, err := m.db.ActiveExperimentConfig(expModel.ID)
	if err != nil {
		return errors.Errorf("cannot restore experiment %d with unparsable config", expModel.ID)
	}

	if err := activeConfig.Searcher().AssertCurrent(); err != nil {
		return errors.Errorf(
			"cannot restore experiment %d with legacy searcher", expModel.ID,
		)
	}
	workspaceModel, err := workspace.WorkspaceByProjectID(context.TODO(), expModel.ProjectID)
	if err != nil && errors.Cause(err) != sql.ErrNoRows {
		return err
	}
	workspaceID := resolveWorkspaceID(workspaceModel)
	poolName, err := m.rm.ResolveResourcePool(
		rm.ResourcePoolName(activeConfig.Resources().ResourcePool()),
		workspaceID,
		activeConfig.Resources().SlotsPerTrial(),
	)
	if err != nil {
		return fmt.Errorf("invalid resource configuration: %w", err)
	}
	if _, err = m.rm.ValidateResources(sproto.ValidateResourcesRequest{
		ResourcePool: poolName.String(),
		Slots:        activeConfig.Resources().SlotsPerTrial(),
		IsSingleNode: false,
	}); err != nil {
		return fmt.Errorf("validating resources: %v", err)
	}
	taskContainerDefaults, err := m.rm.TaskContainerDefaults(
		poolName,
		m.config.TaskContainerDefaults,
	)
	if err != nil {
		return fmt.Errorf("error getting TaskContainerDefaults: %w", err)
	}
	taskSpec := *m.taskSpec
	taskSpec.TaskContainerDefaults = taskContainerDefaults
	owner, err := user.ByUsername(context.TODO(), expModel.Username)
	if err != nil {
		return errors.Wrapf(err, "retrieving full user on restart")
	}
	taskSpec.Owner = owner

	token, err := user.StartSession(context.Background(), owner)
	if err != nil {
		return fmt.Errorf("unable to create user session inside task: %w", err)
	}
	taskSpec.UserSessionToken = token

	log.WithField("experiment", expModel.ID).Debug("restoring experiment")
	snapshot, err := m.retrieveExperimentSnapshot(expModel)
	if err != nil {
		return errors.Wrapf(err, "failed to restore experiment %d", expModel.ID)
	}
	e, _, err := newExperiment(m, expModel, nil, activeConfig, &taskSpec)
	if err != nil {
		return errors.Wrapf(err, "failed to create experiment %d from model", expModel.ID)
	}
	if snapshot != nil {
		if err := e.restore(snapshot); err != nil {
			return errors.Wrap(err, "failed to restore experiment")
		}
		e.restored = true
	}

	if err := e.Start(); err != nil {
		return errors.Wrapf(err, "failed to start experiment %d", expModel.ID)
	}

	return nil
}

func (e *internalExperiment) restoreRun(
	ckpt *model.Checkpoint, searcher experiment.RunSearcherState,
) {
	syslog := e.syslog
	syslog.Debug("restoring run")

	var trial *model.Trial
	var trialID *int
	var err error

	if searcher.RunID != nil {
		if _, ok := e.trials[*searcher.RunID]; ok {
			syslog.Errorf("run %d was already restored, exiting", *searcher.RunID)
			return
		}
		trial, err = db.TrialByID(context.TODO(), int(*searcher.RunID))
		if err != nil {
			syslog.WithError(err).Error("failed to retrieve previous run, restoring new run")
		} else {
			syslog = syslog.WithField("run-id", trial.ID)
		}
	}

	taskID := model.TaskID(fmt.Sprintf("%d.%s", e.ID, model.NewTaskID()))

	if trial != nil {
		trialID = &trial.ID
		// If the run being restored was in a terminal state, replay the close for the searcher and exit.
		if model.TerminalStates[trial.State] {
			syslog.Debugf("run was in terminal state in restore: %s", trial.State)
			if !e.searcher.TrialIsClosed(*searcher.RunID) {
				e.runClosed(*searcher.RunID, nil)
			}
			return
		}
		trialTaskIDs, err := db.TrialTaskIDsByTrialID(context.TODO(), trial.ID)
		if err != nil {
			syslog.WithError(err).Error("failed to retrieve tasks for run")
		} else if len(trialTaskIDs) == 0 {
			syslog.Errorf("no tasks for run with id %d", trial.ID)
		} else {
			taskID = trialTaskIDs[len(trialTaskIDs)-1].TaskID
		}
	}

	config := schemas.Copy(e.activeConfig)
	t, err := newTrial(
		e.logCtx, taskID, e.JobID, e.StartTime, e.ID, e.State,
		searcher, e.rm, e.db, config, ckpt, e.taskSpec, e.generatedKeys, true, trialID,
		nil, e.RunClosed,
	)
	if err != nil {
		syslog.WithError(err).Error("failed restoring run, aborting restore")
		if searcher.RunID != nil && !e.searcher.TrialIsClosed(*searcher.RunID) {
			e.runClosed(*searcher.RunID, ptrs.Ptr(model.Errored))
		}
		return
	}
	e.trialCreated(t)

	syslog.Debug("restored run")
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
			return nil, errors.Wrap(err, "failed to shim experiment snapshot")
		}
		return snapshot, nil
	}
}

func (e *internalExperiment) snapshotAndSave() {
	es, err := e.snapshot()
	if err != nil {
		e.faultToleranceEnabled = false
		e.syslog.WithError(err).Errorf("failed to snapshot experiment, fault tolerance is lost")
		return
	}
	err = e.db.SaveSnapshot(e.ID, experimentSnapshotVersion, es)
	if err != nil {
		e.faultToleranceEnabled = false
		e.syslog.WithError(err).Errorf("failed to persist experiment snapshot, fault tolerance is lost")
		return
	}
}

// experimentSnapshotShims maps a version to the shim that bumps that version.
var experimentSnapshotShims = map[int]snapshotShimFunc{
	0: shimExperimentSnapshotV0,
	1: shimExperimentSnapshotV1,
	2: shimExperimentSnapshotV2,
	4: shimExperimentSnapshotV4,
}

// shimExperimentSnapshot shims an experiment snapshot to the version required by the master,
// returning an error in the event the shim fails or the snapshot version is greater
// than the current version (which could happen in a downgrade).
func shimExperimentSnapshot(snapshot []byte, version int) ([]byte, error) {
	return shimSnapshot(experimentSnapshotShims, snapshot, version)
}

func shimSnapshot(shims map[int]snapshotShimFunc, snapshot []byte, version int) ([]byte, error) {
	if version > experimentSnapshotVersion {
		return nil, fmt.Errorf("cannot shim from %d to %d", version, experimentSnapshotVersion)
	}
	var err error
	for version < experimentSnapshotVersion {
		shim, ok := shims[version]
		if !ok {
			return nil, fmt.Errorf("missing shim from %d to %d", version, experimentSnapshotVersion)
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

// ExperimentSnapshotShimError describes an error encountered while shimming.
type ExperimentSnapshotShimError struct {
	Message string
}

func (e ExperimentSnapshotShimError) Error() string {
	return e.Message
}

// shimExperimentSnapshotV4 shims a v4 snapshot to a v4 snapshot. From v4 to v5,
// Length lost its units and became just an int again.
func shimExperimentSnapshotV4(snapshot []byte) ([]byte, error) {
	var experimentSnapshotV4 map[string]interface{}
	if err := json.Unmarshal(snapshot, &experimentSnapshotV4); err != nil {
		return nil, err
	}

	trialSearcherState := experimentSnapshotV4["trial_searcher_state"].(map[string]interface{})
	for _, state := range trialSearcherState {
		mState, ok := state.(map[string]interface{})
		if !ok {
			return nil, ExperimentSnapshotShimError{Message: "state was not a map"}
		}

		op, ok := mState["Op"]
		if !ok {
			return nil, ExperimentSnapshotShimError{Message: "missing expected key Op"}
		}
		mOp, ok := op.(map[string]interface{})
		if !ok {
			return nil, ExperimentSnapshotShimError{Message: "Op was not a map"}
		}

		length, ok := mOp["Length"]
		if !ok {
			return nil, ExperimentSnapshotShimError{Message: "missing expected key Length"}
		}
		mLength, ok := length.(map[string]interface{})
		if !ok {
			return nil, ExperimentSnapshotShimError{Message: "Length was not a map"}
		}
		if len(mLength) != 1 {
			return nil, ExperimentSnapshotShimError{Message: fmt.Sprintf("bad length: %v", length)}
		}

		var units interface{}
		for _, u := range mLength {
			units = u
		}
		mOp["Length"] = units
	}

	return json.Marshal(experimentSnapshotV4)
}

// shimExperimentSnapshotV5 shims a v4 snapshot to a v5 snapshot. From v4 to v5:
// - `searcher_state.TrialsRequested` -> `RunsRequested`
// - `searcher_state.TrialsCreated` -> `RunsCreated`
// - `searcher_state.TrialsClosed` -> `RunsClosed`
// - `searcher_state.Exits` -> `Exits`
// - `searcher_state.Cancels` -> `Cancels`
// - `searcher_state.Failures` -> `Failures`
// - `searcher_state.TrialProgress` -> `RunProgress`
// - `searcher_state.CompletedOperations` -> dropped
// - `searcher_state.Shutdown` -> dropped
//
// - `trial_searcher_state` -> `run_searcher_state`
// - `trial_searcher_state.Create (searcher.Operation)` -> `run_searcher_state.Create (searcher.Action)`
// - `trial_searcher_state.Complete` -> dropped
// - `trial_searcher_state.Op (searcher.ValidateAfter)` -> dropped
// - `trial_searcher_state.Stop` -> dropped
// - `trial_searcher_state.Closed` -> `run_searcher_state.Closed`
func shimExperimentSnapshotV5(snapshot []byte) ([]byte, error) {

	type v4SearcherState struct {
		TrialsRequested   int                         `json:"trials_requested"`
		TrialsCreated     map[model.RequestID]bool    `json:"trials_created"`
		TrialsClosed      map[model.RequestID]bool    `json:"trials_closed"`
		Exits             map[model.RequestID]bool    `json:"exits"`
		Cancels           map[model.RequestID]bool    `json:"cancels"`
		Failures          map[model.RequestID]bool    `json:"failures"`
		TrialProgress     map[model.RequestID]float64 `json:"trial_progress"`
		Rand              *nprand.State               `json:"rand"`
		SearchMethodState json.RawMessage             `json:"search_method_state"`
	}
	type v4CreateOp struct {
		HParams   map[string]interface{} `json:"hparams"`
		RequestID model.RequestID        `json:"request_id"`
		TrialSeed uint32                 `json:"trial_seed"`
	}

	type v4TrialSearcherState struct {
		Create   v4CreateOp
		Stop     bool
		Closed   bool
		Complete bool
	}
	type experimentSnapshotV4 struct {
		SearcherState      v4SearcherState                          `json:"searcher_state"`
		TrialSearcherState map[model.RequestID]v4TrialSearcherState `json:"trial_searcher_state"`
	}

	v4ExperimentSnapshot := experimentSnapshotV4{}

	if err := json.Unmarshal(snapshot, &v4ExperimentSnapshot); err != nil {
		return nil, err
	}

	runsCreated, err := mapRequestIDToRunID(v4ExperimentSnapshot.SearcherState.TrialsCreated)
	runsClosed, err := mapRequestIDToRunID(v4ExperimentSnapshot.SearcherState.TrialsClosed)
	exits, err := mapRequestIDToRunID(v4ExperimentSnapshot.SearcherState.Exits)
	cancels, err := mapRequestIDToRunID(v4ExperimentSnapshot.SearcherState.Cancels)
	failures, err := mapRequestIDToRunID(v4ExperimentSnapshot.SearcherState.Failures)
	runProgress, err := mapRequestIDToRunID(v4ExperimentSnapshot.SearcherState.TrialProgress)

	searchMethodStateV4 := map[string]interface{}{}
	err = json.Unmarshal(v4ExperimentSnapshot.SearcherState.SearchMethodState, &searchMethodStateV4)
	searchMethodState, err := shimSearchMethodStateV5(searchMethodStateV4)
	runSearcherStateV4, err := mapRequestIDToRunID(v4ExperimentSnapshot.TrialSearcherState)
	if err != nil {
		return nil, ExperimentSnapshotShimError{Message: "unable to parse searcher state"}
	}

	var runSearcherState map[int]interface{}

	for rID, trialSearcherState := range runSearcherStateV4 {
		// Only for ASHA
		subsearchID, _ := searchMethodState.(map[string]interface{})["run_table"].(map[int]int)[rID]
		runSearcherState[rID] = map[string]interface{}{
			"Create": map[string]interface{}{
				"hparams":       trialSearcherState.Create.HParams,
				"run_seed":      trialSearcherState.Create.TrialSeed,
				"sub_search_id": subsearchID,
			},
			"RunID":   rID,
			"Stopped": trialSearcherState.Stop || trialSearcherState.Complete,
			"Closed":  trialSearcherState.Closed,
		}
	}

	experimentSnapshotV5 := map[string]interface{}{
		"searcher_state": map[string]interface{}{
			"runs_requested":      v4ExperimentSnapshot.SearcherState.TrialsRequested,
			"runs_created":        runsCreated,
			"runs_closed":         runsClosed,
			"exits":               exits,
			"cancels":             cancels,
			"failures":            failures,
			"run_progress":        runProgress,
			"rand":                v4ExperimentSnapshot.SearcherState.Rand,
			"search_method_state": searchMethodState,
		},
		"run_searcher_state": runSearcherState,
	}

	return json.Marshal(experimentSnapshotV5)
}

func shimSearchMethodStateV5(v4SearchMethodState map[string]interface{}) (interface{}, error) {
	searchMethodType, ok := v4SearchMethodState["search_method_type"].(searcher.SearchMethodType)
	if !ok {
		return nil, ExperimentSnapshotShimError{Message: "search_method_type not recognized"}
	}
	switch searchMethodType {
	case searcher.SingleSearch:
		fallthrough
	case searcher.RandomSearch:
		createdTrials, ok := v4SearchMethodState["created_trials"].(int)
		pendingTrials, ok := v4SearchMethodState["pending_trials"].(int)
		if !ok {
			return nil, ExperimentSnapshotShimError{Message: "cannot parse search_method_state"}
		}
		return map[string]interface{}{
			"created_runs":       createdTrials,
			"pending_runs":       pendingTrials,
			"search_method_type": v4SearchMethodState["search_method_type"],
		}, nil
	case searcher.GridSearch:
		createdTrials, ok := v4SearchMethodState["created_trials"].(int)
		remainingTrials, ok := v4SearchMethodState["remaining_trials"].(int)
		if !ok {
			return nil, ExperimentSnapshotShimError{Message: "cannot parse search_method_state"}
		}
		return map[string]interface{}{
			"created_runs":       createdTrials,
			"remaining_runs":     remainingTrials,
			"search_method_type": v4SearchMethodState["search_method_type"],
		}, nil
	case searcher.AdaptiveASHASearch:
		trialTable, ok := v4SearchMethodState["trial_table"].(map[model.RequestID]int)
		subSearchStatesV4, ok := v4SearchMethodState["sub_search_states"].([]map[string]interface{})
		runTable, err := mapRequestIDToRunID(trialTable)
		if !ok || err != nil {
			return nil, ExperimentSnapshotShimError{Message: "unable to parse search_method_state"}
		}
		var subSearchStates []map[string]interface{}

		for _, subSearchStateV4 := range subSearchStatesV4 {
			rungsV4, ok := subSearchStateV4["rungsV4"].([]map[string]interface{})
			trialRungs, ok := subSearchStateV4["trial_rungs"].(map[model.RequestID]int)
			earlyExitTrials, ok := subSearchStateV4["early_exit_trials"].(map[model.RequestID]bool)
			runRungs, err := mapRequestIDToRunID(trialRungs)
			earlyExitRuns, err := mapRequestIDToRunID(earlyExitTrials)
			trialsCompleted, ok := subSearchStateV4["trials_completed"].(int)
			invalidTrials, ok := subSearchStateV4["invalid_trials"].(int)
			if !ok || err != nil {
				return nil, ExperimentSnapshotShimError{Message: "unable to parse search_method_state"}
			}
			var rungs []map[string]interface{}

			for _, rungV4 := range rungsV4 {
				unitsNeeded, ok := rungV4["units_needed"].(uint64)
				metricsV4, ok := rungV4["metrics"].([]map[string]interface{})
				if !ok {
					return nil, ExperimentSnapshotShimError{Message: "unable to parse search_method_state"}
				}
				var metrics []map[string]interface{}

				for _, metric := range metricsV4 {
					requestID, ok := metric["request_id"].(model.RequestID)
					metric, ok := metric["metric"].(model.ExtendedFloat64)
					if !ok {
						return nil, ExperimentSnapshotShimError{Message: "unable to parse search_method_state"}
					}
					tID, err := db.TrialIDByRequestID(context.TODO(), requestID)
					if err != nil {
						return nil, ExperimentSnapshotShimError{Message: "unable to parse search_method_state"}
					}
					metrics = append(metrics, map[string]interface{}{
						"run_id": tID,
						"metric": metric,
					})
				}

				rungs = append(rungs, map[string]interface{}{
					"units_needed": unitsNeeded,
					"metrics":      metrics,
				})
			}
			subSearchStates = append(subSearchStates, map[string]interface{}{
				"invalid_runs":    invalidTrials,
				"runs_completed":  trialsCompleted,
				"early_exit_runs": earlyExitRuns,
				"run_rungs":       runRungs,
			})
		}
		return map[string]interface{}{
			"run_table":          runTable,
			"sub_search_states":  subSearchStates,
			"search_method_type": v4SearchMethodState["search_method_type"],
		}, nil
	default:
		return nil, ExperimentSnapshotShimError{Message: "unsupported search_method_type"}
	}
}

func mapRequestIDToRunID[T any](obj map[model.RequestID]T) (map[int]T, error) {
	runIDMap := make(map[int]T)

	for k, v := range obj {
		tID, err := db.TrialIDByRequestID(context.TODO(), k)
		if err != nil {
			return nil, err
		}
		runIDMap[*tID] = v
	}
	return runIDMap, nil
}
