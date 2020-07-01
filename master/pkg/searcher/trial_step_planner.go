package searcher

import (
	"fmt"
	"math"

	"github.com/determined-ai/determined/master/pkg/model"
)

// trialStepPlanner manages all the context around how requested "units" actually turn into
// workloads that the trial runs. probably can be merged with trial_workload_sequencer.go once
// that code is pulled up to the experiment level.
type trialStepPlanner struct {
	targetBatchesPerStep  int
	recordsPerEpoch       int
	trialGlobalBatchSizes map[RequestID]int
	stepCounts            map[RequestID]int
}

func newTrialStepPlanner(targetBatchesPerStep, recordsPerEpoch int) trialStepPlanner {
	return trialStepPlanner{
		targetBatchesPerStep:  targetBatchesPerStep,
		recordsPerEpoch:       recordsPerEpoch,
		trialGlobalBatchSizes: make(map[RequestID]int),
		stepCounts:            make(map[RequestID]int),
	}
}

func (s *trialStepPlanner) create(ctx context, sample hparamSample) Create {
	op := NewCreate(ctx.rand, sample, model.TrialWorkloadSequencerType)
	s.trialGlobalBatchSizes[op.RequestID] = op.Hparams.GlobalBatchSize()
	s.stepCounts[op.RequestID] = 0
	return op
}

func (s *trialStepPlanner) createFromCheckpoint(
	ctx context, sample hparamSample, checkpoint WorkloadOperation,
) Create {
	op := NewCreateFromCheckpoint(
		ctx.rand, sample, checkpoint.RequestID, checkpoint.StepID, model.TrialWorkloadSequencerType)
	s.trialGlobalBatchSizes[op.RequestID] = op.Hparams.GlobalBatchSize()
	s.stepCounts[op.RequestID] = 0
	return op
}

func (s *trialStepPlanner) trainAndValidate(
	requestID RequestID,
	unitsNeeded model.Length,
) (ops []Operation, truncated model.Length) {
	batchesNeeded, truncated := s.unitsToBatches(unitsNeeded, requestID)
	for curBatches := 0; curBatches < batchesNeeded; curBatches += s.targetBatchesPerStep {
		batchesLeft := batchesNeeded - curBatches
		batchesThisStep := min(batchesLeft, s.targetBatchesPerStep)
		s.stepCounts[requestID]++
		ops = append(ops, NewTrain(requestID, s.stepCounts[requestID], batchesThisStep))
	}
	ops = append(ops, NewValidate(requestID, s.stepCounts[requestID]))
	return ops, truncated
}

func (s *trialStepPlanner) checkpoint(requestID RequestID) WorkloadOperation {
	return NewCheckpoint(requestID, s.stepCounts[requestID])
}

func (s *trialStepPlanner) close(requestID RequestID) Close {
	return NewClose(requestID)
}

// unitsFromWorkload determines the number of units completed during a given workload.
func (s trialStepPlanner) unitsFromWorkload(
	kind model.Kind, workload Workload, requestID RequestID,
) model.Length {
	switch kind {
	case model.Records:
		return model.NewLengthInRecords(workload.NumBatches * s.trialGlobalBatchSizes[requestID])
	case model.Batches:
		return model.NewLengthInBatches(workload.NumBatches)
	case model.Epochs:
		// Round up because if we ran a partial epoch, we always _meant_ to run a full one and
		// truncated on the nearest batch.
		numRecords := workload.NumBatches * s.trialGlobalBatchSizes[requestID]
		numEpochs := math.Ceil(float64(numRecords) / float64(s.recordsPerEpoch))
		return model.NewLengthInEpochs(int(numEpochs))
	default:
		panic(fmt.Sprintf("invalid Kind passed to unitsFromStep %s", kind))
	}
}

// unitsToBatches converts a training length to the nearest batch. This function is necessary
// because the harness expects RUN_STEP's to contain the number of batches to train for, so searcher
// training length must be rounded to the nearest batch before they are sent and partial batches are
// hard.
func (s trialStepPlanner) unitsToBatches(
	l model.Length, requestID RequestID,
) (batches int, truncated model.Length) {
	globalBatchSize := s.trialGlobalBatchSizes[requestID]
	switch l.Kind {
	case model.Records:
		return l.Units / globalBatchSize, model.NewLengthInRecords(l.Units % globalBatchSize)
	case model.Batches:
		return l.Units, model.NewLengthInBatches(0)
	case model.Epochs:
		return (l.Units * s.recordsPerEpoch) / globalBatchSize, model.NewLengthInEpochs(0)
	default:
		panic(fmt.Sprintf("invalid Kind passed to unitsToBatches %s", l.Kind))
	}
}
