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

// trialExitedEarly creates a new trial upon receiving an InvalidHP workload.
// Otherwise, it does nothing since actions are not taken based on search status.
func (s *randomSearch) trialExitedEarly(
	ctx context, requestID model.RequestID, exitedReason workload.ExitedReason,
) ([]Operation, error) {
	if exitedReason == workload.InvalidHP {
		var ops []Operation
		create := NewCreate(ctx.rand, sampleAll(ctx.hparams, ctx.rand), model.TrialWorkloadSequencerType)
		ops = append(ops, create)
		ops = append(ops, NewTrain(create.RequestID, s.MaxLength))
		ops = append(ops, NewValidate(create.RequestID))
		ops = append(ops, NewClose(create.RequestID))
		return ops, nil
	}
	return nil, nil
}
