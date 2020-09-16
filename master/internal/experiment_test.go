package internal

import (
	"github.com/determined-ai/determined/master/pkg/workload"
	"testing"

	"gotest.tools/assert"

	"github.com/determined-ai/determined/master/pkg/model"
)

type metricCase struct {
	value  float64
	isBest bool
}

func testBestValidationCase(t *testing.T, smallerIsBetter bool, metrics []metricCase) {
	exp := &experiment{
		Experiment: &model.Experiment{
			Config: model.ExperimentConfig{
				Searcher: model.SearcherConfig{
					Metric:          "metric",
					SmallerIsBetter: smallerIsBetter,
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
