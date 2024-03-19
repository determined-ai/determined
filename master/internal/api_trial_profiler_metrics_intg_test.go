//go:build integration
// +build integration

package internal

import (
	"context"
	"encoding/json"
	"math/rand"
	"testing"
	"time"

	structpb "github.com/golang/protobuf/ptypes/struct"
	"google.golang.org/protobuf/encoding/protojson"
	"google.golang.org/protobuf/types/known/timestamppb"

	"github.com/determined-ai/determined/proto/pkg/commonv1"
	"github.com/determined-ai/determined/proto/pkg/trialv1"

	"github.com/determined-ai/determined/proto/pkg/apiv1"

	"gotest.tools/assert"
)

func TestTrialProfilerMetricsAvailableSeries(t *testing.T) {
	api, curUser, ctx := setupAPITest(t, nil)

	trial, _ := createTestTrial(t, api, curUser)

	ctx, cancel := context.WithTimeout(ctx, time.Second*10)
	defer cancel()

	reportTime := timestamppb.Now()

	gpuMetricNames := []string{"gpu_util", "gpu_free_memory"}
	gpuUUIDs := []string{"GPU-UUID-1", "GPU-UUID-2"}
	cpuMetricNames := []string{"cpu_util"}
	agentIDs := []string{"agent-A", "agent-B"}

	testCases := []struct {
		group      string
		names      []string
		agentID    string
		gpuUUIDs   []string
		reportTime *timestamppb.Timestamp
	}{
		{"gpu", gpuMetricNames, agentIDs[0], gpuUUIDs, reportTime},
		{"gpu", gpuMetricNames, agentIDs[1], gpuUUIDs, reportTime},
		{"cpu", cpuMetricNames, agentIDs[0], []string{}, reportTime},
		{"cpu", cpuMetricNames, agentIDs[1], []string{}, reportTime},
	}

	var testMetrics []*trialv1.TrialMetrics

	for _, tc := range testCases {
		testMetric := makeTestSystemMetrics(trial.ID, tc.names, tc.agentID, tc.gpuUUIDs, tc.reportTime)
		testMetrics = append(testMetrics, testMetric)
	}

	for i, tb := range testMetrics {
		_, err := api.ReportTrialMetrics(ctx, &apiv1.ReportTrialMetricsRequest{
			Metrics: tb,
			Group:   testCases[i].group,
		})
		assert.NilError(t, err, "failed to insert mocked trial profiler metrics")
	}

	res := &mockStream[*apiv1.GetTrialProfilerAvailableSeriesResponse]{ctx: ctx}
	TrialAvailableSeriesBatchWaitTime = 10 * time.Millisecond
	err := api.GetTrialProfilerAvailableSeries(&apiv1.GetTrialProfilerAvailableSeriesRequest{
		TrialId: int32(trial.ID),
	}, res)

	assert.NilError(t, err, "failed to initiate trial profiler series stream")

	resp := res.getData()
	respLabels := resp[0].Labels

	// Expect # GPUs * # GPU metric names * # agents + # CPU metric names * # agents
	expLabelCount := (len(gpuUUIDs) * len(gpuMetricNames) * len(agentIDs)) + (len(cpuMetricNames) * len(agentIDs))
	assert.Equal(t, expLabelCount, len(respLabels))
}

func makeTestSystemMetrics(
	trialID int,
	names []string,
	agentID string,
	gpuUUIDs []string,
	timestamp *timestamppb.Timestamp,
) *trialv1.TrialMetrics {
	sysMetrics := map[string]interface{}{}
	sysMetrics[agentID] = map[string]interface{}{}

	if len(gpuUUIDs) == 0 {
		for _, name := range names {
			sysMetrics[agentID].(map[string]interface{})[name] = rand.Float32() //nolint: gosec
		}
	}

	for _, gpuUUID := range gpuUUIDs {
		sysMetrics[agentID].(map[string]interface{})[gpuUUID] = map[string]float32{}
		for _, name := range names {
			sysMetrics[agentID].(map[string]interface{})[gpuUUID].(map[string]float32)[name] = rand.Float32() //nolint: gosec
		}
	}

	mJSON, _ := json.Marshal(sysMetrics)
	metrics := structpb.Struct{}
	_ = protojson.Unmarshal(mJSON, &metrics)
	return &trialv1.TrialMetrics{
		TrialId:    int32(trialID),
		TrialRunId: int32(0),
		ReportTime: timestamp,
		Metrics: &commonv1.Metrics{
			AvgMetrics: &metrics,
		},
	}
}
