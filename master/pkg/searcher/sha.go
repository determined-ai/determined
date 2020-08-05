package searcher

import (
	"math"
	"sort"

	"github.com/pkg/errors"

	"github.com/determined-ai/determined/master/pkg/model"
)

// syncHalvingSearch implements a search using the synchronous successive halving algorithm
// (SHA).
type syncHalvingSearch struct {
	defaultSearchMethod
	model.SyncHalvingConfig

	rungs      []*rung
	trialRungs map[RequestID]int
	// earlyExitTrials contains trials that exited early that are still considered in the search.
	earlyExitTrials   map[RequestID]bool
	expectedWorkloads int
	trialsCompleted   int
	batchesPerStep    int
}

const shaExitedMetricValue = math.MaxFloat64

func newSyncHalvingSearch(config model.SyncHalvingConfig, batchesPerStep int) SearchMethod {
	rungs := make([]*rung, 0, config.NumRungs)
	expectedSteps := 0
	for id := 0; id < config.NumRungs; id++ {
		compound := math.Pow(config.Divisor, float64(config.NumRungs-id-1))
		stepsNeeded := max(int(float64(config.TargetTrialSteps)/compound), 1)
		startTrials := max(int(compound), 1)
		rungs = append(rungs,
			&rung{
				stepsNeeded: stepsNeeded,
				startTrials: startTrials,
			},
		)
		if id == 0 {
			expectedSteps += stepsNeeded * startTrials
		} else {
			expectedSteps += (stepsNeeded - rungs[id-1].stepsNeeded) * startTrials
		}
	}

	expectedWorkloads := 0
	multiplier := float64(config.StepBudget) / float64(expectedSteps)
	for id := 0; id < config.NumRungs; id++ {
		cur := rungs[id]
		cur.startTrials = int(multiplier * float64(cur.startTrials))
		if id != 0 {
			prev := rungs[id-1]
			cur.stepsNeeded = max(cur.stepsNeeded, prev.stepsNeeded+1)
			cur.startTrials = max(min(cur.startTrials, prev.startTrials), 1)
			prev.promoteTrials = cur.startTrials
			expectedWorkloads += (cur.stepsNeeded - prev.stepsNeeded + 1) * cur.startTrials
		} else {
			expectedWorkloads += (cur.stepsNeeded + 1) * cur.startTrials
		}
	}

	return &syncHalvingSearch{
		SyncHalvingConfig: config,
		rungs:             rungs,
		trialRungs:        make(map[RequestID]int),
		earlyExitTrials:   make(map[RequestID]bool),
		expectedWorkloads: expectedWorkloads,
		batchesPerStep:    batchesPerStep,
	}
}

type trialMetric struct {
	requestID RequestID
	metric    float64
	// fields below used by asha.go.
	promoted bool
}

// rung describes a set of trials that are to be trained for the same number of steps.
type rung struct {
	stepsNeeded   int
	metrics       []trialMetric
	startTrials   int
	promoteTrials int
	// field below used by asha.go.
	outstandingTrials int
}

// promotions handles bookkeeping of validation metrics and returns a RequestID to promote if
// appropriate.
func (r *rung) promotionsSync(requestID RequestID, metric float64) []RequestID {
	// Insert the new trial result in the appropriate place in the sorted list.
	insertIndex := sort.Search(
		len(r.metrics),
		func(i int) bool { return r.metrics[i].metric > metric },
	)
	r.metrics = append(r.metrics, trialMetric{})
	copy(r.metrics[insertIndex+1:], r.metrics[insertIndex:])
	r.metrics[insertIndex] = trialMetric{
		requestID: requestID,
		metric:    metric,
	}

	// If there are enough trials done to definitively promote one, do so. Otherwise, return nil.
	currPromote := len(r.metrics) + r.promoteTrials - r.startTrials
	switch {
	case currPromote <= 0: // Not enough trials completed for any promotions.
		return nil
	case insertIndex < currPromote: // Incoming trial should be promoted.
		return []RequestID{requestID}
	default: // Promote next trial in sorted metrics array.
		t := &r.metrics[currPromote-1]
		return []RequestID{t.requestID}
	}
}

func (s *syncHalvingSearch) initialOperations(ctx context) ([]Operation, error) {
	var ops []Operation
	for trial := 0; trial < s.rungs[0].startTrials; trial++ {
		create := NewCreate(
			ctx.rand, sampleAll(ctx.hparams, ctx.rand), model.TrialWorkloadSequencerType)
		ops = append(ops, create)
		trainVal := trainAndValidate(
			create.RequestID, 0, s.rungs[0].stepsNeeded, s.batchesPerStep)
		ops = append(ops, trainVal...)
	}
	return ops, nil
}

func (s *syncHalvingSearch) trainCompleted(
	ctx context, requestID RequestID, message Workload,
) ([]Operation, error) {
	return nil, nil
}

func (s *syncHalvingSearch) validationCompleted(
	ctx context, requestID RequestID, message Workload, metrics ValidationMetrics,
) ([]Operation, error) {
	// Extract the relevant metric as a float.
	metric, err := metrics.Metric(s.Metric)
	if err != nil {
		return nil, err
	}
	if !s.SmallerIsBetter {
		metric *= -1
	}

	return s.promoteSync(ctx, requestID, message, metric)
}

func (s *syncHalvingSearch) promoteSync(
	ctx context, requestID RequestID, message Workload, metric float64,
) ([]Operation, error) {
	rungIndex := s.trialRungs[requestID]
	rung := s.rungs[rungIndex]

	// If the trial has completed the top rung's validation, close the trial and do nothing else.
	if rungIndex == s.NumRungs-1 {
		s.trialsCompleted++
		if !s.earlyExitTrials[requestID] {
			return []Operation{NewClose(requestID)}, nil
		}
		return nil, nil
	}

	var ops []Operation
	// Since this is not the top rung, handle promotions if there are any, then close the rung if
	// all trials have finished.
	if toPromote := rung.promotionsSync(requestID, metric); len(toPromote) > 0 {
		for _, promotionID := range toPromote {
			s.trialRungs[promotionID] = rungIndex + 1
			if !s.earlyExitTrials[promotionID] {
				ops = append(ops, trainAndValidate(promotionID, rung.stepsNeeded,
					s.rungs[rungIndex+1].stepsNeeded, s.batchesPerStep)...)
			} else {
				// Since the trial being promoted has already exited and will never finish any more workloads,
				// we should treat it as immediately completing the next rung with the worst possible result.
				step := s.rungs[rungIndex+1].stepsNeeded
				wkld := Workload{
					Kind:         ComputeValidationMetrics,
					ExperimentID: message.ExperimentID,
					TrialID:      message.TrialID,
					StepID:       step,
				}
				// The recursive call is safe because the rung being considered goes up by one each time and
				// there are a finite number of rungs.
				return s.promoteSync(ctx, promotionID, wkld, shaExitedMetricValue)
			}
		}
		// Close the unpromoted trials in the rung once all trials in the rung finish.
		if rung.startTrials < len(rung.metrics) {
			return nil, errors.Errorf("number of trials exceeded initial trials for rung: %d < %d",
				rung.startTrials, len(rung.metrics))
		}
		if len(rung.metrics) == rung.startTrials {
			for _, trialMetric := range rung.metrics[rung.promoteTrials:] {
				s.trialsCompleted++
				if !s.earlyExitTrials[trialMetric.requestID] {
					ops = append(ops, NewClose(trialMetric.requestID))
				}
			}
		}
	}
	return ops, nil
}

func (s *syncHalvingSearch) progress(workloadsCompleted int) float64 {
	return math.Min(1, float64(workloadsCompleted)/float64(s.expectedWorkloads))
}

func (s *syncHalvingSearch) trialExitedEarly(
	ctx context, requestID RequestID, message Workload,
) ([]Operation, error) {
	s.earlyExitTrials[requestID] = true
	return s.promoteSync(ctx, requestID, message, shaExitedMetricValue)
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
