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
		RawRandomConfig: &expconf.RandomConfig{
			RawMaxTrials:           ptrs.Ptr(4),
			RawMaxConcurrentTrials: ptrs.Ptr(2),
		},
	}
	intHparam := &expconf.IntHyperparameter{RawMaxval: 10, RawCount: ptrs.Ptr(3)}
	hparams := expconf.Hyperparameters{
		"x": expconf.Hyperparameter{RawIntHyperparameter: intHparam},
	}
	testSearchRunner := NewTestSearchRunner(t, conf, hparams)

	// Expect 2 initial runs created.
	created, stopped := testSearchRunner.start()
	require.Len(t, created, 2)
	require.Len(t, stopped, 0)
	for _, r := range created {
		require.True(t, r.hparams["x"].(int) <= 10 && r.hparams["x"].(int) > 0)
	}
	run1, run2 := created[0], created[1]

	// Run 1 finished training, create run 3
	runsCreated, runsStopped := testSearchRunner.closeRun(run1.id)
	require.Len(t, runsCreated, 1)
	require.Len(t, runsStopped, 0)
	run3 := runsCreated[0]

	// Run 2 finished training, create run 4
	runsCreated, runsStopped = testSearchRunner.closeRun(run2.id)
	require.Len(t, runsCreated, 1)
	require.Len(t, runsStopped, 0)
	run4 := runsCreated[0]

	// Run 3 & 4 finished training, no new runs created.
	runsCreated, runsStopped = testSearchRunner.closeRun(run3.id)
	require.Len(t, runsCreated, 0)
	runsCreated, runsStopped = testSearchRunner.closeRun(run4.id)
	require.Len(t, runsCreated, 0)
}

func TestSingleSearchMethod(t *testing.T) {
	conf := expconf.SearcherConfig{
		RawSingleConfig: &expconf.SingleConfig{},
	}

	testSearchRunner := NewTestSearchRunner(t, conf, expconf.Hyperparameters{})

	// Single search should create exactly one run.
	created, stopped := testSearchRunner.start()
	require.Len(t, created, 1)
	require.Len(t, stopped, 0)

	run := created[0]

	// When the run is finished, no new runs should be created.
	created, stopped = testSearchRunner.closeRun(run.id)
	require.Len(t, created, 0)
}
