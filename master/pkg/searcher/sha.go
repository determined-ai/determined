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
	earlyExitTrials map[RequestID]bool
	trialsCompleted int

	expectedUnits model.Length
}

const shaExitedMetricValue = math.MaxFloat64

func newSyncHalvingSearch(config model.SyncHalvingConfig) SearchMethod {
	rungs := make([]*rung, 0, config.NumRungs)
	expectedUnits := 0
	for id := 0; id < config.NumRungs; id++ {
		compound := math.Pow(config.Divisor, float64(config.NumRungs-id-1))
		unitsNeeded := model.NewLength(
			config.MaxLength.Unit,
			max(int(float64(config.MaxLength.Units)/compound), 1),
		)
		startTrials := max(int(compound), 1)
		rungs = append(rungs,
			&rung{
				unitsNeeded: unitsNeeded,
				startTrials: startTrials,
			},
		)
		if id == 0 {
			expectedUnits += unitsNeeded.Units * startTrials
		} else {
			expectedUnits += (unitsNeeded.Units - rungs[id-1].unitsNeeded.Units) * startTrials
		}
	}

	multiplier := float64(config.Budget.Units) / float64(expectedUnits)
	expectedUnits = 0
	for id := 0; id < config.NumRungs; id++ {
		cur := rungs[id]
		cur.startTrials = int(multiplier * float64(cur.startTrials))
		if id == 0 {
			expectedUnits += cur.unitsNeeded.Units * cur.startTrials
		} else {
			prev := rungs[id-1]
			cur.unitsNeeded = model.NewLength(
				cur.unitsNeeded.Unit,
				max(cur.unitsNeeded.Units, prev.unitsNeeded.Units),
			)
			cur.startTrials = max(min(cur.startTrials, prev.startTrials), 1)
			prev.promoteTrials = cur.startTrials
			expectedUnits += (cur.unitsNeeded.Units - prev.unitsNeeded.Units) * cur.startTrials
		}
	}

	return &syncHalvingSearch{
		SyncHalvingConfig: config,
		rungs:             rungs,
		trialRungs:        make(map[RequestID]int),
		earlyExitTrials:   make(map[RequestID]bool),
		expectedUnits:     model.NewLength(config.MaxLength.Unit, expectedUnits),
	}
}

type trialMetric struct {
	requestID RequestID
	metric    float64
	// fields below used by asha.go.
	promoted bool
}

// rung describes a set of trials that are to be trained for the same number of units.
type rung struct {
	unitsNeeded   model.Length
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
		ops = append(ops, NewTrain(create.RequestID, s.rungs[0].unitsNeeded))
		ops = append(ops, NewValidate(create.RequestID))
	}
	return ops, nil
}

func (s *syncHalvingSearch) validationCompleted(
	ctx context, requestID RequestID, validate Validate, metrics ValidationMetrics,
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
	ctx context, requestID RequestID, metric float64,
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
				ops = append(ops, NewTrain(
					promotionID, s.rungs[rungIndex+1].unitsNeeded.Sub(rung.unitsNeeded)))
				ops = append(ops, NewValidate(promotionID))
			} else {
				// We can make a recursive call (and discard the results)
				// because of the following invariants:
				//   1) There are other trials executing that will receive any
				//   extra operations when they complete workloads. We know
				//   this is true since otherwise we would have received
				//   TrialClosed responses already and the searcher would have
				//   closed.
				//
				//   2) We are bounded on the depth of this recursive stack by
				//   the number of rungs. We default this to max out at 5.
				_, err := s.promoteSync(ctx, promotionID, shaExitedMetricValue)
				return nil, err
			}
		}
		// Closes the unpromoted trials in the rung once all trials in the rung finish.
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

func (s *syncHalvingSearch) progress(unitsCompleted model.Length) float64 {
	return math.Min(1, float64(unitsCompleted.Units)/float64(s.expectedUnits.Units))
}

func (s *syncHalvingSearch) trialExitedEarly(
	ctx context, requestID RequestID,
) ([]Operation, error) {
	s.earlyExitTrials[requestID] = true
	return s.promoteSync(ctx, requestID, shaExitedMetricValue)
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
