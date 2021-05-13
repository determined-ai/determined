package searcher

import (
	"encoding/json"
	"math"
	"sort"

	"github.com/determined-ai/determined/master/pkg/workload"

	"github.com/determined-ai/determined/master/pkg/model"
)

// PBTSearch implements population-based training (PBT). See https://arxiv.org/abs/1711.09846 for
// details.
type (
	pbtSearchState struct {
		RoundsCompleted      int                              `json:"rounds_completed"`
		Metrics              map[model.RequestID]float64      `json:"metrics"`
		TrialRoundsCompleted map[model.RequestID]int          `json:"trial_rounds_completed"`
		TrialParams          map[model.RequestID]hparamSample `json:"trial_params"`

		// EarlyExitTrials contains trials that exited early that are still considered in the search.
		EarlyExitTrials map[model.RequestID]bool `json:"early_exit_trials"`

		SearchMethodType SearchMethodType `json:"search_method_type"`
	}

	pbtSearch struct {
		defaultSearchMethod
		model.PBTConfig
		pbtSearchState
	}
)

const pbtExitedMetricValue = math.MaxFloat64

func newPBTSearch(config model.PBTConfig) SearchMethod {
	return &pbtSearch{
		PBTConfig: config,
		pbtSearchState: pbtSearchState{
			Metrics:              make(map[model.RequestID]float64),
			TrialRoundsCompleted: make(map[model.RequestID]int),
			TrialParams:          make(map[model.RequestID]hparamSample),
			EarlyExitTrials:      make(map[model.RequestID]bool),
			SearchMethodType:     PBTSearch,
		},
	}
}

func (s *pbtSearch) Snapshot() (json.RawMessage, error) {
	return json.Marshal(s.pbtSearchState)
}

func (s *pbtSearch) Restore(state json.RawMessage) error {
	if state == nil {
		return nil
	}
	return json.Unmarshal(state, &s.pbtSearchState)
}

func (s *pbtSearch) initialOperations(ctx context) ([]Operation, error) {
	var ops []Operation
	for trial := 0; trial < s.PopulationSize; trial++ {
		create := NewCreate(
			ctx.rand, sampleAll(ctx.hparams, ctx.rand), model.TrialWorkloadSequencerType)
		s.TrialParams[create.RequestID] = create.Hparams
		ops = append(ops, create)
		ops = append(ops, NewValidateAfter(create.RequestID, s.LengthPerRound))
	}
	return ops, nil
}

func (s *pbtSearch) validationCompleted(
	ctx context, requestID model.RequestID, metric float64,
) ([]Operation, error) {
	// If we haven't gotten results from the whole population yet, do nothing.
	sign := 1.0
	if !s.SmallerIsBetter {
		sign = -1.0
	}
	s.Metrics[requestID] = metric * sign

	return s.runNewTrials(ctx, requestID)
}

func (s *pbtSearch) runNewTrials(ctx context, requestID model.RequestID) ([]Operation, error) {
	var ops []Operation

	s.TrialRoundsCompleted[requestID]++
	if len(s.Metrics) < s.PopulationSize {
		return ops, nil
	}

	// We've finished all the rounds, so close everything.
	s.RoundsCompleted++
	if s.RoundsCompleted >= s.NumRounds {
		for requestID := range s.Metrics {
			if !s.EarlyExitTrials[requestID] {
				ops = append(ops, NewClose(requestID))
			}
		}
		return ops, nil
	}

	// We have all the results and another round to run; now apply truncation to select which trials
	// to close and which to copy.
	numTruncate := int(s.TruncateFraction * float64(s.PopulationSize))

	// Sort trials by metric value.
	trialIDs := make([]model.RequestID, 0, len(s.Metrics))
	for trialID := range s.Metrics {
		trialIDs = append(trialIDs, trialID)
	}
	sort.Slice(trialIDs, func(i, j int) bool {
		id1 := trialIDs[i]
		id2 := trialIDs[j]
		m1 := s.Metrics[id1]
		m2 := s.Metrics[id2]
		if m1 != m2 {
			return m1 < m2
		}
		return id1.Before(id2)
	})
	s.Metrics = make(map[model.RequestID]float64)

	// Close the worst trials.
	for i := len(trialIDs) - numTruncate; i < len(trialIDs); i++ {
		if !s.EarlyExitTrials[trialIDs[i]] {
			// TODO specify the right kind of ID for ops
			ops = append(ops, NewClose(trialIDs[i]))
		}
	}

	// Checkpoint and copy the best trials.
	for _, requestID := range trialIDs[:numTruncate] {
		if !s.EarlyExitTrials[requestID] {
			origParams := s.TrialParams[requestID]
			newParams := s.exploreParams(ctx, origParams)

			create := NewCreateFromCheckpoint(
				ctx.rand, newParams, requestID, model.TrialWorkloadSequencerType)
			s.TrialParams[create.RequestID] = newParams

			ops = append(ops,
				create,
				NewValidateAfter(create.RequestID, s.LengthPerRound))
		}
	}

	// Continue all non-closed trials.
	for _, requestID := range trialIDs[:len(trialIDs)-numTruncate] {
		if !s.EarlyExitTrials[requestID] {
			ops = append(ops, NewValidateAfter(
				requestID, s.LengthPerRound.MultInt(s.TrialRoundsCompleted[requestID]+1)))
		} else {
			s.Metrics[requestID] = pbtExitedMetricValue
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

func (s *pbtSearch) progress(trialProgress map[model.RequestID]model.PartialUnits) float64 {
	unitsCompleted := sumTrialLengths(trialProgress)
	unitsExpected := s.LengthPerRound.MultInt(s.PopulationSize).MultInt(s.NumRounds).Units
	return float64(unitsCompleted) / float64(unitsExpected)
}

func (s *pbtSearch) trialExitedEarly(
	ctx context, requestID model.RequestID, exitedReason workload.ExitedReason,
) ([]Operation, error) {
	s.EarlyExitTrials[requestID] = true
	s.Metrics[requestID] = pbtExitedMetricValue
	return s.runNewTrials(ctx, requestID)
}
