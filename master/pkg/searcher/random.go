package searcher

import (
	"github.com/determined-ai/determined/master/pkg/model"
	"github.com/determined-ai/determined/master/pkg/workload"
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
	var ops []Operation
	for trial := 0; trial < s.MaxTrials; trial++ {
		create := NewCreate(ctx.rand, sampleAll(ctx.hparams, ctx.rand), model.TrialWorkloadSequencerType)
		ops = append(ops, create)
		ops = append(ops, NewTrain(create.RequestID, s.MaxLength))
		ops = append(ops, NewValidate(create.RequestID))
		ops = append(ops, NewClose(create.RequestID))
	}
	return ops, nil
}

func (s *randomSearch) progress(unitsCompleted float64) float64 {
	return unitsCompleted / float64(s.MaxLength.MultInt(s.MaxTrials).Units)
}

// trialExitedEarly does nothing since random does not take actions based on
// search status or progress.
func (s *randomSearch) trialExitedEarly(
	context, RequestID, workload.ExitedReason,
) ([]Operation, error) {
	return nil, nil
}
