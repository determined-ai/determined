package searcher

import (
	"testing"

	"gotest.tools/assert"

	"github.com/determined-ai/determined/master/pkg/schemas/expconf"
)

type TestSearchRunner struct {
	config   expconf.SearcherConfig
	searcher *Searcher
	method   SearchMethod
	runs     map[int32]testRun
	t        *testing.T
}

type testRun struct {
	id           int32
	hparams      HParamSample
	searchRunner *TestSearchRunner
}

func NewTestSearchRunner(t *testing.T, config expconf.SearcherConfig, hparams expconf.Hyperparameters) *TestSearchRunner {
	expSeed := uint32(102932948)
	method := NewSearchMethod(config)
	searcher := NewSearcher(expSeed, method, hparams)
	return &TestSearchRunner{t: t, config: config, searcher: searcher, method: method, runs: make(map[int32]testRun)}
}

func (sr *TestSearchRunner) start() ([]testRun, []testRun) {
	creates, err := sr.searcher.InitialRuns()
	assert.NilError(sr.t, err, "error getting initial runs")
	created, stopped := sr.handleActions(creates)
	return created, stopped
}

func (sr *TestSearchRunner) reportValidationMetric(runID int32, stepNum int, metricVal float64) ([]testRun, []testRun) {
	metrics := map[string]interface{}{
		sr.config.Metric(): metricVal,
	}
	if sr.config.RawAdaptiveASHAConfig != nil {
		timeMetric := string(sr.config.RawAdaptiveASHAConfig.Length().Unit)
		metrics[timeMetric] = float64(stepNum)
	}
	if sr.config.RawAsyncHalvingConfig != nil {
		timeMetric := string(sr.config.RawAsyncHalvingConfig.Length().Unit)
		metrics[timeMetric] = float64(stepNum)
	}
	actions, err := sr.searcher.ValidationCompleted(runID, metrics)
	assert.NilError(sr.t, err, "error completing validation")

	created, stopped := sr.handleActions(actions)

	return created, stopped
}

// closeRun simulates a run completing its train loop and exiting.
func (sr *TestSearchRunner) closeRun(runID int32) ([]testRun, []testRun) {
	actions, err := sr.searcher.RunClosed(runID)
	assert.NilError(sr.t, err, "error closing run")
	return sr.handleActions(actions)
}

func (sr *TestSearchRunner) handleActions(actions []Action) ([]testRun, []testRun) {
	var runsCreated []testRun
	var runsStopped []testRun

	for _, action := range actions {
		switch action := action.(type) {
		case Create:
			run := testRun{id: int32(len(sr.searcher.state.RunsCreated)), hparams: action.Hparams, searchRunner: sr}
			_, err := sr.searcher.RunCreated(run.id, action)
			assert.NilError(sr.t, err, "error creating run")

			sr.runs[run.id] = run
			runsCreated = append(runsCreated, run)
		case Stop:
			run := sr.runs[action.RunID]
			runsStopped = append(runsStopped, run)
		}
	}
	return runsCreated, runsStopped
}
