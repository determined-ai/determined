//nolint:exhaustruct
package searcher

import (
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/determined-ai/determined/master/pkg/model"
	"github.com/determined-ai/determined/master/pkg/ptrs"
	"github.com/determined-ai/determined/master/pkg/schemas"
	"github.com/determined-ai/determined/master/pkg/schemas/expconf"
)

func TestMakeRungs(t *testing.T) {
	cases := []struct {
		numRungs      int
		maxTime       uint64
		divisor       float64
		expectedRungs []*rung
	}{
		{
			numRungs: 3,
			maxTime:  9,
			divisor:  float64(3),
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
			numRungs: 4,
			maxTime:  10,
			divisor:  float64(2),
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
			numRungs: 1,
			maxTime:  9,
			divisor:  float64(3),
			expectedRungs: []*rung{
				{
					UnitsNeeded: 9,
				},
			},
		},
		{
			numRungs: 3,
			maxTime:  900,
			divisor:  float64(3),
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
		rungs := makeRungs(c.numRungs, c.divisor, c.maxTime)
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
		Metrics:     []runMetric{},
	}
	for _, c := range cases {
		var currentMetrics []runMetric
		for _, m := range c.existingMetrics {
			currentMetrics = append(currentMetrics, runMetric{
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
			require.InEpsilon(t, c.expectedMetric, *searcherMetric, 0.001, "searcher metric value doesn't match")
		}
	}
}

func TestStopTrials(t *testing.T) {
	type testMetric struct {
		runID    int32
		timeStep uint64
		metric   float64
	}

	cases := []struct {
		name             string
		rungs            []*rung
		runRungs         map[int32]int
		divisor          float64
		metric           testMetric
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
			metric: testMetric{
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
					Metrics: []runMetric{
						{
							TrialID: 1,
							Metric:  model.ExtendedFloat64(0.5),
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
					Metrics: []runMetric{
						{
							TrialID: 1,
							Metric:  model.ExtendedFloat64(0.5),
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
			metric: testMetric{
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
					Metrics: []runMetric{
						{
							TrialID: 2,
							Metric:  model.ExtendedFloat64(0.4),
						},
						{
							TrialID: 1,
							Metric:  model.ExtendedFloat64(0.5),
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
					Metrics: []runMetric{
						{
							TrialID: 1,
							Metric:  model.ExtendedFloat64(0.5),
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
			metric: testMetric{
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
					Metrics: []runMetric{
						{
							TrialID: 1,
							Metric:  model.ExtendedFloat64(0.5),
						},
						{
							TrialID: 2,
							Metric:  model.ExtendedFloat64(0.6),
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
			expectedOps: []Action{Stop{TrialID: 2}},
		},
	}

	searcher := &asyncHalvingStoppingSearch{}
	for _, c := range cases {
		searcher.TrialRungs = c.runRungs
		searcher.Rungs = c.rungs
		searcher.AsyncHalvingConfig.RawDivisor = &c.divisor
		numRungs := len(c.rungs)
		searcher.AsyncHalvingConfig.RawNumRungs = &numRungs
		ops := searcher.stopRun(c.metric.runID, c.metric.timeStep, c.metric.metric)
		require.Equal(t, c.expectedOps, ops)
		require.Equal(t, c.expectedRungs, searcher.Rungs)
		require.Equal(t, c.expectedRunRungs, searcher.TrialRungs)
	}
}

func TestASHAStoppingSearchMethod(t *testing.T) {
	maxConcurrentTrials := 3
	maxTrials := 10
	divisor := 3.0
	maxTime := 900
	metric := "val_loss"
	config := expconf.AsyncHalvingConfig{
		RawMaxTime:             &maxTime,
		RawDivisor:             &divisor,
		RawNumRungs:            ptrs.Ptr(3),
		RawMaxConcurrentTrials: &maxConcurrentTrials,
		RawMaxTrials:           &maxTrials,
		RawTimeMetric:          ptrs.Ptr("batches"),
	}
	searcherConfig := expconf.SearcherConfig{
		RawAsyncHalvingConfig: &config,
		RawSmallerIsBetter:    ptrs.Ptr(true),
		RawMetric:             ptrs.Ptr(metric),
	}
	config = schemas.WithDefaults(config)
	searcherConfig = schemas.WithDefaults(searcherConfig)

	intHparam := &expconf.IntHyperparameter{RawMaxval: 10, RawCount: ptrs.Ptr(3)}
	hparams := expconf.Hyperparameters{
		"x": expconf.Hyperparameter{RawIntHyperparameter: intHparam},
	}

	// Create a new test searcher and verify brackets/rungs.
	testSearchRunner := NewTestSearchRunner(t, searcherConfig, hparams)
	search := testSearchRunner.method.(*asyncHalvingStoppingSearch)

	expectedRungs := []*rung{
		{UnitsNeeded: uint64(100)},
		{UnitsNeeded: uint64(300)},
		{UnitsNeeded: uint64(900)},
	}

	require.Equal(t, expectedRungs, search.Rungs)

	// Simulate the search.
	testSearchRunner.run(900, 100, true)
	progress := search.progress(map[int32]float64{}, map[int32]bool{})
	require.InEpsilon(t, 1.0, progress, 0.01)

	// Expect 10 total runs.
	// Since we reported progressively worse metrics, only the first run should continue.
	require.Len(t, testSearchRunner.runs, maxTrials)
	for i, tr := range testSearchRunner.runs {
		if i == 0 {
			require.Equal(t, 900, tr.stoppedAt)
		} else {
			require.Equal(t, 100, tr.stoppedAt)
		}
	}
}
