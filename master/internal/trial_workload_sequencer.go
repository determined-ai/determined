package internal

import (
	"encoding/json"
	"math"

	"github.com/sirupsen/logrus"

	"github.com/determined-ai/determined/master/pkg/workload"

	"github.com/pkg/errors"

	"github.com/determined-ai/determined/master/pkg/actor"
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

		// A trial may receive checkpoint completed messages before it requests them in the event that a
		// trial is descheduled or paused. For example, a trial that plans to do:
		//     RUN_STEP, COMPUTE_VALIDATION_METRICS, CHECKPOINT_MODEL.
		// that is paused during the run step, receives its checkpoint completed message out of order.
		// We only need to keep one because we only expect them out of order within a step; once we
		// move on to the next step, we can be sure we don't need to cache it for later.
		CachedCheckpoint *workload.CompletedMessage `json:"cached_checkpoint"`
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

		ops []searcher.Runnable

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
			CachedCheckpoint:      nil,
			LatestSnapshot: &trialWorkloadSequencerState{
				NeedInitialValidation: exp.Config.PerformInitialValidation,
				LatestCheckpoint:      firstCheckpoint,
				CachedCheckpoint:      nil,
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
func (s *trialWorkloadSequencer) OperationRequested(op searcher.Runnable) {
	s.ops = append(s.ops, op)
}

// CompleteCachedCheckpoints attempts to complete cached checkpoints that we received previously
// but did not need yet.
func (s *trialWorkloadSequencer) CompleteCachedCheckpoints() (
	searcher.Runnable, interface{}, error,
) {
	if s.UpToDate() {
		return nil, nil, nil
	}

	w, err := s.Workload()
	if err != nil {
		return nil, nil, errors.Wrap(err, "error getting workload from sequencer")
	}

	if s.CachedCheckpoint != nil && s.CachedCheckpoint.Workload.StepID == w.StepID {
		logrus.Infof("trial completed workload: %v", s.CachedCheckpoint.Workload)
		op, metrics, err := s.WorkloadCompleted(*s.CachedCheckpoint, nil)
		s.CachedCheckpoint = nil
		return op, metrics, errors.Wrap(err, "failed to complete cached checkpoint")
	}
	return nil, nil, nil
}

func (s *trialWorkloadSequencer) SetTrialID(trialID int) {
	s.trialID = trialID
	s.trialIDValid = true
}

// WorkloadCompleted receives the searcher.CompletedMessage and updates the internal state of the
// trialWorkloadSequencer accordingly. It is the only method that should update the state.
// It snapshots this state whenever we've just completed a checkpoint, and should be the only method
// to alter the snapshot, too.
func (s *trialWorkloadSequencer) WorkloadCompleted(
	msg workload.CompletedMessage, isBestValFuture actor.Response,
) (op searcher.Runnable, metrics interface{}, err error) {
	// Checkpoints are allowed even if they were not specified by sequencer.workload(). This can
	// occur after a call to precloseCheckpointWorkload or during a replay of a trial that was
	// descheduled.
	if s.UpToDate() {
		if msg.Workload.Kind != workload.CheckpointModel {
			return nil, nil, errors.Errorf(
				"illegal non-checkpoint workload completed message received: %s", msg.Workload)
		}
	} else {
		w, err := s.Workload()
		if err != nil {
			return nil, nil, errors.Wrap(err, "error checking workload")
		}
		if msg.Workload != w {
			if msg.Workload.Kind != workload.CheckpointModel {
				return nil, nil, errors.Errorf(
					"illegal completed message received: expected checkpoint or %s, got %s", w, msg.Workload)
			}
		}
	}
	if msg.ExitedReason != nil {
		s.ExitingEarly = true
		if *msg.ExitedReason == workload.UserCanceled || *msg.ExitedReason == workload.InvalidHP {
			s.GracefulStop = true
		} else {
			return nil, nil, nil
		}
	}

	switch msg.Workload.Kind {
	case workload.RunStep:
		return s.runStepCompleted(msg), nil, nil
	case workload.CheckpointModel:
		op, metrics := s.checkpointModelCompleted(msg)
		return op, metrics, nil
	case workload.ComputeValidationMetrics:
		op, metrics := s.computeValidationMetricsCompleted(msg, isBestValFuture)
		return op, metrics, nil
	default:
		return nil, nil, errors.New("invalid operation for trialWorkloadSequencer")
	}
}

// runStepCompleted updates the internal state of the sequencer to account for a completed
// RUN_STEP workload.
func (s *trialWorkloadSequencer) runStepCompleted(msg workload.CompletedMessage) searcher.Runnable {
	s.CurStepID++
	s.TotalBatchesProcessed += msg.Workload.NumBatches
	s.BatchesTowardsCurrentOp += msg.Workload.NumBatches
	s.BatchesSinceLastVal += msg.Workload.NumBatches
	s.BatchesSinceLastCkpt += msg.Workload.NumBatches
	if tOp, ok := s.ops[s.CurOpIdx].(searcher.Train); ok &&
		// We choose not to handle partial batches.
		tOp.Length.EqualWithinBatch(s.BatchesTowardsCurrentOp, s.unitContext) {
		s.CurOpIdx++
		s.BatchesTowardsCurrentOp = 0
		return tOp
	}
	return nil
}

// computeValidationMetricsCompleted updates the internal state of the sequencer to account for a
// completed COMPUTE_VALIDATION_METRICS worklaod.
func (s *trialWorkloadSequencer) computeValidationMetricsCompleted(
	msg workload.CompletedMessage, isBestValFuture actor.Response,
) (searcher.Runnable, interface{}) {
	s.BatchesSinceLastVal = 0
	if s.NeedInitialValidation {
		s.NeedInitialValidation = false
	}
	if s.BatchesSinceLastCkpt != 0 {
		switch s.checkpointPolicy {
		case model.AllCheckpointPolicy:
			s.NeedPostValidationCkpt = true
		case model.BestCheckpointPolicy:
			if isBestValidation, ok := isBestValFuture.Get().(bool); ok && isBestValidation {
				s.NeedPostValidationCkpt = true
			}
		}
	}
	if tOp, ok := s.ops[s.CurOpIdx].(searcher.Validate); ok {
		s.CurOpIdx++
		// Snapshot here, so we catch the curOpIdx being incremented.
		if s.BatchesSinceLastCkpt == 0 {
			s.snapshotState()
		}
		return tOp, msg.ValidationMetrics
	}
	if s.BatchesSinceLastCkpt == 0 {
		// If this we haven't run any more batches since we checkpointed, we can snapshot here, too.
		s.snapshotState()
	}
	return nil, nil
}

// checkpointModelCompleted updates the internal state of the sequencer to account for a completed
// CHECKPOINT_MODEL workload.
func (s *trialWorkloadSequencer) checkpointModelCompleted(
	msg workload.CompletedMessage,
) (searcher.Runnable, interface{}) {
	defer s.snapshotState()
	checkpoint := checkpointFromCheckpointMetrics(*msg.CheckpointMetrics)
	s.BatchesSinceLastCkpt = 0
	s.NeedPostValidationCkpt = false
	s.LatestCheckpoint = &checkpoint
	if !s.UpToDate() {
		if tOp, ok := s.ops[s.CurOpIdx].(searcher.Checkpoint); ok {
			s.CurOpIdx++
			return tOp, msg.CheckpointMetrics
		}
		s.CachedCheckpoint = &msg
	} else {
		s.CachedCheckpoint = &msg
	}
	return nil, nil
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

	if s.NeedInitialValidation {
		return s.validate(), nil
	}

	if s.postGracefulStopCheckpointNeeded() {
		return s.checkpoint(), nil
	}

	if s.postValidationCheckpointNeeded() {
		return s.checkpoint(), nil
	}

	if s.minValidationNeeded() {
		return s.validate(), nil
	}

	if s.minCheckpointNeeded() {
		return s.checkpoint(), nil
	}

	switch tOp := s.ops[s.CurOpIdx].(type) {
	case searcher.Validate:
		// We choose to always checkpoint before completing any searcher operations. This allows us
		// to reset and all trials while keeping them in a consistent state.
		if s.BatchesSinceLastCkpt != 0 {
			return s.checkpoint(), nil
		}
		return s.validate(), nil
	case searcher.Checkpoint:
		return s.checkpoint(), nil
	case searcher.Train:
		batchesLeft := tOp.Length.ToNearestBatch(s.unitContext) - s.BatchesTowardsCurrentOp
		batchesTilVal := s.batchesUntilValNeeded()
		batchesTilCkpt := s.batchesUntilCkptNeeded()
		batchesThisStep := max(min(
			batchesLeft,
			batchesTilVal,
			batchesTilCkpt,
			s.schedulingUnit,
		), 1)
		return s.train(batchesThisStep), nil
	default:
		return workload.Workload{}, errors.New("unexpected op type determining workload")
	}
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
	return len(s.ops) <= s.CurOpIdx || s.ExitingEarly && !s.postGracefulStopCheckpointNeeded()
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

func (s *trialWorkloadSequencer) batchesUntilCkptNeeded() int {
	if s.minCheckpointPeriod.Units == 0 {
		return math.MaxInt32
	}
	return s.minCheckpointPeriod.ToNearestBatch(s.unitContext) - s.BatchesSinceLastCkpt
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
