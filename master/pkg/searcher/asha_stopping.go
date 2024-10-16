package searcher

import (
	"encoding/json"
	"fmt"
	"math"
	"sort"

	"github.com/determined-ai/determined/master/pkg/ptrs"

	"github.com/determined-ai/determined/master/pkg/mathx"
	"github.com/determined-ai/determined/master/pkg/model"
	"github.com/determined-ai/determined/master/pkg/schemas/expconf"
)

// AsyncHalvingStoppingSearch implements a version of the asynchronous successive halving
// algorithm (ASHA) that early-stops worse performing runs rather than actively promoting better
// performing runs. When a new validation metric is reported, the searcher decides if the run
// should be stopped based on the ranking of the metric compared to other runs' metrics in the
// same rung.
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
		// EarlyExitRuns contains runs that exited early that are still considered in the search.
		EarlyExitRuns    map[int32]bool   `json:"early_exit_runs"`
		RunsCompleted    int              `json:"runs_completed"`
		InvalidRuns      int              `json:"invalid_runs"`
		SearchMethodType SearchMethodType `json:"search_method_type"`
	}

	runMetric struct {
		RunID  int32                 `json:"run_id"`
		Metric model.ExtendedFloat64 `json:"metric"`
	}
	rung struct {
		UnitsNeeded uint64      `json:"units_needed"`
		Metrics     []runMetric `json:"metrics"`
	}
)

func (r *rung) String() string {
	return fmt.Sprintf("Rung{UnitsNeeded: %d, Metrics: %v}", r.UnitsNeeded, r.Metrics)
}

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

// insertMetric adds a completed validation metric to the rung in the appropriate order of all
// the metrics in the rung thus far and returns the insert index.
func (r *rung) insertMetric(runID int32, metric float64) int {
	insertIndex := sort.Search(
		len(r.Metrics),
		func(i int) bool { return float64(r.Metrics[i].Metric) >= metric },
	)

	// Add metrics to state.
	r.Metrics = append(r.Metrics, runMetric{})
	copy(r.Metrics[insertIndex+1:], r.Metrics[insertIndex:])
	r.Metrics[insertIndex] = runMetric{
		RunID:  runID,
		Metric: model.ExtendedFloat64(metric),
	}
	return insertIndex
}

// initialRuns specifies the initial runs that the search will create.
// Since each run can only stop and create a new run, this effectively controls the degree of
// parallelism of the search.
func (s *asyncHalvingStoppingSearch) initialRuns(ctx context) ([]Action, error) {
	var actions []Action
	var maxConcurrentTrials int

	// Use searcher config fields to determine number of runs if set.
	// Otherwise, default to a number of runs that guarantees at least one run will continue
	// to the top rung.
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

func (s *asyncHalvingStoppingSearch) runExited(
	ctx context, runID int32,
) ([]Action, error) {
	s.RunsCompleted++
	return nil, nil
}

// validationCompleted handles every validation metric reported by a run and returns any resulting
// actions the searcher would like to take.
func (s *asyncHalvingStoppingSearch) validationCompleted(
	ctx context, runID int32, metrics map[string]interface{},
) ([]Action, error) {
	timeStep, value, err := s.getMetric(metrics)
	if err != nil {
		return nil, err
	}

	ops := s.stopRun(runID, *timeStep, *value)
	allTrials := len(s.RunRungs) - s.InvalidRuns
	if len(ops) > 0 && allTrials < s.MaxTrials() {
		create := NewCreate(ctx.rand, sampleAll(ctx.hparams, ctx.rand))
		ops = append(ops, create)
	}
	return ops, nil
}

// getMetric reads the searcher metric and time step value from the reported validation metrics.
func (s *asyncHalvingStoppingSearch) getMetric(metrics map[string]interface{}) (*uint64, *float64, error) {
	searcherMetric, ok := metrics[s.Metric].(float64)

	if !ok {
		return nil, nil, fmt.Errorf("error parsing searcher metric (%s) from validation metrics: %v", s.Metric, metrics)
	}
	if !s.SmallerIsBetter {
		searcherMetric *= -1
	}

	unit := string(s.Length().Unit)
	stepNum, ok := metrics[unit].(float64)
	if !ok {
		return nil, nil, fmt.Errorf("error parsing searcher time metric (%s) in validation metrics: %v", unit, metrics)
	}

	return ptrs.Ptr(uint64(stepNum)), &searcherMetric, nil
}

// stopRun handles early-stopping and record-keeping logic for a validation metric reported to the
// searcher.
// If the metric qualifies the run for a rung but is not in the top 1/divisor runs for that rung,
// stopRun will return a single `searcher.Stop` action. Otherwise, no actions will be returned.
func (s *asyncHalvingStoppingSearch) stopRun(
	runID int32, timeStep uint64, metric float64,
) []Action {
	rungIndex := s.RunRungs[runID]
	var actions []Action

	// Starting at current rung, check if run should continue to next rung or early-stop.
	// Since validations aren't controlled by searcher, they could complete > 1 rungs at a time.
	for r := rungIndex; r < s.NumRungs(); r++ {
		rung := s.Rungs[r]
		s.RunRungs[runID] = r

		// If run has not completed enough steps to qualify for this rung, exit.
		if timeStep < rung.UnitsNeeded {
			return actions
		}

		insertIndex := rung.insertMetric(runID, metric)

		// If this is the top rung, close the run and exit.
		if r == s.NumRungs()-1 {
			actions = append(actions, NewStop(runID))
			return actions
		}

		// Top 1/divisor runs should continue, runs - 1/divisor runs should be stopped.
		// If runs < divisor, continue only if this is the best performing run so far.
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
	// Give ourselves an overhead of 20% of maxRuns when calculating progress.
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
		create := NewCreate(ctx.rand, sampleAll(ctx.hparams, ctx.rand))
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
		create := NewCreate(ctx.rand, sampleAll(ctx.hparams, ctx.rand))
		actions = append(actions, create)
	}
	return actions, nil
}

func (s *asyncHalvingStoppingSearch) Type() SearchMethodType {
	return s.SearchMethodType
}
