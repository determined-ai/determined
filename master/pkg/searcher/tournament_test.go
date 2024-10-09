//nolint:exhaustruct
package searcher

import (
	"fmt"
	"github.com/determined-ai/determined/master/pkg/ptrs"
	"github.com/determined-ai/determined/master/pkg/schemas/expconf"
	"github.com/stretchr/testify/require"
	"testing"
)

func TestAdaptiveASHASearchMethod(t *testing.T) {
	maxConcurrentTrials := 3
	maxTrials := 9
	maxRungs := 5
	divisor := 3.0
	maxTime := 90
	metric := "loss"
	config := expconf.AdaptiveASHAConfig{
		RawMaxTime:             &maxTime,
		RawDivisor:             &divisor,
		RawMaxRungs:            &maxRungs,
		RawMaxConcurrentTrials: &maxConcurrentTrials,
		RawMaxTrials:           &maxTrials,
		RawTimeMetric:          ptrs.Ptr("batches"),
		RawMode:                ptrs.Ptr(expconf.StandardMode),
	}
	searcherConfig := expconf.SearcherConfig{
		RawAdaptiveASHAConfig: &config,
		RawSmallerIsBetter:    ptrs.Ptr(true),
		RawMetric:             ptrs.Ptr(metric),
	}
	intHparam := &expconf.IntHyperparameter{RawMaxval: 10, RawCount: ptrs.Ptr(3)}
	hparams := expconf.Hyperparameters{
		"x": expconf.Hyperparameter{RawIntHyperparameter: intHparam},
	}

	// Create a new test searcher and verify brackets/rungs.
	testSearchRunner := NewTestSearchRunner(t, searcherConfig, hparams)
	search := testSearchRunner.method.(*tournamentSearch)
	expectedRungs := []*rung{
		{UnitsNeeded: uint64(10)},
		{UnitsNeeded: uint64(30)},
		{UnitsNeeded: uint64(90)},
	}

	// With max concurrent runs 3 and standard mode, expect 3 initial runs created
	// across 2 brackets, 2 runs in 1 and 1 in the other.
	runsCreated, runsStopped := testSearchRunner.start()
	require.Len(t, runsStopped, 0)
	require.Len(t, runsCreated, maxConcurrentTrials)

	// Verify expected rungs.
	require.Len(t, search.subSearches, 2)
	require.Len(t, search.RunTable, maxConcurrentTrials)
	for _, s := range search.RunTable {
		ashaSearch := search.subSearches[s].(*asyncHalvingStoppingSearch)
		require.Equal(t, expectedRungs[s:], ashaSearch.Rungs)
	}

	bracketRuns := make(map[int][]testRun)
	for _, runCreated := range runsCreated {
		runBracket := search.RunTable[runCreated.id]
		bracketRuns[runBracket] = append(bracketRuns[runBracket], runCreated)
	}
	require.Len(t, bracketRuns, 2)
	require.Len(t, bracketRuns[0], 2)
	require.Len(t, bracketRuns[1], 1)

	// Bracket 1
	bracket1 := bracketRuns[0]
	var stoppedRuns []testRun
	startingMetric := 1.0
	// "Train" all runs to the first rung target. Since we're reporting progressively worse metrics,
	// only the first run should continue to the next rungs.
	for len(bracket1) > 0 {
		var created []testRun
		for _, rc := range bracket1 {
			startingMetric += 1
			creates, stops := testSearchRunner.reportValidationMetric(rc.id, 10, startingMetric)
			stoppedRuns = append(stoppedRuns, stops...)
			created = append(created, creates...)
		}
		bracket1 = created
	}
	// 6 max runs in bracket 1, all except first one stopped.
	require.Equal(t, 5, len(stoppedRuns))
	// First run continues through rest of rungs.
	runsCreated, runsStopped = testSearchRunner.reportValidationMetric(bracketRuns[0][0].id, 30, 1.0)
	require.Len(t, runsStopped, 0)
	require.Len(t, runsCreated, 0)
	runsCreated, runsStopped = testSearchRunner.reportValidationMetric(bracketRuns[0][0].id, 90, 1.0)
	require.Len(t, runsStopped, 1)
	require.Len(t, runsCreated, 0)

	// Bracket 2
	// 3 max runs, 1 concurrent
	bracket2 := bracketRuns[1]

	// Run #1 (initial run) continues all rungs and creates new run (Run #2) when completed top rung.
	runsCreated, runsStopped = testSearchRunner.reportValidationMetric(bracket2[0].id, 30, 1.0)
	require.Len(t, runsStopped, 0)
	require.Len(t, runsCreated, 0)
	runsCreated, runsStopped = testSearchRunner.reportValidationMetric(bracket2[0].id, 90, 1.0)
	require.Len(t, runsStopped, 1)
	require.Len(t, runsCreated, 1)

	// Report worse metrics for Run #2, stop and create Run #3.
	runsCreated, runsStopped = testSearchRunner.reportValidationMetric(runsCreated[0].id, 30, 2.0)
	require.Len(t, runsStopped, 1)
	require.Len(t, runsCreated, 1)

	// Report worse metrics for Run #3, stops and does not create new run (bracket max runs = 3).
	runsCreated, runsStopped = testSearchRunner.reportValidationMetric(runsCreated[0].id, 30, 2.0)
	require.Len(t, runsStopped, 1)
	require.Len(t, runsCreated, 0)
}

func TestAdaptiveASHA(t *testing.T) {
	maxConcurrentTrials := 3
	maxTrials := 9
	maxRungs := 5
	divisor := 3.0
	maxTime := 90
	metric := "loss"
	config := expconf.AdaptiveASHAConfig{
		RawMaxTime:             &maxTime,
		RawDivisor:             &divisor,
		RawMaxRungs:            &maxRungs,
		RawMaxConcurrentTrials: &maxConcurrentTrials,
		RawMaxTrials:           &maxTrials,
		RawTimeMetric:          ptrs.Ptr("batches"),
		RawMode:                ptrs.Ptr(expconf.StandardMode),
	}
	searcherConfig := expconf.SearcherConfig{
		RawAdaptiveASHAConfig: &config,
		RawSmallerIsBetter:    ptrs.Ptr(true),
		RawMetric:             ptrs.Ptr(metric),
	}
	intHparam := &expconf.IntHyperparameter{RawMaxval: 10, RawCount: ptrs.Ptr(3)}
	hparams := expconf.Hyperparameters{
		"x": expconf.Hyperparameter{RawIntHyperparameter: intHparam},
	}

	// Create a new test searcher and verify brackets/rungs.
	testSearchRunner := NewTestSearchRunner(t, searcherConfig, hparams)
	testSearchRunner.trainLoop()
	fmt.Printf("runs: %v\n", testSearchRunner.runs)
}
