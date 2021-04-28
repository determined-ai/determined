package searcher

import (
	"encoding/json"
	"math"
	"sort"

	"github.com/determined-ai/determined/master/pkg/workload"

	"github.com/determined-ai/determined/master/pkg/model"
)

// AsyncHalvingStoppingSearch implements a modified version of the asynchronous successive
// halving algorithm (ASHA) that does not require fault tolerance to perform early-stopping.
// For each trial, after a train and validation workload, the algorithm will decide whether
// to stop or continue training the trial based on the ranking of the validation metric
// compared to other trials in a particular rung.  Once a trial has been stopped, it will not
// be resumed later; this is why the algorithm does not require fault tolerance.
// The searcher state and config match that of AsyncHalvingSearch but we will only run
// the stopping based version if StopOnce is true.
type asyncHalvingStoppingSearch struct {
	model.AsyncHalvingConfig
	asyncHalvingSearchState
}

func newAsyncHalvingStoppingSearch(config model.AsyncHalvingConfig) SearchMethod {
	rungs := make([]*rung, 0, config.NumRungs)
	unitsNeeded := 0
	for id := 0; id < config.NumRungs; id++ {
		// We divide the MaxLength by downsampling rate to get the target units
		// for a rung.
		downsamplingRate := math.Pow(config.Divisor, float64(config.NumRungs-id-1))
		unitsNeeded += max(int(float64(config.MaxLength.Units)/downsamplingRate), 1)
		rungs = append(rungs,
			&rung{
				UnitsNeeded:       model.NewLength(config.Unit(), unitsNeeded),
				OutstandingTrials: 0,
			})
	}

	return &asyncHalvingStoppingSearch{
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

func (s *asyncHalvingStoppingSearch) Snapshot() (json.RawMessage, error) {
	return json.Marshal(s.asyncHalvingSearchState)
}

func (s *asyncHalvingStoppingSearch) Restore(state json.RawMessage) error {
	return json.Unmarshal(state, &s.asyncHalvingSearchState)
}

// promotions handles bookkeeping of validation metrics and decides whether to continue
// training the current trial.
func (r *rung) continueTraining(requestID model.RequestID, metric float64, divisor float64) bool {
	// Compute cutoff for promotion to next rung to continue training.
	numPromote := max(int(float64(len(r.Metrics)+1)/divisor), 1)

	// Insert the new trial result in the appropriate place in the sorted list.
	insertIndex := sort.Search(
		len(r.Metrics),
		func(i int) bool { return r.Metrics[i].Metric >= metric },
	)
	// We will continue training if trial ranked in top 1/divisor for the rung or
	// if there are fewere than divisor trials in the rung.
	promoteNow := insertIndex < numPromote

	r.Metrics = append(r.Metrics, trialMetric{})
	copy(r.Metrics[insertIndex+1:], r.Metrics[insertIndex:])
	r.Metrics[insertIndex] = trialMetric{
		RequestID: requestID,
		Metric:    metric,
		Promoted:  promoteNow,
	}

	return promoteNow
}

func (s *asyncHalvingStoppingSearch) initialOperations(ctx context) ([]Operation, error) {
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
	}
	return ops, nil
}

func (s *asyncHalvingStoppingSearch) trialCreated(
	ctx context, requestID model.RequestID) ([]Operation, error) {
	s.Rungs[0].OutstandingTrials++
	s.TrialRungs[requestID] = 0
	return nil, nil
}

func (s *asyncHalvingStoppingSearch) trialClosed(
	ctx context, requestID model.RequestID) ([]Operation, error) {
	s.TrialsCompleted++
	s.ClosedTrials[requestID] = true
	return nil, nil
}

func (s *asyncHalvingStoppingSearch) validationCompleted(
	ctx context, requestID model.RequestID, metric float64,
) ([]Operation, error) {
	if !s.SmallerIsBetter {
		metric *= -1
	}
	return s.promoteAsync(ctx, requestID, metric), nil
}

func (s *asyncHalvingStoppingSearch) promoteAsync(
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
		// We need to run continueTraining even if the trial was terminated early so that we
		// can add the metric to the rung.
		promoteTrial := rung.continueTraining(
			requestID,
			metric,
			s.Divisor,
		)
		// In contrast to promotion-based ASHA, we will not let early-exited trials add
		// -/+inf metrics to higher rungs even if portion of terminated trials in bottom rung
		// is greater than 1 - 1 / divisor.
		if !s.EarlyExitTrials[requestID] {
			if promoteTrial {
				s.TrialRungs[requestID] = rungIndex + 1
				nextRung.OutstandingTrials++
				unitsNeeded := max(nextRung.UnitsNeeded.Units-rung.UnitsNeeded.Units, 1)
				ops = append(ops, NewValidateAfter(requestID, model.NewLength(s.Unit(), unitsNeeded)))
				addedTrainWorkload = true
			} else {
				ops = append(ops, NewClose(requestID))
				s.ClosedTrials[requestID] = true
			}
		}
	}

	allTrials := len(s.TrialRungs) - s.InvalidTrials
	if !addedTrainWorkload && allTrials < s.MaxTrials {
		create := NewCreate(
			ctx.rand, sampleAll(ctx.hparams, ctx.rand), model.TrialWorkloadSequencerType)
		s.TrialRungs[create.RequestID] = 0
		ops = append(ops, create)
		ops = append(ops, NewValidateAfter(create.RequestID, s.Rungs[0].UnitsNeeded))
	}

	return ops
}

func (s *asyncHalvingStoppingSearch) progress(map[model.RequestID]model.PartialUnits) float64 {
	allTrials := len(s.Rungs[0].Metrics)
	// Give ourselves an overhead of 20% of maxTrials when calculating progress.
	progress := float64(allTrials) / (1.2 * float64(s.MaxTrials))
	if allTrials == s.MaxTrials {
		numValidTrials := float64(s.TrialsCompleted) - float64(s.InvalidTrials)
		progressNoOverhead := numValidTrials / float64(s.MaxTrials)
		progress = math.Max(progressNoOverhead, progress)
	}
	return progress
}

func (s *asyncHalvingStoppingSearch) trialExitedEarly(
	ctx context, requestID model.RequestID, exitedReason workload.ExitedReason,
) ([]Operation, error) {
	if exitedReason == workload.InvalidHP {
		var ops []Operation
		s.EarlyExitTrials[requestID] = true
		ops = append(ops, NewClose(requestID))
		s.ClosedTrials[requestID] = true
		s.InvalidTrials++
		// Remove metrics associated with InvalidHP trial across all rungs
		highestRungIndex := s.TrialRungs[requestID]
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
		return ops, nil
	}
	s.EarlyExitTrials[requestID] = true
	s.ClosedTrials[requestID] = true
	return s.promoteAsync(ctx, requestID, ashaExitedMetricValue), nil
}
