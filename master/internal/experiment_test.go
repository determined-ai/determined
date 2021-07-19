package internal

import (
	"testing"

	"github.com/determined-ai/determined/master/pkg/workload"

	"gotest.tools/assert"

	"github.com/determined-ai/determined/master/pkg/model"
	"github.com/determined-ai/determined/master/pkg/ptrs"
	"github.com/determined-ai/determined/master/pkg/schemas/expconf"
)

type metricCase struct {
	value  float64
	isBest bool
}

func testBestValidationCase(t *testing.T, smallerIsBetter bool, metrics []metricCase) {
	exp := &experiment{
		Experiment: &model.Experiment{
			Config: expconf.ExperimentConfig{
				RawSearcher: &expconf.SearcherConfig{
					RawMetric:          ptrs.StringPtr("metric"),
					RawSmallerIsBetter: &smallerIsBetter,
				},
			},
		},
	}

	for _, metric := range metrics {
		msg := workload.CompletedMessage{
			ValidationMetrics: &workload.ValidationMetrics{
				Metrics: map[string]interface{}{"metric": metric.value},
			},
		}
		isBest := exp.isBestValidation(*msg.ValidationMetrics)
		assert.Equal(t, metric.isBest, isBest, "failed on metric value %f", metric.value)
	}
}

func (e *experiment) isBestValidation(metrics workload.ValidationMetrics) bool {
	metricName := e.Config.Searcher().Metric()
	validation, err := metrics.Metric(metricName)
	if err != nil {
		// TODO: Better error handling here.
		return false
	}
	smallerIsBetter := e.Config.Searcher().SmallerIsBetter()
	isBest := (e.BestValidation == nil) ||
		(smallerIsBetter && validation <= *e.BestValidation) ||
		(!smallerIsBetter && validation >= *e.BestValidation)
	if isBest {
		e.BestValidation = &validation
	}
	return isBest
}

func TestBestValidation(t *testing.T) {
	testBestValidationCase(
		t,
		true,
		[]metricCase{{5, true}, {9, false}, {4, true}, {10, false}, {7, false}, {3, true}},
	)
	testBestValidationCase(
		t,
		false,
		[]metricCase{{5, true}, {9, true}, {4, false}, {10, true}, {7, false}, {3, false}},
	)
}
