package searcher

import (
	"fmt"
	"math"
	"sort"

	"github.com/pkg/errors"

	"github.com/determined-ai/determined/master/pkg/model"
)

// OperationPlanner performs a one-to-many mapping of search method operations to workload
// operations and the reverse mapping to turn completed workloads back into their corresponding
// search method operation to inform the search method about. Thie purpose is so the tbe search
// method agonostic of scheduling and fault-tolerance concerns.
type OperationPlanner struct {
	// configuration
	unit                 model.Unit
	targetBatchesPerStep int
	recordsPerEpoch      int
	minValidationPeriod  model.Length
	minCheckpointPeriod  model.Length
	checkpointPolicy     string

	// trial state
	trialGlobalBatchSizes map[RequestID]int
	stepCounts            map[RequestID]int
	trialOpSequences      map[RequestID]opSequence
	unitsSinceValidation  map[RequestID]model.Length
	unitsSinceCheckpoint  map[RequestID]model.Length

	// searcher state
	totalUnitsCompleted model.Length
}

type workloadToSearcherOp struct {
	workload   WorkloadOperation
	searcherOp Operation
}

type opSequence []workloadToSearcherOp

func insertSorted(seq opSequence, pairs ...workloadToSearcherOp) opSequence {
	tmp := append(seq, pairs...)
	sort.Sort(tmp)
	return dedupSorted(tmp)
}

func dedupSorted(seq opSequence) opSequence {
	var dupes []int
	var prev WorkloadOperation
	for i, op := range seq {
		if op.workload == prev {
			dupes = append(dupes, i)
		}
		prev = op.workload
	}

	for i := len(dupes) - 1; i >= 0; i-- {
		dupe := dupes[i]
		seq = append(seq[:dupe], seq[dupe+1:]...)
	}
	return seq
}

func (o opSequence) Len() int {
	return len(o)
}

func (o opSequence) Swap(i, j int) {
	o[i], o[j] = o[j], o[i]
}

func (o opSequence) Less(i, j int) bool {
	// Less returns if op[i] < op[j], ordered as the trial_workload_sequencer would order workloads.
	if o[i].workload.StepID == o[j].workload.StepID {
		switch o[i].workload.Kind {
		case RunStep:
			if o[j].workload.Kind == ComputeValidationMetrics ||
				o[j].workload.Kind == CheckpointModel {
				return true
			}
			return false
		case ComputeValidationMetrics:
			if o[j].workload.Kind == CheckpointModel {
				return true
			}
			return false
		case CheckpointModel:
			return false
		}
	}
	return o[i].workload.StepID < o[j].workload.StepID
}

// NewOperationPlanner creates an new operation planner with the given configurations.
func NewOperationPlanner(
	batchesPerStep, recordsPerEpoch int,
	minValidationPeriod, minCheckpointPeriod model.Length, checkpointPolicy string,
) OperationPlanner {
	return OperationPlanner{
		unit:                 minCheckpointPeriod.Unit,
		targetBatchesPerStep: batchesPerStep,
		recordsPerEpoch:      recordsPerEpoch,
		minValidationPeriod:  minValidationPeriod,
		minCheckpointPeriod:  minCheckpointPeriod,
		checkpointPolicy:     checkpointPolicy,

		trialGlobalBatchSizes: make(map[RequestID]int),
		stepCounts:            make(map[RequestID]int),
		trialOpSequences:      make(map[RequestID]opSequence),
		unitsSinceValidation:  make(map[RequestID]model.Length),
		unitsSinceCheckpoint:  make(map[RequestID]model.Length),

		totalUnitsCompleted: model.NewLength(minCheckpointPeriod.Unit, 0),
	}
}

// Plan plans the given operations.
func (p *OperationPlanner) Plan(ops []Operation) (plannedOps []Operation) {
	for _, op := range ops {
		switch tOp := op.(type) {
		case Create:
			p.create(tOp)
			plannedOps = append(plannedOps, tOp)
		case Train:
			workloads := p.train(tOp.RequestID, tOp.Length)
			p.sequence(tOp.RequestID, tOp, workloads, RunStep)
			plannedOps = append(plannedOps, workloads...)
		case Validate:
			workloads := p.validate(tOp.RequestID)
			p.sequence(tOp.RequestID, tOp, workloads, ComputeValidationMetrics)
			plannedOps = append(plannedOps, workloads...)
		case Checkpoint:
			workloads := p.checkpoint(tOp.RequestID)
			p.sequence(tOp.RequestID, tOp, workloads, CheckpointModel)
			plannedOps = append(plannedOps, workloads...)
		default:
			plannedOps = append(plannedOps, tOp)
		}
	}
	return plannedOps
}

func (p *OperationPlanner) create(create Create) {
	p.trialGlobalBatchSizes[create.RequestID] = create.Hparams.GlobalBatchSize()
	p.stepCounts[create.RequestID] = 0
	p.unitsSinceValidation[create.RequestID] = model.NewLength(p.unit, 0)
	p.unitsSinceCheckpoint[create.RequestID] = model.NewLength(p.unit, 0)
}

func (p *OperationPlanner) train(requestID RequestID, unitsNeeded model.Length) (ops []Operation) {
	batchesNeeded, trunc := p.unitsToBatches(requestID, unitsNeeded)
	p.totalUnitsCompleted = p.totalUnitsCompleted.Add(trunc)
	var batchesThisStep int
	for curBatches := 0; curBatches < batchesNeeded; curBatches += batchesThisStep {
		batchesLeft := batchesNeeded - curBatches
		batchesTilVal := p.batchesUntilValNeeded(requestID)
		batchesTilCkpt := p.batchesUntilCkptNeeded(requestID)
		batchesThisStep = min(
			batchesLeft,
			batchesTilVal,
			batchesTilCkpt,
			p.targetBatchesPerStep,
		)
		p.stepCounts[requestID]++
		ops = append(ops, p.trainStep(requestID, batchesThisStep))
		if batchesThisStep == batchesTilVal {
			ops = append(ops, p.validate(requestID)...)
		}
		if batchesThisStep == batchesTilCkpt {
			ops = append(ops, p.checkpoint(requestID)...)
		}
	}
	return ops
}

func (p *OperationPlanner) trainStep(requestID RequestID, numBatches int) WorkloadOperation {
	unitsThisStep := p.unitsFromBatches(requestID, numBatches)
	p.unitsSinceValidation[requestID] = p.unitsSinceValidation[requestID].Add(unitsThisStep)
	p.unitsSinceCheckpoint[requestID] = p.unitsSinceCheckpoint[requestID].Add(unitsThisStep)
	return NewTrainWorkload(requestID, p.stepCounts[requestID], numBatches)
}

func (p *OperationPlanner) validate(requestID RequestID) (ops []Operation) {
	if p.unitsSinceValidation[requestID].Units == 0 {
		return nil
	}
	p.unitsSinceValidation[requestID] = model.NewLength(p.unit, 0)
	ops = append(ops, NewValidateWorkload(requestID, p.stepCounts[requestID]))
	return ops
}

func (p *OperationPlanner) checkpoint(requestID RequestID) (ops []Operation) {
	if p.unitsSinceCheckpoint[requestID].Units == 0 {
		return nil
	}
	p.unitsSinceCheckpoint[requestID] = model.NewLength(p.unit, 0)
	ops = append(ops, NewCheckpointWorkload(requestID, p.stepCounts[requestID]))
	return ops
}

func (p *OperationPlanner) sequence(
	requestID RequestID, searcherOp Operation, ops []Operation, completedByLast Kind,
) {
	if ops != nil {
		var pairs []workloadToSearcherOp
		for _, op := range ops {
			pairs = append(pairs, workloadToSearcherOp{op.(WorkloadOperation), nil})
		}
		p.trialOpSequences[requestID] = insertSorted(p.trialOpSequences[requestID], pairs...)
	}

	if searcherOp != nil {
		for i := len(p.trialOpSequences[requestID]) - 1; i >= 0; i-- {
			if p.trialOpSequences[requestID][i].workload.Kind == completedByLast {
				p.trialOpSequences[requestID][i].searcherOp = searcherOp
				return
			}
		}
	}
}

// WorkloadCompleted collates workloads back into search method operations, through multiple calls.
func (p *OperationPlanner) WorkloadCompleted(
	requestID RequestID, workload Workload, isBestValidation bool,
) (completedSearcherOp Operation, checkpoints []Operation, err error) {
	if p.trialOpSequences[requestID].Len() == 0 {
		return nil, nil, errors.New("call to WorkloadComplete with no expected workloads left")
	}
	expected, left := p.trialOpSequences[requestID][0], p.trialOpSequences[requestID][1:]
	p.trialOpSequences[requestID] = left
	if err != nil {
		return nil, nil, errors.Wrap(err, "error retreiving expected workload")
	}

	if !expected.workload.equals(workload) {
		return nil, nil, fmt.Errorf("received %s but expected operation %s", workload, expected.workload)
	}

	unitsThisWorkload := p.unitsFromBatches(requestID, workload.NumBatches)
	p.totalUnitsCompleted = p.totalUnitsCompleted.Add(unitsThisWorkload)
	if workload.Kind == ComputeValidationMetrics {
		switch p.checkpointPolicy {
		case model.AllCheckpointPolicy:
			checkpoints = append(checkpoints, NewCheckpointWorkload(requestID, workload.StepID))
			p.sequence(requestID, nil, checkpoints, CheckpointModel)
		case model.BestCheckpointPolicy:
			if isBestValidation {
				checkpoints = append(checkpoints, NewCheckpointWorkload(requestID, workload.StepID))
				p.sequence(requestID, nil, checkpoints, CheckpointModel)
			}
		}
	}

	return expected.searcherOp, checkpoints, nil
}

func (p *OperationPlanner) batchesUntilValNeeded(requestID RequestID) int {
	if p.minValidationPeriod.Units == 0 {
		return math.MaxInt32
	}
	unitsUntilVal := p.minValidationPeriod.Sub(p.unitsSinceValidation[requestID])
	batchesUntilVal, _ := p.unitsToBatches(requestID, unitsUntilVal)
	return batchesUntilVal
}

func (p *OperationPlanner) batchesUntilCkptNeeded(requestID RequestID) int {
	if p.minCheckpointPeriod.Units == 0 {
		return math.MaxInt32
	}
	unitsUntilCkpt := p.minCheckpointPeriod.Sub(p.unitsSinceCheckpoint[requestID])
	batchesUntilCkpt, _ := p.unitsToBatches(requestID, unitsUntilCkpt)
	return batchesUntilCkpt
}

// unitsFromBatches determines the number of units completed during a given workload.
func (p OperationPlanner) unitsFromBatches(requestID RequestID, batches int) model.Length {
	switch p.unit {
	case model.Records:
		return model.NewLengthInRecords(batches * p.trialGlobalBatchSizes[requestID])
	case model.Batches:
		return model.NewLengthInBatches(batches)
	case model.Epochs:
		// Round up because if we ran a partial epoch, we always _meant_ to run a full one and
		// truncated on the nearest batch.
		numRecords := batches * p.trialGlobalBatchSizes[requestID]
		numEpochs := math.Ceil(float64(numRecords) / float64(p.recordsPerEpoch))
		return model.NewLengthInEpochs(int(numEpochs))
	default:
		panic(fmt.Sprintf("invalid in OperationPlanner: %s", p.unit))
	}
}

// unitsToBatches converts a training length to the nearest batch, potentially truncating some units
// if they are provided as records or epochs.
func (p OperationPlanner) unitsToBatches(
	requestID RequestID, l model.Length,
) (batches int, truncated model.Length) {
	globalBatchSize := p.trialGlobalBatchSizes[requestID]
	switch l.Unit {
	case model.Records:
		return l.Units / globalBatchSize, model.NewLengthInRecords(l.Units % globalBatchSize)
	case model.Batches:
		return l.Units, model.NewLengthInBatches(0)
	case model.Epochs:
		return (l.Units * p.recordsPerEpoch) / globalBatchSize, model.NewLengthInEpochs(0)
	default:
		panic(fmt.Sprintf("invalid Unit passed to unitsToBatches %s", l.Unit))
	}
}
