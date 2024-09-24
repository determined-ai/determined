//nolint:exhaustruct
package searcher

import (
	"github.com/determined-ai/determined/master/pkg/model"
	"github.com/determined-ai/determined/master/pkg/ptrs"
	"github.com/determined-ai/determined/master/pkg/schemas"
	"github.com/determined-ai/determined/master/pkg/schemas/expconf"
	"github.com/stretchr/testify/require"
	"testing"
)

func TestMakeRungs(t *testing.T) {
	cases := []struct {
		numRungs      int
		maxLength     uint64
		divisor       float64
		expectedRungs []*rung
	}{
		{
			numRungs:  3,
			maxLength: 9,
			divisor:   float64(3),
			expectedRungs: []*rung{
				{
					UnitsNeeded: 1,
				},
				{
					UnitsNeeded: 3,
				},
				{
					UnitsNeeded: 9,
				},
			},
		},
		{
			numRungs:  4,
			maxLength: 10,
			divisor:   float64(2),
			expectedRungs: []*rung{
				{
					UnitsNeeded: 1,
				},
				{
					UnitsNeeded: 2,
				},
				{
					UnitsNeeded: 5,
				},
				{
					UnitsNeeded: 10,
				},
			},
		},
		{
			numRungs:  1,
			maxLength: 9,
			divisor:   float64(3),
			expectedRungs: []*rung{
				{
					UnitsNeeded: 9,
				},
			},
		},
		{
			numRungs:  3,
			maxLength: 900,
			divisor:   float64(3),
			expectedRungs: []*rung{
				{
					UnitsNeeded: 100,
				},
				{
					UnitsNeeded: 300,
				},
				{
					UnitsNeeded: 900,
				},
			},
		},
	}
	for _, c := range cases {
		rungs := makeRungs(c.numRungs, c.divisor, c.maxLength)
		require.Equal(t, c.expectedRungs, rungs)
	}
}

func TestInsertCompletedMetric(t *testing.T) {
	cases := []struct {
		newMetric           float64
		existingMetrics     []float64
		expectedInsertIndex int
		expectedMetrics     []float64
	}{
		{
			newMetric:           1.2,
			existingMetrics:     []float64{0.0, 1.5, 2.1},
			expectedInsertIndex: 1,
			expectedMetrics:     []float64{0.0, 1.2, 1.5, 2.1},
		},
		{
			newMetric:           3.0,
			existingMetrics:     []float64{0.0, 1.5, 2.0},
			expectedInsertIndex: 3,
			expectedMetrics:     []float64{0.0, 1.5, 2.0, 3.0},
		},
		{
			newMetric:           -3.4,
			existingMetrics:     []float64{-3.0, -2.0, -1.0},
			expectedInsertIndex: 0,
			expectedMetrics:     []float64{-3.4, -3.0, -2.0, -1.0},
		},
		{
			newMetric:           1.2,
			existingMetrics:     []float64{},
			expectedInsertIndex: 0,
			expectedMetrics:     []float64{1.2},
		},
	}
	rung := rung{
		UnitsNeeded: 0,
		Metrics:     []trialMetric{},
	}
	for _, c := range cases {
		var currentMetrics []trialMetric
		for _, m := range c.existingMetrics {
			currentMetrics = append(currentMetrics, trialMetric{
				Metric: model.ExtendedFloat64(m),
			})
		}
		rung.Metrics = currentMetrics
		insertIndex := rung.insertMetric(1, c.newMetric)
		var addedMetrics []float64
		for _, m := range rung.Metrics {
			addedMetrics = append(addedMetrics, float64(m.Metric))
		}
		require.Equal(t, c.expectedInsertIndex, insertIndex)
		require.Equal(t, c.expectedMetrics, addedMetrics)
	}
}

func TestGetMetric(t *testing.T) {
	cases := []struct {
		metrics          map[string]interface{}
		metricName       string
		timeMetricName   string
		timeMetric       int
		smallerIsBetter  bool
		expectedTimeStep int
		expectedMetric   float64
		expectedError    string
	}{
		{
			metrics:          map[string]interface{}{"loss": 0.25, "batches": 2.0},
			metricName:       "loss",
			timeMetricName:   "batches",
			smallerIsBetter:  true,
			expectedTimeStep: 2,
			expectedMetric:   0.25,
		},
		{
			metrics:          map[string]interface{}{"loss": 0.2, "batches": 3.0},
			metricName:       "loss",
			timeMetricName:   "batches",
			smallerIsBetter:  false,
			expectedTimeStep: 3,
			expectedMetric:   -0.2,
		},
		{
			metrics:          map[string]interface{}{"loss": 1.2, "custom_time_step": 5.0},
			metricName:       "loss",
			timeMetricName:   "custom_time_step",
			smallerIsBetter:  true,
			expectedTimeStep: 5,
			expectedMetric:   1.2,
		},
		{
			metrics:         model.JSONObj{"batches": 2.0},
			metricName:      "loss",
			timeMetricName:  "batches",
			smallerIsBetter: true,
			expectedError:   "error parsing searcher metric",
		},
	}

	searcher := &asyncHalvingStoppingSearch{}
	for _, c := range cases {
		searcher.Metric = c.metricName
		searcher.RawTimeMetric = &c.timeMetricName
		searcher.SmallerIsBetter = c.smallerIsBetter
		stepNum, searcherMetric, err := searcher.getMetric(c.metrics)
		if c.expectedError != "" {
			require.ErrorContains(t, err, c.expectedError)
		} else {
			require.NoError(t, err, "got unexpected error %v: %v", err, c)
			require.Equal(t, uint64(c.expectedTimeStep), *stepNum, "time step does not match")
			require.Equal(t, c.expectedMetric, *searcherMetric, "searcher metric value doesn't match")
		}
	}
}

func TestStopTrials(t *testing.T) {
	type runMetric struct {
		runID    int32
		timeStep uint64
		metric   float64
	}

	cases := []struct {
		name             string
		rungs            []*rung
		runRungs         map[int32]int
		divisor          float64
		metric           runMetric
		expectedOps      []Action
		expectedRunRungs map[int32]int
		expectedRungs    []*rung
	}{
		{
			name: "first validation",
			rungs: []*rung{
				{
					UnitsNeeded: 1,
				},
				{
					UnitsNeeded: 3,
				},
				{
					UnitsNeeded: 9,
				},
			},
			runRungs: map[int32]int{
				1: 0,
			},
			divisor: 3.0,
			metric: runMetric{
				runID:    1,
				timeStep: 1,
				metric:   0.5,
			},
			expectedRunRungs: map[int32]int{
				1: 1,
			},
			expectedRungs: []*rung{
				{
					UnitsNeeded: 1,
					Metrics: []trialMetric{
						{
							RunID:  1,
							Metric: model.ExtendedFloat64(0.5),
						},
					},
				},
				{
					UnitsNeeded: 3,
				},
				{
					UnitsNeeded: 9,
				},
			},
			expectedOps: []Action(nil),
		},
		{
			name: "second validation better than first",
			rungs: []*rung{
				{
					UnitsNeeded: 1,
					Metrics: []trialMetric{
						{
							RunID:  1,
							Metric: model.ExtendedFloat64(0.5),
						},
					},
				},
				{
					UnitsNeeded: 3,
				},
				{
					UnitsNeeded: 9,
				},
			},
			runRungs: map[int32]int{
				1: 1,
				2: 0,
			},
			divisor: 3.0,
			metric: runMetric{
				runID:    2,
				timeStep: 1,
				metric:   0.4,
			},
			expectedRunRungs: map[int32]int{
				1: 1,
				2: 1,
			},
			expectedRungs: []*rung{
				{
					UnitsNeeded: 1,
					Metrics: []trialMetric{
						{
							RunID:  2,
							Metric: model.ExtendedFloat64(0.4),
						},
						{
							RunID:  1,
							Metric: model.ExtendedFloat64(0.5),
						},
					},
				},
				{
					UnitsNeeded: 3,
				},
				{
					UnitsNeeded: 9,
				},
			},
			expectedOps: []Action(nil),
		},
		{
			name: "second validation worse than first",
			rungs: []*rung{
				{
					UnitsNeeded: 1,
					Metrics: []trialMetric{
						{
							RunID:  1,
							Metric: model.ExtendedFloat64(0.5),
						},
					},
				},
				{
					UnitsNeeded: 3,
				},
				{
					UnitsNeeded: 9,
				},
			},
			runRungs: map[int32]int{
				1: 1,
				2: 0,
			},
			divisor: 3.0,
			metric: runMetric{
				runID:    2,
				timeStep: 1,
				metric:   0.6,
			},
			expectedRunRungs: map[int32]int{
				1: 1,
				2: 0,
			},
			expectedRungs: []*rung{
				{
					UnitsNeeded: 1,
					Metrics: []trialMetric{
						{
							RunID:  1,
							Metric: model.ExtendedFloat64(0.5),
						},
						{
							RunID:  2,
							Metric: model.ExtendedFloat64(0.6),
						},
					},
				},
				{
					UnitsNeeded: 3,
				},
				{
					UnitsNeeded: 9,
				},
			},
			expectedOps: []Action{Stop{RunID: 2}},
		},
	}

	searcher := &asyncHalvingStoppingSearch{}
	for _, c := range cases {
		searcher.RunRungs = c.runRungs
		searcher.Rungs = c.rungs
		searcher.AsyncHalvingConfig.RawDivisor = &c.divisor
		numRungs := len(c.rungs)
		searcher.AsyncHalvingConfig.RawNumRungs = &numRungs
		ops := searcher.stopRun(c.metric.runID, c.metric.timeStep, c.metric.metric)
		require.Equal(t, c.expectedOps, ops)
		require.Equal(t, c.expectedRungs, searcher.Rungs)
		require.Equal(t, c.expectedRunRungs, searcher.RunRungs)
	}
}

func TestASHAStopping(t *testing.T) {
	maxConcurrentTrials := 3
	maxTrials := 10
	divisor := 3.0
	maxTime := 900
	smallerIsBetter := true
	metric := "loss"
	config := expconf.AsyncHalvingConfig{
		RawMaxTime:             &maxTime,
		RawDivisor:             &divisor,
		RawNumRungs:            ptrs.Ptr(3),
		RawMaxConcurrentTrials: &maxConcurrentTrials,
		RawMaxTrials:           &maxTrials,
		RawTimeMetric:          ptrs.Ptr(metric),
	}
	searcherConfig := expconf.SearcherConfig{
		RawAsyncHalvingConfig: &config,
		RawSmallerIsBetter:    ptrs.Ptr(smallerIsBetter),
		RawMetric:             ptrs.Ptr("loss"),
	}
	config = schemas.WithDefaults(config)
	searcherConfig = schemas.WithDefaults(searcherConfig)

	intHparam := &expconf.IntHyperparameter{RawMaxval: 10, RawCount: ptrs.Ptr(3)}
	hparams := expconf.Hyperparameters{
		"x": expconf.Hyperparameter{RawIntHyperparameter: intHparam},
	}

	// Create a new test searcher and verify brackets/rungs.
	method := newAsyncHalvingStoppingSearch(config, smallerIsBetter, metric)
	searcher := NewSearcher(uint32(102932948), method, hparams)
	testSearchRunner := &TestSearchRunner{t: t, config: searcherConfig, searcher: searcher, method: method, runs: make(map[int32]testRun)}
	search := testSearchRunner.method.(*asyncHalvingStoppingSearch)

	expectedRungs := []*rung{
		{UnitsNeeded: uint64(100)},
		{UnitsNeeded: uint64(300)},
		{UnitsNeeded: uint64(900)},
	}

	require.Equal(t, expectedRungs, search.Rungs)

	// Start the search, validate correct number of initial runs created.
	runsCreated, runsStopped := testSearchRunner.start()
	require.Equal(t, maxConcurrentTrials, len(runsCreated))
	require.Equal(t, 0, len(runsStopped))

	startingMetric := 1.0

	// "Train" all runs to the first rung target. Since we're reporting progressively worse metrics,
	// only the first run should continue to the next rungs.
	var stoppedRuns []testRun

	for len(runsCreated) > 0 {
		var created []testRun
		for _, rc := range runsCreated {
			startingMetric += 1
			creates, stops := testSearchRunner.reportValidationMetric(rc.id, 100, startingMetric)
			stoppedRuns = append(stoppedRuns, stops...)
			created = append(created, creates...)
		}
		runsCreated = created
	}
	require.Equal(t, maxTrials-1, len(stoppedRuns))

	// Report metrics for second and third rungs. Run should continue.
	creates, stops := testSearchRunner.reportValidationMetric(0, 300, startingMetric)
	require.Equal(t, 0, len(creates))
	require.Equal(t, 0, len(stops))

	// Report metrics for last rung, run should stop.
	creates, stops = testSearchRunner.reportValidationMetric(0, 900, startingMetric)
	require.Equal(t, 0, len(creates))
	require.Equal(t, 1, len(stops))
}
