package searcher

import (
	"encoding/json"
	"math"
	"sort"

	"github.com/determined-ai/determined/master/pkg/workload"

	"github.com/pkg/errors"

	"github.com/determined-ai/determined/master/pkg/model"
)

// PBTSearch implements population-based training (PBT). See https://arxiv.org/abs/1711.09846 for
// details.
type (
	pbtSearchState struct {
		RoundsCompleted      int                               `json:"rounds_completed"`
		Metrics              map[model.RequestID]float64       `json:"metrics"`
		TrialRoundsCompleted map[model.RequestID]int           `json:"trial_rounds_completed"`
		TrialParams          map[model.RequestID]hparamSample  `json:"trial_params"`
		WaitingCheckpoints   map[model.RequestID]OperationList `json:"waiting_checkpoints"`

		// EarlyExitTrials contains trials that exited early that are still considered in the search.
		EarlyExitTrials map[model.RequestID]bool `json:"early_exit_trials"`
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
			WaitingCheckpoints:   make(map[model.RequestID]OperationList),
			EarlyExitTrials:      make(map[model.RequestID]bool),
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
		ops = append(ops, NewTrain(create.RequestID, s.LengthPerRound))
		ops = append(ops, NewValidate(create.RequestID))
	}
	return ops, nil
}

func (s *pbtSearch) validationCompleted(
	ctx context, requestID model.RequestID, validate Validate, metrics workload.ValidationMetrics,
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
			checkpoint := NewCheckpoint(requestID)
			ops = append(ops, checkpoint)

			origParams := s.TrialParams[requestID]
			newParams := s.exploreParams(ctx, origParams)

			create := NewCreateFromCheckpoint(
				ctx.rand, newParams, checkpoint, model.TrialWorkloadSequencerType)
			s.TrialParams[create.RequestID] = newParams

			// The new trial cannot begin until the checkpoint has been completed.
			s.WaitingCheckpoints[checkpoint.RequestID] = []Operation{create}
			s.WaitingCheckpoints[checkpoint.RequestID] = append(s.WaitingCheckpoints[checkpoint.RequestID],
				NewTrain(create.RequestID, s.LengthPerRound), NewValidate(create.RequestID))
		}
	}

	// Continue all non-closed trials.
	for _, requestID := range trialIDs[:len(trialIDs)-numTruncate] {
		if !s.EarlyExitTrials[requestID] {
			ops = append(ops, NewTrain(requestID, s.LengthPerRound), NewValidate(requestID))
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

func (s *pbtSearch) checkpointCompleted(
	ctx context, requestID model.RequestID, checkpoint Checkpoint, metrics workload.CheckpointMetrics,
) ([]Operation, error) {
	ops := s.WaitingCheckpoints[checkpoint.RequestID]
	delete(s.WaitingCheckpoints, checkpoint.RequestID)
	return ops, nil
}

func (s *pbtSearch) progress(unitsCompleted float64) float64 {
	return unitsCompleted / float64(
		s.LengthPerRound.MultInt(s.PopulationSize).MultInt(s.NumRounds).Units)
}

func (s *pbtSearch) trialExitedEarly(
	ctx context, requestID model.RequestID, exitedReason workload.ExitedReason,
) ([]Operation, error) {
	s.EarlyExitTrials[requestID] = true
	s.Metrics[requestID] = pbtExitedMetricValue
	return s.runNewTrials(ctx, requestID)
}
