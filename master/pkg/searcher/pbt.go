package searcher

import (
	"math"
	"sort"

	"github.com/pkg/errors"

	"github.com/determined-ai/determined/master/pkg/model"
)

// PBTSearch implements population-based training (PBT). See https://arxiv.org/abs/1711.09846 for
// details.
type pbtSearch struct {
	defaultSearchMethod
	trialStepPlanner
	model.PBTConfig

	roundsCompleted      int
	metrics              map[RequestID]float64
	trialRoundsCompleted map[RequestID]int
	trialParams          map[RequestID]hparamSample
	waitingOps           map[WorkloadOperation][]Operation

	unitsCompleted model.Length
	expectedUnits  model.Length

	// earlyExitTrials contains trials that exited early that are still considered in the search.
	earlyExitTrials map[RequestID]bool
}

const pbtExitedMetricValue = math.MaxFloat64

func newPBTSearch(config model.PBTConfig, targetBatchesPerStep, recordsPerEpoch int) SearchMethod {
	return &pbtSearch{
		trialStepPlanner:     newTrialStepPlanner(targetBatchesPerStep, recordsPerEpoch),
		PBTConfig:            config,
		metrics:              make(map[RequestID]float64),
		trialRoundsCompleted: make(map[RequestID]int),
		trialParams:          make(map[RequestID]hparamSample),
		waitingOps:           make(map[WorkloadOperation][]Operation),
		earlyExitTrials:      make(map[RequestID]bool),
		unitsCompleted:       model.NewLength(config.LengthPerRound.Kind, 0),
		expectedUnits: model.NewLength(config.LengthPerRound.Kind,
			config.NumRounds*config.PopulationSize*config.LengthPerRound.Units),
	}
}

func (s *pbtSearch) initialOperations(ctx context) ([]Operation, error) {
	var ops []Operation
	for trial := 0; trial < s.PopulationSize; trial++ {
		create := s.create(ctx, sampleAll(ctx.hparams, ctx.rand))
		s.trialParams[create.RequestID] = create.Hparams
		ops = append(ops, create)
		trainVal, trunc := s.trainAndValidate(create.RequestID, s.LengthPerRound)
		s.expectedUnits = s.expectedUnits.Sub(trunc)
		ops = append(ops, trainVal...)
	}
	return ops, nil
}

func (s *pbtSearch) trainCompleted(
	ctx context, requestID RequestID, workload Workload,
) ([]Operation, error) {
	unitsCompletedThisStep := s.unitsFromWorkload(s.unitsCompleted.Kind, workload, requestID)
	s.unitsCompleted = s.unitsCompleted.Add(unitsCompletedThisStep)
	return nil, nil
}

func (s *pbtSearch) validationCompleted(
	ctx context, requestID RequestID, message Workload, metrics ValidationMetrics,
) ([]Operation, error) {
	// Extract the relevant metric as a float.
	rawMetric := metrics.Metrics[s.Metric]
	metric, ok := rawMetric.(float64)
	if !ok {
		return nil, errors.Errorf(
			"selected metric %s is not a scalar float value: %v", s.Metric, rawMetric,
		)
	}

	// If we haven't gotten results from the whole population yet, do nothing.
	sign := 1.0
	if !s.SmallerIsBetter {
		sign = -1.0
	}
	s.metrics[requestID] = metric * sign

	return s.runNewTrials(ctx, requestID, message)
}

func (s *pbtSearch) runNewTrials(
	ctx context, requestID RequestID, message Workload,
) ([]Operation, error) {
	var ops []Operation

	s.trialRoundsCompleted[requestID]++
	if len(s.metrics) < s.PopulationSize {
		return ops, nil
	}

	// We've finished all the rounds, so close everything.
	s.roundsCompleted++
	if s.roundsCompleted >= s.NumRounds {
		for requestID := range s.metrics {
			if !s.earlyExitTrials[requestID] {
				ops = append(ops, s.close(requestID))
			}
		}
		return ops, nil
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
		if !s.earlyExitTrials[trialIDs[i]] {
			// TODO specify the right kind of ID for ops
			ops = append(ops, s.close(trialIDs[i]))
		}
	}

	// Checkpoint and copy the best trials.
	for _, requestID := range trialIDs[:numTruncate] {
		if !s.earlyExitTrials[requestID] {
			// TODO(pickup): context:
			// checkpoints get inserted with a step id,
			// maybe this goes through the step planner, gets curStep - 1, maybe that doesn't
			// always work. hard to know without reading a lot of code.
			checkpoint := s.checkpoint(requestID)
			ops = append(ops, checkpoint)

			origParams := s.trialParams[requestID]
			newParams := s.exploreParams(ctx, origParams)

			create := s.createFromCheckpoint(ctx, newParams, checkpoint)
			s.trialParams[create.RequestID] = newParams

			// The new trial cannot begin until the checkpoint has been completed.
			s.waitingOps[checkpoint] = []Operation{create}
			trainVal, trunc := s.trainAndValidate(create.RequestID, s.LengthPerRound)
			s.expectedUnits = s.expectedUnits.Sub(trunc)
			s.waitingOps[checkpoint] = append(s.waitingOps[checkpoint], trainVal...)
		}
	}

	// Continue all non-closed trials.
	for _, requestID := range trialIDs[:len(trialIDs)-numTruncate] {
		if !s.earlyExitTrials[requestID] {
			trainVal, trunc := s.trainAndValidate(requestID, s.LengthPerRound)
			s.expectedUnits = s.expectedUnits.Sub(trunc)
			ops = append(ops, trainVal...)
		} else {
			s.metrics[requestID] = pbtExitedMetricValue
		}
	}

	return ops, nil
}

// exploreParams modifies a hyperparameter sample to produce a different one that is "nearby": it
// resamples some parameters anew, and perturbs the rest from their previous values by some
// multiplicative factor.
func (s *pbtSearch) exploreParams(ctx context, old hparamSample) hparamSample {
	params := make(hparamSample)
	ctx.hparams.Each(func(name string, sampler model.Hyperparameter) {
		if ctx.rand.UnitInterval() < s.ResampleProbability {
			params[name] = sampleOne(sampler, ctx.rand)
		} else {
			val := old[name]
			decrease := ctx.rand.UnitInterval() < .5
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

func (s *pbtSearch) checkpointCompleted(
	ctx context, requestID RequestID, message Workload, metrics CheckpointMetrics,
) ([]Operation, error) {
	checkpointOp := NewCheckpoint(requestID, message.StepID)
	ops := s.waitingOps[checkpointOp]
	delete(s.waitingOps, checkpointOp)
	return ops, nil
}

func (s *pbtSearch) progress() float64 {
	return float64(s.unitsCompleted.Units) / float64(s.expectedUnits.Units)
}

func (s *pbtSearch) trialExitedEarly(
	ctx context, requestID RequestID, message Workload,
) ([]Operation, error) {
	s.earlyExitTrials[requestID] = true
	s.metrics[requestID] = pbtExitedMetricValue
	return s.runNewTrials(ctx, requestID, message)
}
