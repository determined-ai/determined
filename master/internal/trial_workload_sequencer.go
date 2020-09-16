package internal

import (
	"github.com/determined-ai/determined/master/pkg/workload"
	"math"

	"github.com/pkg/errors"
	log "github.com/sirupsen/logrus"

	"github.com/determined-ai/determined/master/pkg/actor"
	"github.com/determined-ai/determined/master/pkg/model"
	"github.com/determined-ai/determined/master/pkg/searcher"
)

// trialWorkloadSequencer manages transforming the work requested by the searcher into Workloads
// and yielding them to the trial to run. It has 3 main methods, OperationRequested, Workload, and
// WorkloadCompleted. OperationRequested adds operations requested by the search method,
// WorkloadCompleted updates the internal state of the sequencer to account for the completed
// message, and Workload examines the state to determine the next workload.
type trialWorkloadSequencer struct {
	ops []searcher.Runnable
	trialWorkloadSequencerState
	latestCheckpointSequencerSnapshot trialWorkloadSequencerState

	experiment *model.Experiment
	create     searcher.Create

	checkpointPolicy    string
	minValidationPeriod model.Length
	minCheckpointPeriod model.Length

	unitContext model.UnitContext

	schedulingUnit int

	trialID      int
	trialIDValid bool
}

type trialWorkloadSequencerState struct {
	batchesTowardsCurrentOp int
	batchesSinceLastVal     int
	batchesSinceLastCkpt    int
	totalBatchesProcessed   int

	needInitialValidation  bool
	needPostValidationCkpt bool

	exitingEarly      bool
	userRequestedStop bool

	curOpIdx  int
	curStepID int

	latestCheckpoint *model.Checkpoint
	// A trial may receive checkpoint completed messages before it requests them in the event that a
	// trial is descheduled or paused. For example, a trial that plans to do:
	//     RUN_STEP, COMPUTE_VALIDATION_METRICS, CHECKPOINT_MODEL.
	// that is paused during the run step, receives its checkpoint completed message out of order.
	cachedCheckpoints map[workload.Workload]workload.CompletedMessage
}

func (s *trialWorkloadSequencerState) deepCopy() trialWorkloadSequencerState {
	cachedCheckpoints := map[workload.Workload]workload.CompletedMessage{}
	for v, k := range s.cachedCheckpoints {
		cachedCheckpoints[v] = k
	}
	return trialWorkloadSequencerState{
		batchesTowardsCurrentOp: s.batchesTowardsCurrentOp,
		batchesSinceLastVal:     s.batchesSinceLastVal,
		batchesSinceLastCkpt:    s.batchesSinceLastCkpt,
		needInitialValidation:   s.needInitialValidation,
		needPostValidationCkpt:  s.needPostValidationCkpt,
		exitingEarly:            s.exitingEarly,
		userRequestedStop:       s.userRequestedStop,
		totalBatchesProcessed:   s.totalBatchesProcessed,
		curOpIdx:                s.curOpIdx,
		curStepID:               s.curStepID,
		latestCheckpoint:        s.latestCheckpoint,
		cachedCheckpoints:       cachedCheckpoints,
	}
}

func newTrialWorkloadSequencer(
	exp *model.Experiment, create searcher.Create, firstCheckpoint *model.Checkpoint,
) *trialWorkloadSequencer {
	state := trialWorkloadSequencerState{
		needInitialValidation: exp.Config.PerformInitialValidation,
		latestCheckpoint:      firstCheckpoint,
		cachedCheckpoints:     map[workload.Workload]workload.CompletedMessage{},
	}
	return &trialWorkloadSequencer{
		trialWorkloadSequencerState:       state,
		latestCheckpointSequencerSnapshot: state.deepCopy(),
		checkpointPolicy:                  exp.Config.CheckpointPolicy,
		minValidationPeriod:               exp.Config.MinValidationPeriod,
		minCheckpointPeriod:               exp.Config.MinCheckpointPeriod,
		unitContext: model.NewUnitContext(
			exp.Config.Unit(), create.Hparams.GlobalBatchSize(), exp.Config.RecordsPerEpoch),
		schedulingUnit: exp.Config.SchedulingUnit,
		create:         create,
		experiment:     exp,
	}
}

func (s *trialWorkloadSequencer) SetTrialID(trialID int) {
	s.trialID = trialID
	s.trialIDValid = true
}

func (s *trialWorkloadSequencer) LatestCheckpoint() *model.Checkpoint {
	return s.latestCheckpoint
}

func (s *trialWorkloadSequencer) WorkloadManagerType() model.WorkloadManagerType {
	return model.TrialWorkloadManagerType
}

// OperationRequested records an operation requested by the searcher.
func (s *trialWorkloadSequencer) OperationRequested(op searcher.Runnable) error {
	switch op := op.(type) {
	case searcher.Runnable:
		s.ops = append(s.ops, op)
	default:
		return errors.Errorf("illegal workload for trialWorkloadSequencer: %v", op)
	}
	return nil
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
		return nil, nil, errors.Wrap(err, "Error getting workload from sequencer")
	}

	if msg, ok := s.cachedCheckpoints[w]; ok {
		log.Infof("trial completed workload: %v", msg.Workload)
		delete(s.cachedCheckpoints, w)
		op, metrics, err := s.WorkloadCompleted(msg, nil)
		if err != nil {
			return nil, nil, errors.Wrap(err, "Error completing cached checkpoint")
		}
		return op, metrics, nil
	}
	return nil, nil, nil
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
		s.exitingEarly = true
		if *msg.ExitedReason == workload.UserCanceled {
			s.userRequestedStop = true
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
	s.curStepID++
	s.totalBatchesProcessed += msg.Workload.NumBatches
	s.batchesTowardsCurrentOp += msg.Workload.NumBatches
	s.batchesSinceLastVal += msg.Workload.NumBatches
	s.batchesSinceLastCkpt += msg.Workload.NumBatches
	if tOp, ok := s.ops[s.curOpIdx].(searcher.Train); ok &&
		// We choose not to handle partial batches.
		tOp.Length.EqualWithinBatch(s.batchesTowardsCurrentOp, s.unitContext) {
		s.curOpIdx++
		s.batchesTowardsCurrentOp = 0
		return tOp
	}
	return nil
}

// computeValidationMetricsCompleted updates the internal state of the sequencer to account for a
// completed COMPUTE_VALIDATION_METRICS worklaod.
func (s *trialWorkloadSequencer) computeValidationMetricsCompleted(
	msg workload.CompletedMessage, isBestValFuture actor.Response,
) (searcher.Runnable, interface{}) {
	s.batchesSinceLastVal = 0
	if s.needInitialValidation {
		s.needInitialValidation = false
	}
	if s.batchesSinceLastCkpt != 0 {
		switch s.checkpointPolicy {
		case model.AllCheckpointPolicy:
			s.needPostValidationCkpt = true
		case model.BestCheckpointPolicy:
			if isBestValidation, ok := isBestValFuture.Get().(bool); ok && isBestValidation {
				s.needPostValidationCkpt = true
			}
		}
	}
	if tOp, ok := s.ops[s.curOpIdx].(searcher.Validate); ok {
		s.curOpIdx++
		// Snapshot here, so we catch the curOpIdx being incremented.
		if s.batchesSinceLastCkpt == 0 {
			s.snapshotState()
		}
		return tOp, msg.ValidationMetrics
	}
	if s.batchesSinceLastCkpt == 0 {
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
	s.batchesSinceLastCkpt = 0
	s.needPostValidationCkpt = false
	s.latestCheckpoint = &checkpoint
	if !s.UpToDate() {
		if tOp, ok := s.ops[s.curOpIdx].(searcher.Checkpoint); ok {
			s.curOpIdx++
			return tOp, msg.CheckpointMetrics
		}
		s.cachedCheckpoints[msg.Workload] = msg
	} else {
		s.cachedCheckpoints[msg.Workload] = msg
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

	if s.needInitialValidation {
		return s.validate(), nil
	}

	if s.postUserCancellationCheckpointNeeded() {
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

	switch tOp := s.ops[s.curOpIdx].(type) {
	case searcher.Validate:
		// We choose to always checkpoint before completing any searcher operations. This allows us
		// to confidently rollback searcher events and all trials while keeping them in a consistent
		// state.
		if s.batchesSinceLastCkpt != 0 {
			return s.checkpoint(), nil
		}
		return s.validate(), nil
	case searcher.Checkpoint:
		return s.checkpoint(), nil
	case searcher.Train:
		batchesLeft := tOp.Length.ToNearestBatch(s.unitContext) - s.batchesTowardsCurrentOp
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
	if s.batchesSinceLastCkpt == 0 {
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
		StepID:       s.curStepID,
	}
}

// Rollback sequencer rolls back the sequencer to the last available step with a checkpoint.
func (s *trialWorkloadSequencer) RollBackSequencer() int {
	s.trialWorkloadSequencerState = s.latestCheckpointSequencerSnapshot.deepCopy()
	return s.curStepID
}

// UpToDate returns if the sequencer has completed all searcher requested operations.
func (s *trialWorkloadSequencer) UpToDate() bool {
	// If all operations for the last asked-for step are done, then the trial has no more workloads
	// to run at the moment.
	return len(s.ops) == s.curOpIdx || s.exitingEarly && !s.postUserCancellationCheckpointNeeded()
}

func (s trialWorkloadSequencer) train(numBatches int) workload.Workload {
	return workload.Workload{
		Kind:                  workload.RunStep,
		ExperimentID:          s.experiment.ID,
		TrialID:               s.trialID,
		StepID:                s.curStepID + 1,
		NumBatches:            numBatches,
		TotalBatchesProcessed: s.totalBatchesProcessed,
	}
}

func (s trialWorkloadSequencer) validate() workload.Workload {
	return workload.Workload{
		Kind:                  workload.ComputeValidationMetrics,
		ExperimentID:          s.experiment.ID,
		TrialID:               s.trialID,
		StepID:                s.curStepID,
		TotalBatchesProcessed: s.totalBatchesProcessed,
	}
}

func (s trialWorkloadSequencer) checkpoint() workload.Workload {
	return workload.Workload{
		Kind:                  workload.CheckpointModel,
		ExperimentID:          s.experiment.ID,
		TrialID:               s.trialID,
		StepID:                s.curStepID,
		TotalBatchesProcessed: s.totalBatchesProcessed,
	}
}

func (s *trialWorkloadSequencer) snapshotState() {
	s.latestCheckpointSequencerSnapshot = s.trialWorkloadSequencerState.deepCopy()
}

func (s *trialWorkloadSequencer) minValidationNeeded() bool {
	if s.minValidationPeriod.Units == 0 {
		return false
	}
	return s.minValidationPeriod.EqualWithinBatch(s.batchesSinceLastVal, s.unitContext)
}

func (s *trialWorkloadSequencer) batchesUntilValNeeded() int {
	if s.minValidationPeriod.Units == 0 {
		return math.MaxInt32
	}
	return s.minValidationPeriod.ToNearestBatch(s.unitContext) - s.batchesSinceLastVal
}

func (s *trialWorkloadSequencer) minCheckpointNeeded() bool {
	if s.minCheckpointPeriod.Units == 0 {
		return false
	}
	return s.minCheckpointPeriod.EqualWithinBatch(s.batchesSinceLastCkpt, s.unitContext)
}

func (s *trialWorkloadSequencer) postUserCancellationCheckpointNeeded() bool {
	return s.userRequestedStop && s.batchesSinceLastCkpt != 0
}

func (s *trialWorkloadSequencer) postValidationCheckpointNeeded() bool {
	return s.needPostValidationCkpt && s.batchesSinceLastCkpt != 0
}

func (s *trialWorkloadSequencer) batchesUntilCkptNeeded() int {
	if s.minCheckpointPeriod.Units == 0 {
		return math.MaxInt32
	}
	return s.minCheckpointPeriod.ToNearestBatch(s.unitContext) - s.batchesSinceLastCkpt
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
