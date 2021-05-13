package searcher

import (
	"encoding/json"
	"math"
	"sort"

	"github.com/determined-ai/determined/master/pkg/workload"

	"github.com/determined-ai/determined/master/pkg/model"
)

// AsyncHalvingSearch implements a search using the asynchronous successive halving algorithm
// (ASHA). The experiment will run until the target number of trials have been completed
// in the bottom rung and no further promotions can be made to higher rungs.
type (
	asyncHalvingSearchState struct {
		Rungs      []*rung                 `json:"rungs"`
		TrialRungs map[model.RequestID]int `json:"trial_rungs"`
		// EarlyExitTrials contains trials that exited early that are still considered in the search.
		EarlyExitTrials  map[model.RequestID]bool `json:"early_exit_trials"`
		ClosedTrials     map[model.RequestID]bool `json:"closed_trials"`
		TrialsCompleted  int                      `json:"trials_completed"`
		InvalidTrials    int                      `json:"invalid_trials"`
		PendingTrials    int                      `json:"pending_trials"`
		SearchMethodType SearchMethodType         `json:"search_method_type"`
	}

	asyncHalvingSearch struct {
		model.AsyncHalvingConfig
		asyncHalvingSearchState
	}

	trialMetric struct {
		RequestID model.RequestID `json:"request_id"`
		Metric    float64         `json:"metric"`
		// fields below used by asha.go.
		Promoted bool `json:"promoted"`
	}

	// rung describes a set of trials that are to be trained for the same number of units.
	rung struct {
		UnitsNeeded   model.Length  `json:"units_needed"`
		Metrics       []trialMetric `json:"metrics"`
		StartTrials   int           `json:"start_trials"`
		PromoteTrials int           `json:"promote_trials"`
		// field below used by asha.go.
		OutstandingTrials int `json:"outstanding_trials"`
	}
)

const ashaExitedMetricValue = math.MaxFloat64

func newAsyncHalvingSearch(config model.AsyncHalvingConfig) SearchMethod {
	rungs := make([]*rung, 0, config.NumRungs)
	unitsNeeded := 0
	for id := 0; id < config.NumRungs; id++ {
		// We divide the MaxLength by downsampling rate to get the target units
		// for a rung.
		downsamplingRate := math.Pow(config.Divisor, float64(config.NumRungs-id-1))
		unitsNeeded += max(int(float64(config.MaxLength.Units)/downsamplingRate), 1)
		rungs = append(rungs, &rung{UnitsNeeded: model.NewLength(config.Unit(), unitsNeeded)})
	}

	return &asyncHalvingSearch{
		AsyncHalvingConfig: config,
		asyncHalvingSearchState: asyncHalvingSearchState{
			Rungs:            rungs,
			TrialRungs:       make(map[model.RequestID]int),
			EarlyExitTrials:  make(map[model.RequestID]bool),
			ClosedTrials:     make(map[model.RequestID]bool),
			SearchMethodType: ASHASearch,
		},
	}
}

func (s *asyncHalvingSearch) Snapshot() (json.RawMessage, error) {
	return json.Marshal(s.asyncHalvingSearchState)
}

func (s *asyncHalvingSearch) Restore(state json.RawMessage) error {
	return json.Unmarshal(state, &s.asyncHalvingSearchState)
}

// promotions handles bookkeeping of validation metrics and returns a RequestID to promote if
// appropriate.
func (r *rung) promotionsAsync(
	requestID model.RequestID, metric float64, divisor float64,
) []model.RequestID {
	// See if there is a trial to promote. We are increasing the total number of trials seen by 1; the
	// number of best trials that definitely should have been promoted so far (numPromote) can only
	// stay the same or increase by 1.
	oldNumPromote := int(float64(len(r.Metrics)) / divisor)
	numPromote := int(float64(len(r.Metrics)+1) / divisor)

	// Insert the new trial result in the appropriate place in the sorted list.
	insertIndex := sort.Search(
		len(r.Metrics),
		func(i int) bool { return r.Metrics[i].Metric > metric },
	)
	promoteNow := insertIndex < numPromote

	r.Metrics = append(r.Metrics, trialMetric{})
	copy(r.Metrics[insertIndex+1:], r.Metrics[insertIndex:])
	r.Metrics[insertIndex] = trialMetric{
		RequestID: requestID,
		Metric:    metric,
		Promoted:  promoteNow,
	}

	// If the new trial is good enough, it should be promoted immediately (whether or not numPromote
	// changes). Otherwise, if numPromote changes, there is some other trial that should be promoted,
	// unless it has been promoted already.
	switch {
	case promoteNow:
		return []model.RequestID{requestID}
	case numPromote != oldNumPromote && !r.Metrics[oldNumPromote].Promoted:
		t := &r.Metrics[oldNumPromote]
		t.Promoted = true
		return []model.RequestID{t.RequestID}
	default:
		return nil
	}
}

func (s *asyncHalvingSearch) initialOperations(ctx context) ([]Operation, error) {
	// The number of initialOperations will control the degree of parallelism
	// of the search experiment since we guarantee that each validationComplete
	// call will return a new train workload until we reach MaxTrials.

	// We will use searcher config field if available.
	// Otherwise we will default to a number of trials that will
	// guarantee at least one trial at the top rung.
	var ops []Operation
	var maxConcurrentTrials int

	if s.MaxConcurrentTrials > 0 {
		maxConcurrentTrials = min(s.MaxConcurrentTrials, s.MaxTrials)
	} else {
		maxConcurrentTrials = max(
			min(int(math.Pow(s.Divisor, float64(s.NumRungs-1))), s.MaxTrials),
			1)
	}

	for trial := 0; trial < maxConcurrentTrials; trial++ {
		create := NewCreate(
			ctx.rand, sampleAll(ctx.hparams, ctx.rand), model.TrialWorkloadSequencerType)
		s.TrialRungs[create.RequestID] = 0
		ops = append(ops, create)
		ops = append(ops, NewValidateAfter(create.RequestID, s.Rungs[0].UnitsNeeded))
		s.PendingTrials++
	}
	return ops, nil
}

func (s *asyncHalvingSearch) trialCreated(
	ctx context, requestID model.RequestID,
) ([]Operation, error) {
	s.Rungs[0].OutstandingTrials++
	s.TrialRungs[requestID] = 0
	return nil, nil
}

func (s *asyncHalvingSearch) trialClosed(
	ctx context, requestID model.RequestID,
) ([]Operation, error) {
	s.TrialsCompleted++
	s.ClosedTrials[requestID] = true
	return nil, nil
}

func (s *asyncHalvingSearch) validationCompleted(
	ctx context, requestID model.RequestID, metric float64,
) ([]Operation, error) {
	s.PendingTrials--
	if !s.SmallerIsBetter {
		metric *= -1
	}
	return s.promoteAsync(ctx, requestID, metric), nil
}

func (s *asyncHalvingSearch) promoteAsync(
	ctx context, requestID model.RequestID, metric float64,
) []Operation {
	// Upon a validation complete, we should return at least one more train&val workload
	// unless the bracket of successive halving is finished.
	rungIndex := s.TrialRungs[requestID]
	rung := s.Rungs[rungIndex]
	rung.OutstandingTrials--
	addedTrainWorkload := false

	var ops []Operation
	// If the trial has completed the top rung's validation, close the trial.
	if rungIndex == s.NumRungs-1 {
		rung.Metrics = append(rung.Metrics,
			trialMetric{
				RequestID: requestID,
				Metric:    metric,
			},
		)

		if !s.EarlyExitTrials[requestID] {
			ops = append(ops, NewClose(requestID))
			s.ClosedTrials[requestID] = true
		}
	} else {
		// This is not the top rung, so do promotions to the next rung.
		nextRung := s.Rungs[rungIndex+1]
		for _, promotionID := range rung.promotionsAsync(
			requestID,
			metric,
			s.Divisor,
		) {
			s.TrialRungs[promotionID] = rungIndex + 1
			nextRung.OutstandingTrials++
			if !s.EarlyExitTrials[promotionID] {
				unitsNeeded := max(nextRung.UnitsNeeded.Units-rung.UnitsNeeded.Units, 1)
				ops = append(ops, NewValidateAfter(promotionID, model.NewLength(s.Unit(), unitsNeeded)))
				addedTrainWorkload = true
				s.PendingTrials++
			} else {
				// We make a recursive call that will behave the same
				// as if we'd actually run the promoted job and received
				// the worse possible result in return.
				return s.promoteAsync(ctx, promotionID, ashaExitedMetricValue)
			}
		}
	}

	allTrials := len(s.TrialRungs) - s.InvalidTrials
	if !addedTrainWorkload && allTrials < s.MaxTrials {
		s.PendingTrials++
		create := NewCreate(
			ctx.rand, sampleAll(ctx.hparams, ctx.rand), model.TrialWorkloadSequencerType)
		s.TrialRungs[create.RequestID] = 0
		ops = append(ops, create)
		ops = append(ops, NewValidateAfter(create.RequestID, s.Rungs[0].UnitsNeeded))
	}

	// Only close out trials once we have reached the MaxTrials for the searcher.
	if len(s.Rungs[0].Metrics) == s.MaxTrials {
		ops = append(ops, s.closeOutRungs()...)
	}
	return ops
}

// closeOutRungs closes all remaining unpromoted trials in any rungs that have no more outstanding
// trials.
func (s *asyncHalvingSearch) closeOutRungs() []Operation {
	var ops []Operation
	for _, rung := range s.Rungs {
		if rung.OutstandingTrials > 0 {
			break
		}
		for _, trialMetric := range rung.Metrics {
			if !trialMetric.Promoted && !s.ClosedTrials[trialMetric.RequestID] {
				if !s.EarlyExitTrials[trialMetric.RequestID] {
					ops = append(ops, NewClose(trialMetric.RequestID))
					s.ClosedTrials[trialMetric.RequestID] = true
				}
			}
		}
	}
	return ops
}

func (s *asyncHalvingSearch) progress(map[model.RequestID]model.PartialUnits) float64 {
	if s.MaxConcurrentTrials > 0 && s.PendingTrials > s.MaxConcurrentTrials {
		panic("pending trials is greater than max_concurrent_trials")
	}
	allTrials := len(s.Rungs[0].Metrics)
	// Give ourselves an overhead of 20% of MaxTrials when calculating progress.
	progress := float64(allTrials) / (1.2 * float64(s.MaxTrials))
	if allTrials == s.MaxTrials {
		numValidTrials := float64(s.TrialsCompleted) - float64(s.InvalidTrials)
		progressNoOverhead := numValidTrials / float64(s.MaxTrials)
		progress = math.Max(progressNoOverhead, progress)
	}
	return progress
}

func (s *asyncHalvingSearch) trialExitedEarly(
	ctx context, requestID model.RequestID, exitedReason workload.ExitedReason,
) ([]Operation, error) {
	s.PendingTrials--
	if exitedReason == workload.InvalidHP {
		var ops []Operation
		s.EarlyExitTrials[requestID] = true
		ops = append(ops, NewClose(requestID))
		s.ClosedTrials[requestID] = true
		s.InvalidTrials++
		// Remove metrics associated with InvalidHP trial across all rungs
		highestRungIndex := s.TrialRungs[requestID]
		rung := s.Rungs[highestRungIndex]
		rung.OutstandingTrials--
		for rungIndex := 0; rungIndex <= highestRungIndex; rungIndex++ {
			rung := s.Rungs[rungIndex]
			for i, trialMetric := range rung.Metrics {
				if trialMetric.RequestID == requestID {
					rung.Metrics = append(rung.Metrics[:i], rung.Metrics[i+1:]...)
					break
				}
			}
		}
		// Add new trial to searcher queue
		create := NewCreate(
			ctx.rand, sampleAll(ctx.hparams, ctx.rand), model.TrialWorkloadSequencerType)
		s.TrialRungs[create.RequestID] = 0
		ops = append(ops, create)
		ops = append(ops, NewValidateAfter(create.RequestID, s.Rungs[0].UnitsNeeded))
		s.PendingTrials++
		return ops, nil
	}
	s.EarlyExitTrials[requestID] = true
	s.ClosedTrials[requestID] = true
	return s.promoteAsync(ctx, requestID, ashaExitedMetricValue), nil
}
