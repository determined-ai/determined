package searcher

import (
	"github.com/determined-ai/determined/master/pkg/workload"
	"math"
	"sort"

	"github.com/determined-ai/determined/master/pkg/model"
)

// AsyncHalvingSearch implements a search using the asynchronous successive halving algorithm
// (ASHA). The experiment will run until the target number of trials have been completed
// in the bottom rung and no further promotions can be made to higher rungs.
type asyncHalvingSearch struct {
	defaultSearchMethod
	model.AsyncHalvingConfig

	rungs      []*rung
	trialRungs map[RequestID]int
	// earlyExitTrials contains trials that exited early that are still considered in the search.
	earlyExitTrials map[RequestID]bool
	closedTrials    map[RequestID]bool
	maxTrials       int
	trialsCompleted int
}

const ashaExitedMetricValue = math.MaxFloat64

func newAsyncHalvingSearch(config model.AsyncHalvingConfig) SearchMethod {
	rungs := make([]*rung, 0, config.NumRungs)
	for id := 0; id < config.NumRungs; id++ {
		// We divide the MaxLength by downsampling rate to get the target units
		// for a rung.
		downsamplingRate := math.Pow(config.Divisor, float64(config.NumRungs-id-1))
		unitsNeeded := max(int(float64(config.MaxLength.Units)/downsamplingRate), 1)
		rungs = append(rungs,
			&rung{
				unitsNeeded:       model.NewLength(config.Unit(), unitsNeeded),
				outstandingTrials: 0,
			})
	}

	return &asyncHalvingSearch{
		AsyncHalvingConfig: config,
		rungs:              rungs,
		trialRungs:         make(map[RequestID]int),
		earlyExitTrials:    make(map[RequestID]bool),
		closedTrials:       make(map[RequestID]bool),
		maxTrials:          config.MaxTrials,
	}
}

// promotions handles bookkeeping of validation metrics and returns a RequestID to promote if
// appropriate.
func (r *rung) promotionsAsync(requestID RequestID, metric float64, divisor float64) []RequestID {
	// See if there is a trial to promote. We are increasing the total number of trials seen by 1; the
	// number of best trials that definitely should have been promoted so far (numPromote) can only
	// stay the same or increase by 1.
	oldNumPromote := int(float64(len(r.metrics)) / divisor)
	numPromote := int(float64(len(r.metrics)+1) / divisor)

	// Insert the new trial result in the appropriate place in the sorted list.
	insertIndex := sort.Search(
		len(r.metrics),
		func(i int) bool { return r.metrics[i].metric > metric },
	)
	promoteNow := insertIndex < numPromote

	r.metrics = append(r.metrics, trialMetric{})
	copy(r.metrics[insertIndex+1:], r.metrics[insertIndex:])
	r.metrics[insertIndex] = trialMetric{
		requestID: requestID,
		metric:    metric,
		promoted:  promoteNow,
	}

	// If the new trial is good enough, it should be promoted immediately (whether or not numPromote
	// changes). Otherwise, if numPromote changes, there is some other trial that should be promoted,
	// unless it has been promoted already.
	switch {
	case promoteNow:
		return []RequestID{requestID}
	case numPromote != oldNumPromote && !r.metrics[oldNumPromote].promoted:
		t := &r.metrics[oldNumPromote]
		t.promoted = true
		return []RequestID{t.requestID}
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
		s.trialRungs[create.RequestID] = 0
		ops = append(ops, create)
		ops = append(ops, NewTrain(create.RequestID, s.rungs[0].unitsNeeded))
		ops = append(ops, NewValidate(create.RequestID))
	}
	return ops, nil
}

func (s *asyncHalvingSearch) trialCreated(ctx context, requestID RequestID) ([]Operation, error) {
	s.rungs[0].outstandingTrials++
	s.trialRungs[requestID] = 0
	return nil, nil
}

func (s *asyncHalvingSearch) trialClosed(ctx context, requestID RequestID) ([]Operation, error) {
	s.trialsCompleted++
	s.closedTrials[requestID] = true
	return nil, nil
}

func (s *asyncHalvingSearch) validationCompleted(
	ctx context, requestID RequestID, validate Validate, metrics workload.ValidationMetrics,
) ([]Operation, error) {
	// Extract the relevant metric as a float.
	metric, err := metrics.Metric(s.Metric)
	if err != nil {
		return nil, err
	}
	if !s.SmallerIsBetter {
		metric *= -1
	}

	return s.promoteAsync(ctx, requestID, metric), nil
}

func (s *asyncHalvingSearch) promoteAsync(
	ctx context, requestID RequestID, metric float64,
) []Operation {
	// Upon a validation complete, we should return at least one more train&val workload
	// unless the bracket of successive halving is finished.
	rungIndex := s.trialRungs[requestID]
	rung := s.rungs[rungIndex]
	rung.outstandingTrials--
	addedTrainWorkload := false

	var ops []Operation
	// If the trial has completed the top rung's validation, close the trial.
	if rungIndex == s.NumRungs-1 {
		rung.metrics = append(rung.metrics,
			trialMetric{
				requestID: requestID,
				metric:    metric,
			},
		)

		if !s.earlyExitTrials[requestID] {
			ops = append(ops, NewClose(requestID))
			s.closedTrials[requestID] = true
		}
	} else {
		// This is not the top rung, so do promotions to the next rung.
		nextRung := s.rungs[rungIndex+1]
		for _, promotionID := range rung.promotionsAsync(
			requestID,
			metric,
			s.Divisor,
		) {
			s.trialRungs[promotionID] = rungIndex + 1
			nextRung.outstandingTrials++
			if !s.earlyExitTrials[promotionID] {
				unitsNeeded := max(nextRung.unitsNeeded.Units-rung.unitsNeeded.Units, 1)
				ops = append(ops, NewTrain(promotionID, model.NewLength(s.Unit(), unitsNeeded)))
				ops = append(ops, NewValidate(promotionID))
				addedTrainWorkload = true
			} else {
				// We make a recursive call that will behave the same
				// as if we'd actually run the promoted job and received
				// the worse possible result in return.
				return s.promoteAsync(ctx, promotionID, ashaExitedMetricValue)
			}
		}
	}

	allTrials := len(s.trialRungs)
	if !addedTrainWorkload && allTrials < s.maxTrials {
		create := NewCreate(
			ctx.rand, sampleAll(ctx.hparams, ctx.rand), model.TrialWorkloadSequencerType)
		s.trialRungs[create.RequestID] = 0
		ops = append(ops, create)
		ops = append(ops, NewTrain(create.RequestID, s.rungs[0].unitsNeeded))
		ops = append(ops, NewValidate(create.RequestID))
	}

	// Only close out trials once we have reached the maxTrials for the searcher.
	if len(s.rungs[0].metrics) == s.maxTrials {
		ops = append(ops, s.closeOutRungs()...)
	}
	return ops
}

// closeOutRungs closes all remaining unpromoted trials in any rungs that have no more outstanding
// trials.
func (s *asyncHalvingSearch) closeOutRungs() []Operation {
	var ops []Operation
	for _, rung := range s.rungs {
		if rung.outstandingTrials > 0 {
			break
		}
		for _, trialMetric := range rung.metrics {
			if !trialMetric.promoted && !s.closedTrials[trialMetric.requestID] {
				if !s.earlyExitTrials[trialMetric.requestID] {
					ops = append(ops, NewClose(trialMetric.requestID))
					s.closedTrials[trialMetric.requestID] = true
				}
			}
		}
	}
	return ops
}

func (s *asyncHalvingSearch) progress(float64) float64 {
	allTrials := len(s.rungs[0].metrics)
	// Give ourselves an overhead of 20% of maxTrials when calculating progress.
	progress := float64(allTrials) / (1.2 * float64(s.maxTrials))
	if allTrials == s.maxTrials {
		progress = math.Max(float64(s.trialsCompleted)/float64(s.maxTrials), progress)
	}
	return progress
}

func (s *asyncHalvingSearch) trialExitedEarly(
	ctx context, requestID RequestID,
) ([]Operation, error) {
	s.earlyExitTrials[requestID] = true
	s.closedTrials[requestID] = true
	return s.promoteAsync(ctx, requestID, ashaExitedMetricValue), nil
}
