package searcher

import (
	"math"
	"sort"

	"github.com/determined-ai/determined/master/pkg/model"
)

// PBTSearch implements population-based training (PBT). See https://arxiv.org/abs/1711.09846 for
// details.
type pbtSearch struct {
	model.PBTConfig
	roundsCompleted int
	metrics         map[RequestID]float64
}

func newPBTSearch(config model.PBTConfig) SearchMethod {
	return &pbtSearch{
		PBTConfig: config,
		metrics:   make(map[RequestID]float64),
	}
}

func (s *pbtSearch) initialOperations(ctx Context) {
	for trial := 0; trial < s.PopulationSize; trial++ {
		trial := ctx.NewTrial(RandomSampler)
		ctx.TrainAndValidate(trial, s.StepsPerRound)
	}
}

func (s *pbtSearch) validationCompleted(
	ctx Context, requestID RequestID, message Workload, metrics ValidationMetrics,
) error {
	// Extract the relevant metric as a float.
	metric, err := metrics.Metric(s.Metric)
	if err != nil {
		return err
	}

	// If we haven't gotten results from the whole population yet, do nothing.
	sign := 1.0
	if !s.SmallerIsBetter {
		sign = -1.0
	}
	s.metrics[requestID] = metric * sign
	if len(s.metrics) < s.PopulationSize {
		return nil
	}

	// We've finished all the rounds, so close everything.
	s.roundsCompleted++
	if s.roundsCompleted >= s.NumRounds {
		for requestID := range s.metrics {
			ctx.CloseTrial(requestID)
		}
		return nil
	}

	// We have all the results and another round to run; now apply truncation to select which trials
	// to close and which to copy.
	numTruncate := int(s.TruncateFraction * float64(s.PopulationSize))

	// Sort trials by metric value.
	trialIDs := make([]RequestID, 0, len(s.metrics))
	for trialID := range s.metrics {
		trialIDs = append(trialIDs, trialID)
	}
	sort.Slice(trialIDs, func(i, j int) bool {
		id1 := trialIDs[i]
		id2 := trialIDs[j]
		m1 := s.metrics[id1]
		m2 := s.metrics[id2]
		if m1 != m2 {
			return m1 < m2
		}
		return id1.Before(id2)
	})
	s.metrics = make(map[RequestID]float64)

	// Close the worst trials.
	for i := len(trialIDs) - numTruncate; i < len(trialIDs); i++ {
		ctx.CloseTrial(trialIDs[i])
	}

	// Checkpoint and copy the best trials.
	for _, requestID := range trialIDs[:numTruncate] {
		old := ctx.Sample(requestID)
		newParams := s.exploreParams(ctx, old)
		newTrial := ctx.NewTrialFromCheckpoint(PreSampled(newParams), requestID)
		ctx.TrainAndValidate(newTrial, s.StepsPerRound)
	}

	// Continue all non-closed trials.
	for _, requestID := range trialIDs[:len(trialIDs)-numTruncate] {
		ctx.TrainAndValidate(requestID, s.StepsPerRound)
	}
	return nil
}

// exploreParams modifies a hyperparameter sample to produce a different one that is "nearby": it
// resamples some parameters anew, and perturbs the rest from their previous values by some
// multiplicative factor.
func (s *pbtSearch) exploreParams(ctx Context, old hparamSample) hparamSample {
	params := make(hparamSample)
	ctx.Hyperparameters().Each(func(name string, sampler model.Hyperparameter) {
		if ctx.Rand().UnitInterval() < s.ResampleProbability {
			params[name] = sampleOne(sampler, ctx.Rand())
		} else {
			val := old[name]
			decrease := ctx.Rand().UnitInterval() < .5
			var multiplier float64
			if decrease {
				multiplier = 1 - s.PerturbFactor
			} else {
				multiplier = 1 + s.PerturbFactor
			}
			switch {
			case sampler.IntHyperparameter != nil:
				h := sampler.IntHyperparameter
				if decrease {
					val = intClamp(int(math.Floor(float64(val.(int))*multiplier)), h.Minval, h.Maxval)
				} else {
					val = intClamp(int(math.Ceil(float64(val.(int))*multiplier)), h.Minval, h.Maxval)
				}
			case sampler.DoubleHyperparameter != nil:
				h := sampler.DoubleHyperparameter
				val = doubleClamp(val.(float64)*multiplier, h.Minval, h.Maxval)
			case sampler.LogHyperparameter != nil:
				h := sampler.LogHyperparameter
				minval := math.Pow(h.Base, h.Minval)
				maxval := math.Pow(h.Base, h.Maxval)
				val = doubleClamp(val.(float64)*multiplier, minval, maxval)
			}
			params[name] = val
		}
	})
	return params
}

func (s *pbtSearch) progress(workloadsCompleted int) float64 {
	stepWorkloads := s.NumRounds * s.PopulationSize * s.StepsPerRound
	validationWorkloads := s.NumRounds * s.PopulationSize
	checkpointWorkloads := (s.NumRounds - 1) * int(s.TruncateFraction*float64(s.PopulationSize))
	return float64(workloadsCompleted) / float64(stepWorkloads+checkpointWorkloads+validationWorkloads)
}

func (s *pbtSearch) trainCompleted(Context, RequestID, Workload) {}
