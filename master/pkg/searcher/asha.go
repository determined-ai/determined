package searcher

import (
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
	earlyExitTrials     map[RequestID]bool
	totalStepsCompleted int
	maxTrials           int
	trialsCompleted     int
}

const ashaExitedMetricValue = math.MaxFloat64

func newAsyncHalvingSearch(config model.AsyncHalvingConfig) SearchMethod {
	rungs := make([]*rung, 0, config.NumRungs)
	for id := 0; id < config.NumRungs; id++ {
		compound := math.Pow(config.Divisor, float64(config.NumRungs-id-1))
		stepsNeeded := max(int(float64(config.TargetTrialSteps)/compound), 1)
		rungs = append(rungs,
			&rung{
				divisor:           config.Divisor,
				stepsNeeded:       stepsNeeded,
				outstandingTrials: 0})
	}

	return &asyncHalvingSearch{
		AsyncHalvingConfig: config,
		rungs:              rungs,
		trialRungs:         make(map[RequestID]int),
		earlyExitTrials:    make(map[RequestID]bool),
		maxTrials:          config.MaxTrials,
	}
}

type trialMetric struct {
	requestID RequestID
	metric    float64
	promoted  bool
	closed    bool
}

// rung describes a set of trials that are to be trained for the same number of steps.
type rung struct {
	divisor           float64
	stepsNeeded       int
	metrics           []trialMetric
	outstandingTrials int
	promoteTrials     int
	startTrials       int
}

// promotions handles bookkeeping of validation metrics and returns a RequestID to promote if
// appropriate.
func (r *rung) promotionsAsync(requestID RequestID, metric float64) []RequestID {
	// See if there is a trial to promote. We are increasing the total number of trials seen by 1; the
	// number of best trials that definitely should have been promoted so far (numPromote) can only
	// stay the same or increase by 1.
	oldNumPromote := int(float64(len(r.metrics)) / r.divisor)
	numPromote := int(float64(len(r.metrics)+1) / r.divisor)

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
		closed:    false,
	}

	// If the new trial is good enough, it should be promoted immediately (whether or not numPromote
	// changes). Otherwise, if numPromote changes, there is some other trial that should be promoted,
	// unless it has been promoted already.
	if promoteNow {
		return []RequestID{requestID}
	}
	if numPromote != oldNumPromote {
		t := &r.metrics[oldNumPromote]
		if !t.promoted {
			t.promoted = true
			return []RequestID{t.requestID}
		}
	}

	return nil
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

	if s.MaxConcurrentTrials != nil {
		maxConcurrentTrials = min(*s.MaxConcurrentTrials, s.MaxTrials)
	} else {
		maxConcurrentTrials = max(
			min(int(math.Pow(s.Divisor, float64(s.NumRungs-1))), s.MaxTrials),
			1)
	}

	for trial := 0; trial < maxConcurrentTrials; trial++ {
		create := NewCreate(
			ctx.rand, sampleAll(ctx.hparams, ctx.rand), model.TrialWorkloadSequencerType)
		ops = append(ops, create)
		ops = append(ops, trainAndValidate(create.RequestID, 0, s.rungs[0].stepsNeeded)...)
		s.rungs[0].outstandingTrials++
		s.trialRungs[create.RequestID] = 0
	}
	return ops, nil
}

func (s *asyncHalvingSearch) trainCompleted(
	ctx context, requestID RequestID, message Workload,
) ([]Operation, error) {
	s.totalStepsCompleted++
	return nil, nil
}

func (s *asyncHalvingSearch) validationCompleted(
	ctx context, requestID RequestID, message Workload, metrics ValidationMetrics,
) ([]Operation, error) {
	// Extract the relevant metric as a float.
	metric, err := metrics.Metric(s.AsyncHalvingConfig.Metric)
	if err != nil {
		return nil, err
	}
	if !s.AsyncHalvingConfig.SmallerIsBetter {
		metric *= -1
	}

	return s.promoteAsync(ctx, requestID, message, metric), nil
}

func (s *asyncHalvingSearch) promoteAsync(
	ctx context, requestID RequestID, message Workload, metric float64,
) []Operation {
	// Upon a trial is finished, we should return at least one more trial to train unless
	// the bracket of successive halving is finished.
	rungIndex := s.trialRungs[requestID]
	rung := s.rungs[rungIndex]
	rung.outstandingTrials--
	addedNewTrial := false

	ops := make([]Operation, 0)
	// If the trial has completed the top rung's validation, close the trial.
	if rungIndex == s.AsyncHalvingConfig.NumRungs-1 {
		s.trialsCompleted++
		if !s.earlyExitTrials[requestID] {
			ops = append(ops, NewClose(requestID))
		}
	} else {
		// This is not the top rung, so do promotions to the next rung.
		nextRung := s.rungs[rungIndex+1]
		for _, promotionID := range rung.promotionsAsync(requestID, metric) {
			s.trialRungs[promotionID] = rungIndex + 1
			nextRung.outstandingTrials++
			if !s.earlyExitTrials[promotionID] {
				ops = append(
					ops, trainAndValidate(promotionID, rung.stepsNeeded, nextRung.stepsNeeded)...)
				addedNewTrial = true
			} else {
				step := s.rungs[rungIndex+1].stepsNeeded
				wkld := Workload{
					Kind:         ComputeValidationMetrics,
					ExperimentID: message.ExperimentID,
					TrialID:      message.TrialID,
					StepID:       step,
				}

				// We make a recursive call that will behave the same
				// as if we'd actually run the promoted job and received
				// the worse possible result in return.
				return s.promoteAsync(ctx, promotionID, wkld, ashaExitedMetricValue)
			}
		}
	}

	allTrials := len(s.trialRungs)
	if !addedNewTrial && allTrials < s.maxTrials {
		create := NewCreate(
			ctx.rand, sampleAll(ctx.hparams, ctx.rand), model.TrialWorkloadSequencerType)
		ops = append(ops, create)
		ops = append(ops, trainAndValidate(create.RequestID, 0, s.rungs[0].stepsNeeded)...)
		s.rungs[0].outstandingTrials++
		s.trialRungs[create.RequestID] = 0
	}

	// Only close out trials once we have reached the maxTrials for the searcher.
	if allTrials == s.maxTrials {
		ops = append(ops, s.closeOutRungs()...)
	}
	return ops
}

// closeOutRungs closes all remaining unpromoted trials in any rungs that have no more outstanding
// trials.
func (s *asyncHalvingSearch) closeOutRungs() []Operation {
	ops := make([]Operation, 0)
	for _, rung := range s.rungs {
		if rung.outstandingTrials > 0 {
			break
		}
		for tid, trialMetric := range rung.metrics {
			if !trialMetric.promoted && !trialMetric.closed {
				s.trialsCompleted++
				if !s.earlyExitTrials[trialMetric.requestID] {
					ops = append(ops, NewClose(trialMetric.requestID))
				}
				rung.metrics[tid].closed = true
				ops = append(ops, NewClose(trialMetric.requestID))
			}
		}
	}
	return ops
}

func (s *asyncHalvingSearch) progress(workloadsCompleted int) float64 {
	allTrials := len(s.trialRungs)
	// Give ourselves an overhead of 20% of maxTrials when calculating progress.
	progress := float64(allTrials-s.rungs[0].outstandingTrials) / (1.2 * float64(s.maxTrials))
	if allTrials == s.maxTrials {
		return math.Max(float64(s.trialsCompleted)/float64(s.maxTrials), progress)
	}
	return progress
}

func (s *asyncHalvingSearch) trialExitedEarly(
	ctx context, requestID RequestID, message Workload,
) ([]Operation, error) {
	s.earlyExitTrials[requestID] = true
	return s.promoteAsync(ctx, requestID, message, ashaExitedMetricValue), nil
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
