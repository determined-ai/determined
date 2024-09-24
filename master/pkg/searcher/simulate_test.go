package searcher

import (
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/determined-ai/determined/master/pkg/ptrs"
	"github.com/determined-ai/determined/master/pkg/schemas/expconf"
)

func TestSimulateASHA(t *testing.T) {
	maxConcurrentTrials := 5
	maxTrials := 10
	divisor := 3.0
	maxTime := 900
	timeMetric := ptrs.Ptr("batches")
	config := expconf.SearcherConfig{
		RawAdaptiveASHAConfig: &expconf.AdaptiveASHAConfig{
			RawMaxRungs:            ptrs.Ptr(10),
			RawMaxTime:             &maxTime,
			RawDivisor:             &divisor,
			RawMaxConcurrentTrials: &maxConcurrentTrials,
			RawMaxTrials:           &maxTrials,
			RawTimeMetric:          timeMetric,
			RawMode:                ptrs.Ptr(expconf.StandardMode),
		},
		RawMetric:          ptrs.Ptr("loss"),
		RawSmallerIsBetter: ptrs.Ptr(true),
	}
	intHparam := &expconf.IntHyperparameter{RawMaxval: 10, RawCount: ptrs.Ptr(3)}
	hparams := expconf.Hyperparameters{
		"x": expconf.Hyperparameter{RawIntHyperparameter: intHparam},
	}

	res, err := Simulate(config, hparams)
	require.NoError(t, err)
	// Bracket #1: 7 total runs
	// Rungs: [100, 300, 900]
	// - 7 at 100 -> 2 at 300 -> 1 at 900
	// => 5 for 100, 1 for 300, 1 for 900
	//
	// Bracket #2: 3 total runs
	// Rungs: [300, 900]
	// - 3 at 300 -> 1 at 900
	// => 2 for 300, 1 for 900
	require.Equal(t, config, res.Config)
	expectedRunSummary := []TrialSummary{
		{Count: 5, Unit: SearchUnit{Name: timeMetric, Value: ptrs.Ptr(int32(100))}},
		{Count: 3, Unit: SearchUnit{Name: timeMetric, Value: ptrs.Ptr(int32(300))}},
		{Count: 2, Unit: SearchUnit{Name: timeMetric, Value: ptrs.Ptr(int32(900))}},
	}
	require.Equal(t, expectedRunSummary, res.Trials)
}

func TestSimulateGrid(t *testing.T) {
	maxConcurrentTrials := 2
	numHparams := 4
	gridConfig := expconf.GridConfig{
		RawMaxConcurrentTrials: ptrs.Ptr(maxConcurrentTrials),
	}
	searcherConfig := expconf.SearcherConfig{
		RawGridConfig: &gridConfig,
		RawMetric:     ptrs.Ptr("loss"),
	}
	hparams := expconf.Hyperparameters{
		"a": expconf.Hyperparameter{
			RawIntHyperparameter: &expconf.IntHyperparameter{
				RawMinval: 0, RawMaxval: 10, RawCount: ptrs.Ptr(numHparams),
			},
		},
	}

	res, err := Simulate(searcherConfig, hparams)
	require.NoError(t, err)

	// Expect all configured hparams in space = 4 runs at max length.
	require.Equal(t, searcherConfig, res.Config)
	expectedRunSummary := []TrialSummary{
		{Count: numHparams, Unit: SearchUnit{MaxLength: true}},
	}
	require.Equal(t, expectedRunSummary, res.Trials)
}
