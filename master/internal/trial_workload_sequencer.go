package internal

import (
	"encoding/json"
	"fmt"
	"math"

	"github.com/determined-ai/determined/master/pkg/workload"

	"github.com/pkg/errors"

	"github.com/determined-ai/determined/master/pkg/model"
	"github.com/determined-ai/determined/master/pkg/searcher"
)

type (
	trialWorkloadSequencerState struct {
		BatchesTowardsCurrentOp int `json:"batches_towards_current_op"`
		BatchesSinceLastVal     int `json:"batches_since_last_val"`
		BatchesSinceLastCkpt    int `json:"batches_since_last_chkp"`
		TotalBatchesProcessed   int `json:"total_batches_processed"`

		NeedInitialValidation  bool `json:"need_initial_validation"`
		NeedPostValidationCkpt bool `json:"need_post_validation_ckpt"`

		ExitingEarly bool `json:"exiting_early"`
		GracefulStop bool `json:"graceful_stop"`

		CurOpIdx  int `json:"cur_op_idx"`
		CurStepID int `json:"cur_step_id"`

		LatestCheckpoint *model.Checkpoint            `json:"latest_checkpoint"`
		LatestSnapshot   *trialWorkloadSequencerState `json:"latest_snapshot"`
	}

	// trialWorkloadSequencer manages transforming the work requested by the searcher into Workloads
	// and yielding them to the trial to run. It has 3 main methods, OperationRequested, Workload, and
	// WorkloadCompleted. OperationRequested adds operations requested by the search method,
	// WorkloadCompleted updates the internal state of the sequencer to account for the completed
	// message, and Workload examines the state to determine the next workload.
	trialWorkloadSequencer struct {
		trialWorkloadSequencerState

		trialID      int
		trialIDValid bool

		ops []searcher.ValidateAfter

		experiment *model.Experiment
		create     searcher.Create

		checkpointPolicy    string
		minValidationPeriod model.Length
		minCheckpointPeriod model.Length

		unitContext model.UnitContext

		schedulingUnit int
	}
)

func (s trialWorkloadSequencerState) deepCopy() *trialWorkloadSequencerState {
	c := s
	return &c
}

func newTrialWorkloadSequencer(
	exp *model.Experiment, create searcher.Create, firstCheckpoint *model.Checkpoint,
) *trialWorkloadSequencer {
	return &trialWorkloadSequencer{
		trialWorkloadSequencerState: trialWorkloadSequencerState{
			NeedInitialValidation: exp.Config.PerformInitialValidation,
			LatestCheckpoint:      firstCheckpoint,
			LatestSnapshot: &trialWorkloadSequencerState{
				NeedInitialValidation: exp.Config.PerformInitialValidation,
				LatestCheckpoint:      firstCheckpoint,
			},
		},
		checkpointPolicy:    exp.Config.CheckpointPolicy,
		minValidationPeriod: exp.Config.MinValidationPeriod,
		minCheckpointPeriod: exp.Config.MinCheckpointPeriod,
		unitContext: model.NewUnitContext(
			exp.Config.Unit(), create.Hparams.GlobalBatchSize(), exp.Config.RecordsPerEpoch),
		schedulingUnit: exp.Config.SchedulingUnit,
		create:         create,
		experiment:     exp,
	}
}

func (s *trialWorkloadSequencer) Snapshot() (json.RawMessage, error) {
	return json.Marshal(s.trialWorkloadSequencerState)
}

func (s *trialWorkloadSequencer) Restore(state json.RawMessage) error {
	return json.Unmarshal(state, &s.trialWorkloadSequencerState)
}

func (s *trialWorkloadSequencer) WorkloadManagerType() model.WorkloadManagerType {
	return model.TrialWorkloadManagerType
}

// OperationRequested records an operation requested by the searcher.
func (s *trialWorkloadSequencer) OperationRequested(op searcher.ValidateAfter) {
	s.ops = append(s.ops, op)
}

func (s *trialWorkloadSequencer) SetTrialID(trialID int) {
	s.trialID = trialID
	s.trialIDValid = true
}

// WorkloadCompleted receives the searcher.CompletedMessage and updates the internal state of the
// trialWorkloadSequencer accordingly. It and WorkloadFailed are the only methods that should update
// the state. It snapshots this state whenever we've just completed a checkpoint, and should be the
// only method to alter the snapshot, too.
func (s *trialWorkloadSequencer) WorkloadCompleted(
	msg workload.CompletedMessage, isBestValFunc func() bool,
) (reportValidation bool, err error) {
	// Checkpoints are allowed even if they were not specified by sequencer.workload(). This can
	// occur after a call to precloseCheckpointWorkload or during a replay of a trial that was
	// descheduled.
	if s.UpToDate() {
		if msg.Workload.Kind != workload.CheckpointModel {
			return false, errors.Errorf(
				"illegal non-checkpoint workload completed message received: %s", msg.Workload)
		}
	} else {
		w, err := s.Workload()
		if err != nil {
			return false, errors.Wrap(err, "error checking workload")
		}
		if msg.Workload != w {
			if msg.Workload.Kind != workload.CheckpointModel {
				return false, errors.Errorf(
					"illegal completed message received: expected checkpoint or %s, got %s", w, msg.Workload)
			}
		}
	}

	switch msg.Workload.Kind {
	case workload.RunStep:
		s.runStepCompleted(msg)
		return false, nil
	case workload.CheckpointModel:
		s.checkpointModelCompleted(msg)
		return false, nil
	case workload.ComputeValidationMetrics:
		return s.computeValidationMetricsCompleted(msg, isBestValFunc)
	default:
		return false, errors.New("invalid operation for trialWorkloadSequencer")
	}
}

// WorkloadFailed notifies the sequencer that the workload failed. The sequencer returns
// with a bool indicating if the failure should result in a hard or graceful stop.
func (s *trialWorkloadSequencer) WorkloadFailed(
	msg workload.CompletedMessage,
	reason workload.ExitedReason,
) (bool, error) {
	if reason == workload.UserCanceled || reason == workload.InvalidHP {
		// UserCanceled and InvalidHP are still considered "completed".
		if _, err := s.WorkloadCompleted(msg, func() bool {
			return false
		}); err != nil {
			return false, fmt.Errorf("failed to complete workload with exit reason %v: %w", reason, err)
		}
		s.GracefulStop = true
	}
	s.ExitingEarly = true
	return s.ExitingEarly && !s.GracefulStop, nil
}

// runStepCompleted updates the internal state of the sequencer to account for a completed
// RUN_STEP workload.
func (s *trialWorkloadSequencer) runStepCompleted(msg workload.CompletedMessage) {
	s.CurStepID++
	s.TotalBatchesProcessed += msg.Workload.NumBatches
	s.BatchesTowardsCurrentOp += msg.Workload.NumBatches
	s.BatchesSinceLastVal += msg.Workload.NumBatches
	s.BatchesSinceLastCkpt += msg.Workload.NumBatches
	// We choose not to handle partial batches.
	if s.ops[s.CurOpIdx].Length.EqualWithinBatch(s.TotalBatchesProcessed, s.unitContext) {
		s.CurOpIdx++
		s.BatchesTowardsCurrentOp = 0
	}
}

// computeValidationMetricsCompleted updates the internal state of the sequencer to account for a
// completed COMPUTE_VALIDATION_METRICS worklaod.
func (s *trialWorkloadSequencer) computeValidationMetricsCompleted(
	msg workload.CompletedMessage, isBestValFunc func() bool,
) (reportValidation bool, err error) {
	if msg.ValidationMetrics == nil {
		return false, errors.New("missing validation metrics")
	}
	hadSearcherValidation := s.hasSearcherValidation()
	s.BatchesSinceLastVal = 0
	if s.NeedInitialValidation {
		s.NeedInitialValidation = false
	}
	if s.BatchesSinceLastCkpt != 0 {
		switch s.checkpointPolicy {
		case model.AllCheckpointPolicy:
			s.NeedPostValidationCkpt = true
		case model.BestCheckpointPolicy:
			if isBestValFunc() {
				s.NeedPostValidationCkpt = true
			}
		}
	}
	if s.BatchesSinceLastCkpt == 0 {
		// If this we haven't run any more batches since we checkpointed, we can snapshot here, too.
		s.snapshotState()
	}
	return hadSearcherValidation, nil
}

// checkpointModelCompleted updates the internal state of the sequencer to account for a completed
// CHECKPOINT_MODEL workload.
func (s *trialWorkloadSequencer) checkpointModelCompleted(msg workload.CompletedMessage) {
	defer s.snapshotState()
	checkpoint := checkpointFromCheckpointMetrics(*msg.CheckpointMetrics)
	s.BatchesSinceLastCkpt = 0
	s.NeedPostValidationCkpt = false
	s.LatestCheckpoint = &checkpoint
}

// Workload introspects the current state of the trialWorkloadSequencer, without altering it, and
// determines the next workload to run with the information it has. It should not alter state.
func (s trialWorkloadSequencer) Workload() (workload.Workload, error) {
	if s.UpToDate() {
		return workload.Workload{},
			errors.New("cannot call sequencer.Workload() with sequencer.UpToDate() == true")
	}

	if !s.trialIDValid {
		return workload.Workload{},
			errors.New("cannot call sequencer.Workload() before sequencer.SetTrialID()")
	}

	if s.preSearchValidationCheckpointNeeded() || s.postGracefulStopCheckpointNeeded() ||
		s.postValidationCheckpointNeeded() || s.minCheckpointNeeded() {
		return s.checkpoint(), nil
	}

	if s.hasSearcherValidation() || s.NeedInitialValidation || s.minValidationNeeded() {
		return s.validate(), nil
	}

	batchesLeft := s.ops[s.CurOpIdx].Length.ToNearestBatch(s.unitContext) - s.TotalBatchesProcessed
	batchesTilVal := s.batchesUntilValNeeded()
	batchesTilCkpt := s.batchesUntilCkptNeeded()
	batchesThisStep := max(min(
		batchesLeft,
		batchesTilVal,
		batchesTilCkpt,
		s.schedulingUnit,
	), 1)
	return s.train(batchesThisStep), nil
}

func (s trialWorkloadSequencer) hasSearcherValidation() bool {
	// If we just finished an op, but didn't end with a validation, we have a searcher validation.
	if s.BatchesTowardsCurrentOp == 0 && s.BatchesSinceLastVal != 0 {
		return true
	}
	return false
}

// PrecloseCheckpointWorkload determines what the preclose checkpoint workload should be.
func (s trialWorkloadSequencer) PrecloseCheckpointWorkload() *workload.Workload {
	if s.BatchesSinceLastCkpt == 0 {
		return nil
	}
	// Because no workloads can be issued without a trialID, having no trialID indicates we cannot
	// have finished any workloads at all.
	if !s.trialIDValid {
		return nil
	}
	checkpoint := s.checkpoint()
	return &checkpoint
}

// TerminateWorkload determines what the terminate workload should be.
func (s trialWorkloadSequencer) TerminateWorkload() *workload.Workload {
	return &workload.Workload{
		Kind:         workload.Terminate,
		ExperimentID: s.experiment.ID,
		TrialID:      s.trialID,
		StepID:       s.CurStepID,
	}
}

// snapshotState sets the current state to the latest snapshot and clears out the previous snapshot
// to reduce memory usage (we'll only ever restore to the latest one.
func (s *trialWorkloadSequencer) snapshotState() {
	s.LatestSnapshot = s.trialWorkloadSequencerState.deepCopy()
	s.LatestSnapshot.LatestSnapshot = nil
}

// RollBackSequencer rolls back the sequencer to the latest checkpoint and sets the latest
// checkpoint back to the one we just rolled back to.
func (s *trialWorkloadSequencer) RollBackSequencer() (int, error) {
	s.trialWorkloadSequencerState = *s.LatestSnapshot
	s.LatestSnapshot = s.trialWorkloadSequencerState.deepCopy()
	return s.TotalBatchesProcessed, nil
}

// UpToDate returns if the sequencer has completed all searcher requested operations.
func (s *trialWorkloadSequencer) UpToDate() bool {
	// If all operations for the last asked-for step are done, then the trial has no more workloads
	// to run at the moment. We check len(s.ops) <= s.CurOpIdx for the case when in restart
	// the trial has been recreated from a snapshot but not received its current operations.
	return len(s.ops) <= s.CurOpIdx && !s.hasSearcherValidation() ||
		s.ExitingEarly && !s.postGracefulStopCheckpointNeeded()
}

func (s trialWorkloadSequencer) train(numBatches int) workload.Workload {
	return workload.Workload{
		Kind:                  workload.RunStep,
		ExperimentID:          s.experiment.ID,
		TrialID:               s.trialID,
		StepID:                s.CurStepID + 1,
		NumBatches:            numBatches,
		PriorBatchesProcessed: s.TotalBatchesProcessed,
	}
}

func (s trialWorkloadSequencer) validate() workload.Workload {
	return workload.Workload{
		Kind:                  workload.ComputeValidationMetrics,
		ExperimentID:          s.experiment.ID,
		TrialID:               s.trialID,
		StepID:                s.CurStepID,
		PriorBatchesProcessed: s.TotalBatchesProcessed,
	}
}

func (s trialWorkloadSequencer) checkpoint() workload.Workload {
	return workload.Workload{
		Kind:                  workload.CheckpointModel,
		ExperimentID:          s.experiment.ID,
		TrialID:               s.trialID,
		StepID:                s.CurStepID,
		PriorBatchesProcessed: s.TotalBatchesProcessed,
	}
}

func (s *trialWorkloadSequencer) minValidationNeeded() bool {
	if s.minValidationPeriod.Units == 0 {
		return false
	}
	return s.minValidationPeriod.EqualWithinBatch(s.BatchesSinceLastVal, s.unitContext)
}

func (s *trialWorkloadSequencer) batchesUntilValNeeded() int {
	if s.minValidationPeriod.Units == 0 {
		return math.MaxInt32
	}
	return s.minValidationPeriod.ToNearestBatch(s.unitContext) - s.BatchesSinceLastVal
}

func (s *trialWorkloadSequencer) minCheckpointNeeded() bool {
	if s.minCheckpointPeriod.Units == 0 {
		return false
	}
	return s.minCheckpointPeriod.EqualWithinBatch(s.BatchesSinceLastCkpt, s.unitContext)
}

func (s *trialWorkloadSequencer) postGracefulStopCheckpointNeeded() bool {
	return s.GracefulStop && s.BatchesSinceLastCkpt != 0
}

func (s *trialWorkloadSequencer) postValidationCheckpointNeeded() bool {
	return s.NeedPostValidationCkpt && s.BatchesSinceLastCkpt != 0
}

func (s *trialWorkloadSequencer) preSearchValidationCheckpointNeeded() bool {
	return s.hasSearcherValidation() && s.BatchesSinceLastCkpt != 0
}

func (s *trialWorkloadSequencer) batchesUntilCkptNeeded() int {
	if s.minCheckpointPeriod.Units == 0 {
		return math.MaxInt32
	}
	return s.minCheckpointPeriod.ToNearestBatch(s.unitContext) - s.BatchesSinceLastCkpt
}

func (s *trialWorkloadSequencer) Progress() model.PartialUnits {
	return model.UnitsFromBatches(s.unitContext, s.TotalBatchesProcessed)
}

func min(initial int, values ...int) int {
	minValue := initial
	for _, value := range values {
		if value < minValue {
			minValue = value
		}
	}
	return minValue
}

func max(initial int, values ...int) int {
	maxValue := initial
	for _, value := range values {
		if value > maxValue {
			maxValue = value
		}
	}
	return maxValue
}
