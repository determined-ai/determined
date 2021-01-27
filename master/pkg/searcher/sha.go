package searcher

import (
	"encoding/json"
	"math"
	"sort"

	"github.com/determined-ai/determined/master/pkg/workload"

	"github.com/pkg/errors"

	"github.com/determined-ai/determined/master/pkg/model"
)

type (
	// syncHalvingSearchState is the persistent state for the SHA algorithm.
	syncHalvingSearchState struct {
		Rungs      []*rung                 `json:"rungs"`
		TrialRungs map[model.RequestID]int `json:"trial_rungs"`
		// EarlyExitTrials contains trials that exited early that are still considered in the search.
		EarlyExitTrials map[model.RequestID]bool `json:"early_exit_trials"`
		TrialsCompleted int                      `json:"trials_completed"`
	}

	// syncHalvingSearch implements a search using the synchronous successive halving algorithm
	// (SHA).
	syncHalvingSearch struct {
		defaultSearchMethod
		model.SyncHalvingConfig
		syncHalvingSearchState

		expectedUnits model.Length
	}
)

const shaExitedMetricValue = math.MaxFloat64

func newSyncHalvingSearch(config model.SyncHalvingConfig) SearchMethod {
	rungs := make([]*rung, 0, config.NumRungs)
	expectedUnits := 0
	for id := 0; id < config.NumRungs; id++ {
		compound := math.Pow(config.Divisor, float64(config.NumRungs-id-1))
		unitsNeeded := model.NewLength(
			config.Unit(),
			max(int(float64(config.MaxLength.Units)/compound), 1),
		)
		startTrials := max(int(compound), 1)
		rungs = append(rungs,
			&rung{
				UnitsNeeded: unitsNeeded,
				StartTrials: startTrials,
			},
		)
		if id == 0 {
			expectedUnits += unitsNeeded.Units * startTrials
		} else {
			expectedUnits += (unitsNeeded.Units - rungs[id-1].UnitsNeeded.Units) * startTrials
		}
	}

	multiplier := float64(config.Budget.Units) / float64(expectedUnits)
	expectedUnits = 0
	for id := 0; id < config.NumRungs; id++ {
		cur := rungs[id]
		cur.StartTrials = int(multiplier * float64(cur.StartTrials))
		if id == 0 {
			expectedUnits += cur.UnitsNeeded.Units * cur.StartTrials
		} else {
			prev := rungs[id-1]
			cur.UnitsNeeded = model.NewLength(
				config.Unit(),
				max(cur.UnitsNeeded.Units, prev.UnitsNeeded.Units),
			)
			cur.StartTrials = max(min(cur.StartTrials, prev.StartTrials), 1)
			prev.PromoteTrials = cur.StartTrials
			expectedUnits += (cur.UnitsNeeded.Units - prev.UnitsNeeded.Units) * cur.StartTrials
		}
	}

	return &syncHalvingSearch{
		SyncHalvingConfig: config,
		syncHalvingSearchState: syncHalvingSearchState{
			Rungs:           rungs,
			TrialRungs:      make(map[model.RequestID]int),
			EarlyExitTrials: make(map[model.RequestID]bool),
		},
		expectedUnits: model.NewLength(config.Unit(), expectedUnits),
	}
}

type trialMetric struct {
	RequestID model.RequestID `json:"request_id"`
	Metric    float64         `json:"metric"`
	// fields below used by asha.go.
	Promoted bool `json:"promoted"`
}

// rung describes a set of trials that are to be trained for the same number of units.
type rung struct {
	UnitsNeeded   model.Length  `json:"units_needed"`
	Metrics       []trialMetric `json:"metrics"`
	StartTrials   int           `json:"start_trials"`
	PromoteTrials int           `json:"promote_trials"`
	// field below used by asha.go.
	OutstandingTrials int `json:"outstanding_trials"`
}

// promotions handles bookkeeping of validation metrics and returns a RequestID to promote if
// appropriate.
func (r *rung) promotionsSync(requestID model.RequestID, metric float64) []model.RequestID {
	// Insert the new trial result in the appropriate place in the sorted list.
	insertIndex := sort.Search(
		len(r.Metrics),
		func(i int) bool { return r.Metrics[i].Metric > metric },
	)
	r.Metrics = append(r.Metrics, trialMetric{})
	copy(r.Metrics[insertIndex+1:], r.Metrics[insertIndex:])
	r.Metrics[insertIndex] = trialMetric{
		RequestID: requestID,
		Metric:    metric,
	}

	// If there are enough trials done to definitively promote one, do so. Otherwise, return nil.
	currPromote := len(r.Metrics) + r.PromoteTrials - r.StartTrials
	switch {
	case currPromote <= 0: // Not enough trials completed for any promotions.
		return nil
	case insertIndex < currPromote: // Incoming trial should be promoted.
		return []model.RequestID{requestID}
	default: // Promote next trial in sorted metrics array.
		t := &r.Metrics[currPromote-1]
		return []model.RequestID{t.RequestID}
	}
}

func (s *syncHalvingSearch) initialOperations(ctx context) ([]Operation, error) {
	var ops []Operation
	for trial := 0; trial < s.Rungs[0].StartTrials; trial++ {
		create := NewCreate(
			ctx.rand, sampleAll(ctx.hparams, ctx.rand), model.TrialWorkloadSequencerType)
		ops = append(ops, create)
		ops = append(ops, NewTrain(create.RequestID, s.Rungs[0].UnitsNeeded))
		ops = append(ops, NewValidate(create.RequestID))
	}
	return ops, nil
}

func (s *syncHalvingSearch) validationCompleted(
	ctx context, requestID model.RequestID, validate Validate, metrics workload.ValidationMetrics,
) ([]Operation, error) {
	// Extract the relevant metric as a float.
	metric, err := metrics.Metric(s.Metric)
	if err != nil {
		return nil, err
	}
	if !s.SmallerIsBetter {
		metric *= -1
	}

	return s.promoteSync(ctx, requestID, metric)
}

func (s *syncHalvingSearch) promoteSync(
	ctx context, requestID model.RequestID, metric float64,
) ([]Operation, error) {
	rungIndex := s.TrialRungs[requestID]
	rung := s.Rungs[rungIndex]

	// If the trial has completed the top rung's validation, close the trial and do nothing else.
	if rungIndex == s.NumRungs-1 {
		s.TrialsCompleted++
		if !s.EarlyExitTrials[requestID] {
			return []Operation{NewClose(requestID)}, nil
		}
		return nil, nil
	}

	var ops []Operation
	// Since this is not the top rung, handle promotions if there are any, then close the rung if
	// all trials have finished.
	if toPromote := rung.promotionsSync(requestID, metric); len(toPromote) > 0 {
		for _, promotionID := range toPromote {
			s.TrialRungs[promotionID] = rungIndex + 1
			if !s.EarlyExitTrials[promotionID] {
				unitsNeeded := max(s.Rungs[rungIndex+1].UnitsNeeded.Units-rung.UnitsNeeded.Units, 1)
				ops = append(ops, NewTrain(promotionID, model.NewLength(s.Unit(), unitsNeeded)))
				ops = append(ops, NewValidate(promotionID))
			} else {
				// Since the trial being promoted has already exited and will never finish any more workloads,
				// we should treat it as immediately completing the next rung with the worst possible result.
				// The recursive call is safe because the rung being considered goes up by one each time and
				// there are a finite number of rungs.
				return s.promoteSync(ctx, promotionID, shaExitedMetricValue)
			}
		}
		// Close the unpromoted trials in the rung once all trials in the rung finish.
		if rung.StartTrials < len(rung.Metrics) {
			return nil, errors.Errorf("number of trials exceeded initial trials for rung: %d < %d",
				rung.StartTrials, len(rung.Metrics))
		}
		if len(rung.Metrics) == rung.StartTrials {
			for _, trialMetric := range rung.Metrics[rung.PromoteTrials:] {
				s.TrialsCompleted++
				if !s.EarlyExitTrials[trialMetric.RequestID] {
					ops = append(ops, NewClose(trialMetric.RequestID))
				}
			}
		}
	}
	return ops, nil
}

func (s *syncHalvingSearch) progress(unitsCompleted float64) float64 {
	return math.Min(1, unitsCompleted/float64(s.expectedUnits.Units))
}

func (s *syncHalvingSearch) trialExitedEarly(
	ctx context, requestID model.RequestID, _ workload.ExitedReason,
) ([]Operation, error) {
	s.EarlyExitTrials[requestID] = true
	return s.promoteSync(ctx, requestID, shaExitedMetricValue)
}

func (s *syncHalvingSearch) Snapshot() (json.RawMessage, error) {
	return json.Marshal(s.syncHalvingSearchState)
}

func (s *syncHalvingSearch) Restore(state json.RawMessage) error {
	return json.Unmarshal(state, &s.syncHalvingSearchState)
}

func max(initial int, values ...int) int {
	maxValue := initial
	for _, value := range values {
		if value > maxValue {
			maxValue = value
		}
	}
	return maxValue
}

func min(initial int, values ...int) int {
	minValue := initial
	for _, value := range values {
		if value < minValue {
			minValue = value
		}
	}
	return minValue
}
