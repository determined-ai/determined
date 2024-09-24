package searcher

import (
	"encoding/json"
	"fmt"
	"github.com/determined-ai/determined/master/pkg/ptrs"
	"math"
	"sort"

	"github.com/determined-ai/determined/master/pkg/mathx"
	"github.com/determined-ai/determined/master/pkg/model"
	"github.com/determined-ai/determined/master/pkg/schemas/expconf"
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
	expconf.AsyncHalvingConfig
	SmallerIsBetter bool
	Metric          string
	asyncHalvingSearchState
}
type (
	asyncHalvingSearchState struct {
		Rungs    []*rung       `json:"rungs"`
		RunRungs map[int32]int `json:"run_rungs"`
		// EarlyExitRuns contains trials that exited early that are still considered in the search.
		EarlyExitRuns    map[int32]bool   `json:"early_exit_runs"`
		RunsCompleted    int              `json:"runs_completed"`
		InvalidRuns      int              `json:"invalid_runs"`
		SearchMethodType SearchMethodType `json:"search_method_type"`
	}

	trialMetric struct {
		RunID  int32                 `json:"run_id"`
		Metric model.ExtendedFloat64 `json:"metric"`
	}
	rung struct {
		UnitsNeeded uint64        `json:"units_needed"`
		Metrics     []trialMetric `json:"metrics"`
	}
)

func (r *rung) String() string {
	return fmt.Sprintf("Rung(%d, %v)", r.UnitsNeeded, r.Metrics)
}

type (
	legacyAsyncHalvingSearchState struct {
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
	legacyTrialMetric struct {
		RequestID model.RequestID       `json:"request_id"`
		Metric    model.ExtendedFloat64 `json:"metric"`
		// fields below used by asha.go.
		Promoted bool `json:"promoted"`
	}

	// legacyRung describes a set of trials that are to be trained for the same number of units.
	legacyRung struct {
		UnitsNeeded   uint64        `json:"units_needed"`
		Metrics       []trialMetric `json:"metrics"`
		StartTrials   int           `json:"start_trials"`
		PromoteTrials int           `json:"promote_trials"`
		// field below used by asha.go.
		OutstandingTrials int `json:"outstanding_trials"`
	}
)

const ashaExitedMetricValue = math.MaxFloat64

func makeRungs(numRungs int, divisor float64, maxLength uint64) []*rung {
	rungs := make([]*rung, 0, numRungs)
	for i := 0; i < numRungs; i++ {
		// We divide the MaxLength by downsampling rate to get the target units
		// for a bracketRung.
		downsamplingRate := math.Pow(divisor, float64(numRungs-i-1))
		unitsNeeded := mathx.Max(uint64(float64(maxLength)/downsamplingRate), 1)
		rungs = append(rungs,
			&rung{
				UnitsNeeded: unitsNeeded,
			})
	}
	return rungs
}

func newAsyncHalvingStoppingSearch(
	config expconf.AsyncHalvingConfig, smallerIsBetter bool, metric string,
) SearchMethod {
	rungs := makeRungs(config.NumRungs(), config.Divisor(), config.Length().Units)

	return &asyncHalvingStoppingSearch{
		AsyncHalvingConfig: config,
		SmallerIsBetter:    smallerIsBetter,
		Metric:             metric,
		asyncHalvingSearchState: asyncHalvingSearchState{
			Rungs:            rungs,
			RunRungs:         make(map[int32]int),
			EarlyExitRuns:    make(map[int32]bool),
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

// Insert a completed validation metric in the rung.
// Return the insert index.
func (r *rung) insertMetric(runID int32, metric float64) int {
	// Insert the new trial result in the appropriate place in the sorted list.
	insertIndex := sort.Search(
		len(r.Metrics),
		func(i int) bool { return float64(r.Metrics[i].Metric) >= metric },
	)

	// Add metrics to state.
	r.Metrics = append(r.Metrics, trialMetric{})
	copy(r.Metrics[insertIndex+1:], r.Metrics[insertIndex:])
	r.Metrics[insertIndex] = trialMetric{
		RunID:  runID,
		Metric: model.ExtendedFloat64(metric),
	}
	return insertIndex
}

func (s *asyncHalvingStoppingSearch) initialRuns(ctx context) ([]Action, error) {
	// The number of initialOperations will control the degree of parallelism
	// of the search experiment since we guarantee that each validationComplete
	// call will return a new train workload until we reach MaxTrials.
	// xxx: comment
	// We will use searcher config field if available.
	// Otherwise we will default to a number of trials that will
	// guarantee at least one trial at the top rung.
	var actions []Action
	var maxConcurrentTrials int

	if s.MaxConcurrentTrials() > 0 {
		maxConcurrentTrials = mathx.Min(s.MaxConcurrentTrials(), s.MaxTrials())
	} else {
		maxConcurrentTrials = mathx.Clamp(
			1,
			int(math.Pow(s.Divisor(), float64(s.NumRungs()-1))),
			s.MaxTrials(),
		)
	}

	for trial := 0; trial < maxConcurrentTrials; trial++ {
		create := NewCreate(
			ctx.rand, sampleAll(ctx.hparams, ctx.rand))
		actions = append(actions, create)
	}
	return actions, nil
}

func (s *asyncHalvingStoppingSearch) runCreated(
	ctx context, runID int32, action Create,
) ([]Action, error) {
	s.RunRungs[runID] = 0
	return nil, nil
}

func (s *asyncHalvingStoppingSearch) runClosed(
	ctx context, runID int32,
) ([]Action, error) {
	s.RunsCompleted++
	return nil, nil
}

func (s *asyncHalvingStoppingSearch) validationCompleted(
	ctx context, runID int32, metrics map[string]interface{},
) ([]Action, error) {
	timeStep, value, err := s.getMetric(metrics)
	if err != nil {
		return nil, err
	}

	if !s.SmallerIsBetter {
		*value *= -1
	}
	ops := s.stopRun(runID, *timeStep, *value)
	fmt.Printf("validation complete trial=%d, step=%v, metric=%v ops=%v runrungs=%v, rungs=%v\n", runID, *timeStep, *value, ops, s.RunRungs, s.Rungs)
	allTrials := len(s.RunRungs) - s.InvalidRuns
	if len(ops) > 0 && allTrials < s.MaxTrials() {
		create := NewCreate(
			ctx.rand, sampleAll(ctx.hparams, ctx.rand))
		ops = append(ops, create)
	}
	return ops, nil
}

func (s *asyncHalvingStoppingSearch) getMetric(metrics map[string]interface{}) (*uint64, *float64, error) {
	searcherMetric, ok := metrics[s.Metric].(float64)

	if !ok {
		return nil, nil, fmt.Errorf("error parsing searcher metric %s from validation metrics %v", s.Metric, metrics)
	}
	if !s.SmallerIsBetter {
		searcherMetric *= -1
	}

	unit := string(s.Length().Unit)
	stepNum, ok := metrics[unit].(float64)
	if !ok {
		return nil, nil, fmt.Errorf("error parsing searcher time metric (%s) in validation metrics (%v)", unit, metrics)
	}

	return ptrs.Ptr(uint64(stepNum)), &searcherMetric, nil
}

func (s *asyncHalvingStoppingSearch) stopRun(
	runID int32, timeStep uint64, metric float64,
) []Action {
	rungIndex := s.RunRungs[runID]
	var actions []Action

	// Starting at current rung, check for trials to early-stop.
	// Since validations aren't controlled by searcher, they could complete > 1 rungs at a time.
	for r := rungIndex; r < s.NumRungs(); r++ {
		rung := s.Rungs[r]
		s.RunRungs[runID] = r

		// If trial has not completed enough steps to qualify for this rung, exit.
		if timeStep < rung.UnitsNeeded {
			return actions
		}

		insertIndex := rung.insertMetric(runID, metric)

		// If this is the top rung, close the trial and exit.
		if r == s.NumRungs()-1 {
			actions = append(actions, NewStop(runID))
			//s.ClosedTrials[requestID] = true
			return actions
		}

		// Top 1/divisor trials should continue, trials - 1/divisor trials should be stopped.
		// If trials < divisor, continue if this is the best performing trial so far.
		numContinue := mathx.Max(int(float64(len(rung.Metrics))/s.Divisor()), 1)

		if insertIndex >= numContinue {
			actions = append(actions, NewStop(runID))
			return actions
		}

		// Continue to next rung.
	}

	return actions
}

func (s *asyncHalvingStoppingSearch) progress(
	map[int32]float64, map[int32]bool,
) float64 {
	allTrials := len(s.Rungs[0].Metrics)
	// Give ourselves an overhead of 20% of maxTrials when calculating progress.
	progress := float64(allTrials) / (1.2 * float64(s.MaxTrials()))
	if allTrials == s.MaxTrials() {
		numValidTrials := float64(s.RunsCompleted) - float64(s.InvalidRuns)
		progressNoOverhead := numValidTrials / float64(s.MaxTrials())
		progress = math.Max(progressNoOverhead, progress)
	}
	return progress
}

func (s *asyncHalvingStoppingSearch) runExitedEarly(
	ctx context, runID int32, exitedReason model.ExitedReason,
) ([]Action, error) {
	if exitedReason == model.InvalidHP || exitedReason == model.InitInvalidHP {
		var actions []Action
		s.EarlyExitRuns[runID] = true
		actions = append(actions, NewStop(runID))
		//s.ClosedTrials[requestID] = true
		s.InvalidRuns++
		// Remove metrics associated with InvalidHP trial across all rungs
		highestRungIndex := s.RunRungs[runID]
		for rungIndex := 0; rungIndex <= highestRungIndex; rungIndex++ {
			rung := s.Rungs[rungIndex]
			for i, trialMetric := range rung.Metrics {
				if trialMetric.RunID == runID {
					rung.Metrics = append(rung.Metrics[:i], rung.Metrics[i+1:]...)
					break
				}
			}
		}
		// Add new trial to searcher queue
		create := NewCreate(
			ctx.rand, sampleAll(ctx.hparams, ctx.rand))
		actions = append(actions, create)
		return actions, nil
	}
	s.EarlyExitRuns[runID] = true

	var actions []Action
	rungIndex := s.RunRungs[runID]
	rung := s.Rungs[rungIndex]

	rung.insertMetric(runID, ashaExitedMetricValue)

	allTrials := len(s.RunRungs) - s.InvalidRuns
	if allTrials < s.MaxTrials() {
		create := NewCreate(
			ctx.rand, sampleAll(ctx.hparams, ctx.rand))
		actions = append(actions, create)
	}
	return actions, nil
}

func (s *asyncHalvingStoppingSearch) Type() SearchMethodType {
	return s.SearchMethodType
}
