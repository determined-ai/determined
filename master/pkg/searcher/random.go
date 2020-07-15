package searcher

import (
	"github.com/determined-ai/determined/master/pkg/model"
)

// randomSearch corresponds to the standard random search method. Each random trial configuration
// is trained for the specified number of steps, and then validation metrics are computed.
type randomSearch struct {
	defaultSearchMethod
	model.RandomConfig
}

func newRandomSearch(config model.RandomConfig) SearchMethod {
	return &randomSearch{RandomConfig: config}
}

func newSingleSearch(config model.SingleConfig) SearchMethod {
	return &randomSearch{
		RandomConfig: model.RandomConfig{MaxTrials: 1, MaxLength: config.MaxLength},
	}
}

func (s *randomSearch) initialOperations(ctx context) ([]Operation, error) {
	var operations []Operation
	for trial := 0; trial < s.MaxTrials; trial++ {
		create := NewCreate(ctx.rand, sampleAll(ctx.hparams, ctx.rand), model.TrialWorkloadSequencerType)
		operations = append(operations, create)
		operations = append(operations, NewTrain(create.RequestID, s.MaxLength))
		operations = append(operations, NewValidate(create.RequestID))
		operations = append(operations, NewClose(create.RequestID))
	}
	return operations, nil
}

func (s *randomSearch) progress(unitsCompleted model.Length) float64 {
	return float64(unitsCompleted.Units) / float64(s.MaxLength.MultInt(s.MaxTrials).Units)
}

// trialExitedEarly does nothing since random does not take actions based on
// search status or progress.
func (s *randomSearch) trialExitedEarly(context, RequestID) ([]Operation, error) {
	return nil, nil
}
