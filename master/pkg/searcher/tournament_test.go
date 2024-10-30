//nolint:exhaustruct
package searcher

import (
	"testing"

	"github.com/determined-ai/determined/master/pkg/model"

	"github.com/stretchr/testify/require"

	"github.com/determined-ai/determined/master/pkg/ptrs"
	"github.com/determined-ai/determined/master/pkg/schemas/expconf"
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

	// Create a new test searcher and verify correct brackets/rungs initialized.
	testSearchRunner := NewTestSearchRunner(t, searcherConfig, hparams)
	search := testSearchRunner.method.(*tournamentSearch)
	expectedRungs := []*rung{
		{UnitsNeeded: uint64(10)},
		{UnitsNeeded: uint64(30)},
		{UnitsNeeded: uint64(90)},
	}
	for i, s := range search.subSearches {
		ashaSearch := s.(*asyncHalvingStoppingSearch)
		require.Equal(t, expectedRungs[i:], ashaSearch.Rungs)
	}

	// Simulate running the search.
	testSearchRunner.run(90, 10, true)

	// Expect 2 brackets and 9 total trials.
	require.Len(t, search.subSearches, 2)
	require.Len(t, search.TrialTable, maxTrials)

	bracket1 := make(map[model.RequestID]*testTrial)
	bracket2 := make(map[model.RequestID]*testTrial)

	for _, tr := range testSearchRunner.trials {
		if search.TrialTable[tr.requestID] == 0 {
			bracket1[tr.requestID] = tr
		} else {
			bracket2[tr.requestID] = tr
		}
	}

	// Bracket #1: 6 total trials
	// Rungs: [10, 30, 90]
	// Since we reported progressively worse metrics, only one run continues to top rung.
	// All others are stopped at first rung.
	require.Len(t, bracket1, 6)
	stoppedAt90 := 0
	stoppedAt10 := 0
	for _, tr := range bracket1 {
		if tr.stoppedAt == 90 {
			stoppedAt90++
		}
		if tr.stoppedAt == 10 {
			stoppedAt10++
		}
	}
	require.Equal(t, 5, stoppedAt10)
	require.Equal(t, 1, stoppedAt90)

	// Bracket #2: 3 total trials
	// Rungs: [30, 90]
	// First run (run #3 from initialTrials) continues to top rung, two will stop at first rung.
	require.Len(t, bracket2, 3)
	stoppedAt90 = 0
	stoppedAt30 := 0
	for _, tr := range bracket2 {
		if tr.stoppedAt == 90 {
			stoppedAt90++
		}
		if tr.stoppedAt == 30 {
			stoppedAt30++
		}
	}
	require.Equal(t, 1, stoppedAt90)
	require.Equal(t, 2, stoppedAt30)
}
