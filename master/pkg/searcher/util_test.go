package searcher

import (
	"fmt"
	"testing"

	"github.com/determined-ai/determined/master/pkg/model"

	"gotest.tools/assert"

	"github.com/determined-ai/determined/master/pkg/schemas/expconf"
)

type TestSearchRunner struct {
	config   expconf.SearcherConfig
	searcher *Searcher
	method   SearchMethod
	trials   []*testTrial
	t        *testing.T
}

type testTrial struct {
	requestID model.RequestID
	hparams   HParamSample
	stopped   bool
	stoppedAt int
	completed bool
}

func (tr testTrial) String() string {
	return fmt.Sprintf(
		"testTrial{requestID: %v, hparams: %v, stopped: %v, stoppedAt: %v, completed: %v}",
		tr.requestID, tr.hparams, tr.stopped, tr.stoppedAt, tr.completed,
	)
}

func mockRequestID(id int) model.RequestID {
	return model.RequestID{byte(id)}
}

func (sr *TestSearchRunner) run(maxUnits int, valPeriod int, increasing bool) {
	metric := 0.0
	sr.initialRuns()
	for i := 0; i < len(sr.trials); i++ {
		trial := sr.trials[i]
		for j := 0; j <= maxUnits; j += valPeriod {
			if increasing {
				metric++
			} else {
				metric--
			}
			sr.reportValidationMetric(trial.requestID, j, metric)
			if trial.stopped {
				trial.stoppedAt = j
				break
			}
		}
		sr.closeRun(trial.requestID)
	}
}

func NewTestSearchRunner(
	t *testing.T, config expconf.SearcherConfig, hparams expconf.Hyperparameters,
) *TestSearchRunner {
	expSeed := uint32(102932948)
	method := NewSearchMethod(config)
	searcher := NewSearcher(expSeed, method, hparams)
	return &TestSearchRunner{
		t:        t,
		config:   config,
		searcher: searcher,
		method:   method,
		trials:   []*testTrial{},
	}
}

func (sr *TestSearchRunner) initialRuns() ([]testTrial, []testTrial) {
	creates, err := sr.searcher.InitialTrials()
	assert.NilError(sr.t, err, "error getting initial trials")
	created, stopped := sr.handleActions(creates)
	return created, stopped
}

func (sr *TestSearchRunner) reportValidationMetric(
	requestID model.RequestID, stepNum int, metricVal float64,
) ([]testTrial, []testTrial) {
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
	actions, err := sr.searcher.ValidationCompleted(requestID, metrics)
	assert.NilError(sr.t, err, "error completing validation")

	created, stopped := sr.handleActions(actions)

	return created, stopped
}

// closeRun simulates a run completing its train loop and exiting.
func (sr *TestSearchRunner) closeRun(requestID model.RequestID) ([]testTrial, []testTrial) {
	run := sr.getTrialByRequestID(requestID)
	run.completed = true
	actions, err := sr.searcher.TrialExited(requestID)
	assert.NilError(sr.t, err, "error closing run")
	return sr.handleActions(actions)
}

func (sr *TestSearchRunner) getTrialByRequestID(requestID model.RequestID) *testTrial {
	for i, t := range sr.trials {
		if t.requestID == requestID {
			return sr.trials[i]
		}
	}
	return nil
}

func (sr *TestSearchRunner) handleActions(actions []Action) ([]testTrial, []testTrial) {
	var trialsCreated []testTrial
	var trialsStopped []testTrial

	for _, action := range actions {
		switch action := action.(type) {
		case Create:
			run := testTrial{requestID: action.RequestID, hparams: action.Hparams}
			_, err := sr.searcher.TrialCreated(action.RequestID)
			assert.NilError(sr.t, err, "error creating run")
			sr.trials = append(sr.trials, &run)
			trialsCreated = append(trialsCreated, run)
		case Stop:
			trial := sr.getTrialByRequestID(action.RequestID)
			trial.stopped = true
			trialsStopped = append(trialsStopped, *trial)
		}
	}
	return trialsCreated, trialsStopped
}
