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
			isValidation: false,
			AvgMetrics:   avgMetricsStruct,
		},
		*newMetricsBody(avgMetricsStruct, nil, false),
		*newMetricsBody(avgMetricsStruct, nil, false),
		*newMetricsBody(avgMetricsStruct, nil, true),
		{
			isValidation: true,
			AvgMetrics:   avgMetricsStruct,
		},
		{
			isValidation: true,
			AvgMetrics:   avgMetricsStruct,
			BatchMetrics: avgMetricsStruct,
		},
	}

	for idx, body := range cases {
		t.Run(fmt.Sprint(idx), func(t *testing.T) {
			json := body.ToJSONObj()
			_, ok := (*json)["batch_metrics"]
			if body.isValidation {
				require.False(t, ok)
			} else {
				// we can leave this out if it's empty but we keep it for backward compatibility.
				require.True(t, ok)
			}
			key := model.TrialMetricsJSONPath(body.isValidation)
			_, ok = (*json)[key]
			require.True(t, ok)
		})
	}
}
