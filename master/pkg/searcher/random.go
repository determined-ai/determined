package searcher

import (
	"github.com/determined-ai/determined/master/pkg/model"
)

// randomSearch corresponds to the standard random search method. Each random trial configuration
// is trained for the specified number of steps, and then validation metrics are computed.
type randomSearch struct {
	model.RandomConfig
}

func newRandomSearch(config model.RandomConfig) SearchMethod {
	return &randomSearch{RandomConfig: config}
}

func newSingleSearch(config model.SingleConfig) SearchMethod {
	return &randomSearch{RandomConfig: model.RandomConfig{MaxTrials: 1, MaxSteps: config.MaxSteps}}
}

func (s *randomSearch) initialOperations(ctx Context) {
	for i := 0; i < s.MaxTrials; i++ {
		trial := ctx.NewTrial(RandomSampler)
		ctx.TrainAndValidate(trial, s.MaxSteps)
		ctx.CloseTrial(trial)
	}
}

func (s *randomSearch) progress(workloadsCompleted int) float64 {
	return float64(workloadsCompleted) / float64((s.MaxSteps+1)*s.MaxTrials)
}

func (s *randomSearch) trainCompleted(Context, RequestID, Workload) {}
func (s *randomSearch) validationCompleted(Context, RequestID, Workload, ValidationMetrics) error {
	return nil
}
