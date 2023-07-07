package db

import (
	"fmt"
	"testing"

	"github.com/stretchr/testify/require"
	"google.golang.org/protobuf/types/known/structpb"

	"github.com/determined-ai/determined/master/pkg/model"
)

func TestMetricsBodyToJSON(t *testing.T) {
	avgMetrics := map[string]any{
		"loss": 1.0,
	}
	avgMetricsStruct, err := structpb.NewStruct(avgMetrics)
	require.NoError(t, err)

	cases := []metricsBody{
		{
			Type:       "generic",
			AvgMetrics: avgMetricsStruct,
		},
		*newMetricsBody(avgMetricsStruct, nil, "generic"),
		*newMetricsBody(avgMetricsStruct, nil, model.TrainingMetricType),
		*newMetricsBody(avgMetricsStruct, nil, model.ValidationMetricType),
		{
			Type:       model.ValidationMetricType,
			AvgMetrics: avgMetricsStruct,
		},
		{
			Type:         model.ValidationMetricType,
			AvgMetrics:   avgMetricsStruct,
			BatchMetrics: avgMetricsStruct,
		},
	}

	for idx, body := range cases {
		t.Run(fmt.Sprint(idx), func(t *testing.T) {
			json := body.ToJSONObj()
			_, ok := (*json)["batch_metrics"]
			require.False(t, ok)
			key := model.TrialMetricsJSONPath(body.Type == model.ValidationMetricType)
			_, ok = (*json)[key]
			require.True(t, ok)
		})
	}
}
