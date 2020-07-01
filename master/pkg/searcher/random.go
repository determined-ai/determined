package searcher

import (
	"github.com/determined-ai/determined/master/pkg/model"
)

// randomSearch corresponds to the standard random search method. Each random trial configuration
// is trained for the specified number of steps, and then validation metrics are computed.
type randomSearch struct {
	defaultSearchMethod
	trialStepPlanner
	model.RandomConfig

	unitsCompleted model.Length
	expectedUnits  model.Length
}

func newRandomSearch(config model.RandomConfig, targetBatchesPerStep, recordsPerEpoch int) SearchMethod {
	return &randomSearch{
		trialStepPlanner: newTrialStepPlanner(targetBatchesPerStep, recordsPerEpoch),
		RandomConfig:     config,
		unitsCompleted:   model.NewLength(config.MaxLength.Kind, 0),
		expectedUnits:    config.MaxLength.MultInt(config.MaxTrials),
	}
}

func newSingleSearch(config model.SingleConfig, targetBatchesPerStep, recordsPerEpoch int) SearchMethod {
	return &randomSearch{
		trialStepPlanner: newTrialStepPlanner(targetBatchesPerStep, recordsPerEpoch),
		RandomConfig:     model.RandomConfig{MaxTrials: 1, MaxLength: config.MaxLength},
		unitsCompleted:   model.NewLength(config.MaxLength.Kind, 0),
		expectedUnits:    config.MaxLength,
	}
}

func (s *randomSearch) initialOperations(ctx context) ([]Operation, error) {
	var operations []Operation
	for trial := 0; trial < s.MaxTrials; trial++ {
		create := s.create(ctx, sampleAll(ctx.hparams, ctx.rand))
		operations = append(operations, create)
		trainVal, trunc := s.trainAndValidate(create.RequestID, s.MaxLength)
		s.expectedUnits = s.expectedUnits.Sub(trunc)
		operations = append(operations, trainVal...)
		operations = append(operations, s.close(create.RequestID))
	}
	return operations, nil
}

func (s *randomSearch) trainCompleted(
	ctx context, requestID RequestID, workload Workload,
) ([]Operation, error) {
	unitsCompletedThisStep := s.unitsFromWorkload(s.unitsCompleted.Kind, workload, requestID)
	s.unitsCompleted = s.unitsCompleted.Add(unitsCompletedThisStep)
	return nil, nil
}

func (s *randomSearch) progress() float64 {
	return float64(s.unitsCompleted.Units) / float64(s.expectedUnits.Units)
}

// trialExitedEarly does nothing since random does not take actions based on
// search status or progress.
func (s *randomSearch) trialExitedEarly(
	ctx context, requestID RequestID, message Workload,
) ([]Operation, error) {
	return nil, nil
}
