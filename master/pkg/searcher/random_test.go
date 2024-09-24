//nolint:exhaustruct
package searcher

import (
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/determined-ai/determined/master/pkg/ptrs"
	"github.com/determined-ai/determined/master/pkg/schemas/expconf"
)

func TestRandomSearchMethod(t *testing.T) {
	conf := expconf.SearcherConfig{
		RawMetric: ptrs.Ptr("loss"),
		RawRandomConfig: &expconf.RandomConfig{
			RawMaxTrials:           ptrs.Ptr(4),
			RawMaxConcurrentTrials: ptrs.Ptr(2),
		},
	}
	intHparam := &expconf.IntHyperparameter{RawMaxval: 10, RawCount: ptrs.Ptr(4)}
	hparams := expconf.Hyperparameters{
		"x": expconf.Hyperparameter{RawIntHyperparameter: intHparam},
	}
	testSearchRunner := NewTestSearchRunner(t, conf, hparams)

	// Simulate a search and verify expected run states.
	testSearchRunner.run(100, 10, false)
	// 4 total runs created, each with hparam in space and run to completion.
	require.Len(t, testSearchRunner.runs, 4)
	for _, tr := range testSearchRunner.runs {
		hparam := tr.hparams["x"].(int)
		require.True(t, hparam <= 10 && hparam >= 0)
		require.False(t, tr.stopped)
	}
}

func TestSingleSearchMethod(t *testing.T) {
	conf := expconf.SearcherConfig{
		RawMetric:       ptrs.Ptr("loss"),
		RawSingleConfig: &expconf.SingleConfig{},
	}

	testSearchRunner := NewTestSearchRunner(t, conf, expconf.Hyperparameters{})

	// Simulate a search and verify expected run states.
	testSearchRunner.run(100, 10, false)

	// Single search should create exactly one run.
	require.Len(t, testSearchRunner.runs, 1)
	require.False(t, testSearchRunner.runs[0].stopped)
}
