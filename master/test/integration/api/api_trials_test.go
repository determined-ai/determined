//go:build integration
// +build integration

package api

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"math"
	"math/rand"
	"sort"
	"strings"
	"testing"
	"time"

	structpb "github.com/golang/protobuf/ptypes/struct"

	"google.golang.org/protobuf/encoding/protojson"

	"google.golang.org/protobuf/types/known/timestamppb"

	"github.com/determined-ai/determined/proto/pkg/commonv1"
	"github.com/determined-ai/determined/proto/pkg/trialv1"

	"github.com/determined-ai/determined/master/internal"
	"github.com/determined-ai/determined/master/internal/db"

	"github.com/determined-ai/determined/master/test/testutils"

	"github.com/golang/protobuf/ptypes"
	"gotest.tools/assert"

	"github.com/determined-ai/determined/master/pkg/model"
	"github.com/determined-ai/determined/proto/pkg/apiv1"
)

func TestTrialDetail(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	_, _, cl, creds, err := testutils.RunMaster(ctx, nil)
	defer cancel()
	assert.NilError(t, err, "failed to start master")

	trialDetailAPITests(t, creds, cl, pgDB)
}

func TestTrialProfilerMetrics(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	_, _, cl, creds, err := testutils.RunMaster(ctx, nil)
	defer cancel()
	assert.NilError(t, err, "failed to start master")

	trialProfilerMetricsTests(t, creds, cl, pgDB)
}

func TestTrialProfilerMetricsAvailableSeries(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	_, _, cl, creds, err := testutils.RunMaster(ctx, nil)
	defer cancel()
	assert.NilError(t, err, "failed to start master")

	trialProfilerMetricsAvailableSeriesTests(t, creds, cl, pgDB)
}

func trialDetailAPITests(
	t *testing.T, creds context.Context, cl apiv1.DeterminedClient, pgDB *db.PgDB,
) {
	type testCase struct {
		name    string
		req     *apiv1.GetTrialRequest
		metrics map[string]interface{}
	}

	testCases := []testCase{
		{
			name: "scalar metric",
			metrics: map[string]interface{}{
				"myMetricName": 3,
			},
		},
		{
			name: "boolean metric",
			metrics: map[string]interface{}{
				"myMetricName": true,
			},
		},
		{
			name: "null metric",
			metrics: map[string]interface{}{
				"myMetricName": nil,
			},
		},
		{
			name: "list of scalars metric",
			metrics: map[string]interface{}{
				"myMetricName": []float32{1, 2.3, 3},
			},
		},
	}

	runTestCase := func(t *testing.T, tc testCase, id int) {
		t.Run(tc.name, func(t *testing.T) {
			experiment, trial := setupTrial(t, pgDB)

			metrics := trialv1.TrialMetrics{
				TrialId:        int32(trial.ID),
				StepsCompleted: int32(id * experiment.Config.SchedulingUnit()),
			}

			m := structpb.Struct{}
			b, err := json.Marshal(map[string]interface{}{"metrics": tc.metrics})
			assert.NilError(t, err, "failed to marshal metrics")
			err = protojson.Unmarshal(b, &m)
			assert.NilError(t, err, "failed to unmarshal metrics")
			metrics.Metrics = &commonv1.Metrics{
				AvgMetrics: &m,
			}

			err = pgDB.AddTrainingMetrics(context.Background(), &metrics)
			assert.NilError(t, err, "failed to insert step")

			ctx, _ := context.WithTimeout(creds, 10*time.Second)
			req := apiv1.GetTrialRequest{TrialId: int32(trial.ID)}

			_, err = cl.GetTrial(ctx, &req)
			assert.NilError(t, err, "failed to fetch api details")
		})
	}

	for idx, tc := range testCases {
		runTestCase(t, tc, idx)
	}
}

func TestTrialWorkloadsHugeMetrics(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	_, _, cl, creds, err := testutils.RunMaster(ctx, nil)
	defer cancel()
	assert.NilError(t, err, "failed to start master")

	trialWorkloadsAPIHugeMetrics(t, creds, cl, pgDB)
}

func makeMetrics() *structpb.Struct {
	return &structpb.Struct{
		Fields: map[string]*structpb.Value{
			"loss1": {
				Kind: &structpb.Value_NumberValue{
					NumberValue: rand.Float64(),
				},
			},
			"loss2": {
				Kind: &structpb.Value_NumberValue{
					NumberValue: rand.Float64(),
				},
			},
		},
	}
}

func trialWorkloadsAPIHugeMetrics(
	t *testing.T, creds context.Context, cl apiv1.DeterminedClient, pgDB *db.PgDB,
) {
	_, trial := setupTrial(t, pgDB)

	batchMetrics := []*structpb.Struct{}
	const stepSize = 1
	for j := 0; j < stepSize; j++ {
		batchMetrics = append(batchMetrics, makeMetrics())
	}

	metrics := trialv1.TrialMetrics{
		TrialId:        int32(trial.ID),
		StepsCompleted: stepSize,
		Metrics: &commonv1.Metrics{
			AvgMetrics:   makeMetrics(),
			BatchMetrics: batchMetrics,
		},
	}

	err := pgDB.AddTrainingMetrics(context.Background(), &metrics)
	assert.NilError(t, err, "failed to insert step")

	_, err = pgDB.RawQuery("test_insert_huge_metrics", trial.ID, 100000)
	assert.NilError(t, err, "failed to insert huge amount of metrics")

	req := apiv1.GetTrialWorkloadsRequest{
		TrialId:             int32(trial.ID),
		IncludeBatchMetrics: true,
		Limit:               1,
	}
	ctx, _ := context.WithTimeout(creds, 30*time.Second)
	resp, err := cl.GetTrialWorkloads(ctx, &req)
	assert.NilError(t, err, "failed to fetch trial workloads")

	for _, workloadContainer := range resp.Workloads {
		training := workloadContainer.GetTraining()
		if training == nil {
			continue
		}
		assert.Equal(t, len(training.Metrics.BatchMetrics) > 0, true)
	}
}

func trialProfilerMetricsTests(
	t *testing.T, creds context.Context, cl apiv1.DeterminedClient, pgDB *db.PgDB,
) {
	_, trial := setupTrial(t, pgDB)

	// If we begin to stream for metrics.
	ctx, _ := context.WithTimeout(creds, time.Minute)
	tlCl, err := cl.GetTrialProfilerMetrics(ctx, &apiv1.GetTrialProfilerMetricsRequest{
		Labels: &trialv1.TrialProfilerMetricLabels{
			TrialId:    int32(trial.ID),
			Name:       "gpu_util",
			AgentId:    "brad's agent",
			MetricType: trialv1.TrialProfilerMetricLabels_PROFILER_METRIC_TYPE_SYSTEM,
		},
		Follow: true,
	})
	assert.NilError(t, err, "failed to initiate trial profiler metrics stream")

	for i := 0; i < 10; i++ {
		// When we add some metrics that match our stream.
		match := randTrialProfilerSystemMetrics(trial.ID, "gpu_util", "brad's agent", "1")
		_, err := cl.PostTrialProfilerMetricsBatch(creds, &apiv1.PostTrialProfilerMetricsBatchRequest{
			Batches: []*trialv1.TrialProfilerMetricsBatch{
				match,
			},
		})
		assert.NilError(t, err, "failed to insert mocked trial profiler metrics")

		// And some that do not match our stream.
		notMatch := randTrialProfilerSystemMetrics(trial.ID, "gpu_util", "someone else's agent", "1")
		_, err = cl.PostTrialProfilerMetricsBatch(creds, &apiv1.PostTrialProfilerMetricsBatchRequest{
			Batches: []*trialv1.TrialProfilerMetricsBatch{
				notMatch,
			},
		})
		assert.NilError(t, err, "failed to insert mocked unmatched trial profiler metrics")

		// Then when we receive the metrics, they should be the metrics we expect.
		recvMetricsBatch, err := tlCl.Recv()
		assert.NilError(t, err, "failed to stream metrics")

		// Just nil the values since the floats and timestamps lose a little precision getting thrown
		// around so much.
		match.Values = nil
		match.Timestamps = nil
		recvMetricsBatch.Batch.Values = nil
		recvMetricsBatch.Batch.Timestamps = nil

		bOrig, err := protojson.Marshal(match)
		assert.NilError(t, err, "failed marshal original metrics")

		bRecv, err := protojson.Marshal(recvMetricsBatch.Batch)
		assert.NilError(t, err, "failed marshal received metrics")

		origEqRecv := bytes.Equal(bOrig, bRecv)
		assert.Assert(t, origEqRecv, "received:\nt\t%s\noriginal:\n\t%s", bRecv, bOrig)
	}

	err = pgDB.UpdateTrial(trial.ID, model.StoppingCompletedState)
	assert.NilError(t, err, "failed to update trial state")
	err = pgDB.UpdateTrial(trial.ID, model.CompletedState)
	assert.NilError(t, err, "failed to update trial state")

	_, err = tlCl.Recv()
	assert.Equal(t, err, io.EOF, "log stream didn't terminate with trial")
}

func trialProfilerMetricsAvailableSeriesTests(
	t *testing.T, creds context.Context, cl apiv1.DeterminedClient, pgDB *db.PgDB,
) {
	_, trial := setupTrial(t, pgDB)

	ctx, _ := context.WithTimeout(creds, time.Minute)
	tlCl, err := cl.GetTrialProfilerAvailableSeries(ctx, &apiv1.GetTrialProfilerAvailableSeriesRequest{
		TrialId: int32(trial.ID),
		Follow:  true,
	})
	assert.NilError(t, err, "failed to initiate trial profiler series stream")

	testBatches := []*trialv1.TrialProfilerMetricsBatch{
		randTrialProfilerSystemMetrics(trial.ID, "gpu_util", "agent0", "1"),
		randTrialProfilerSystemMetrics(trial.ID, "gpu_util", "agent1", "1"),
		randTrialProfilerSystemMetrics(trial.ID, "cpu_util", "agent0", "1"),
		randTrialProfilerSystemMetrics(trial.ID, "cpu_util", "agent2", "1"),
		randTrialProfilerSystemMetrics(trial.ID, "other_metric", "", ""),
	}

	var expected []string
	internal.TrialAvailableSeriesBatchWaitTime = 10 * time.Millisecond

	for _, tb := range testBatches {
		expected = append(expected, tb.Labels.Name)
		_, err := cl.PostTrialProfilerMetricsBatch(creds, &apiv1.PostTrialProfilerMetricsBatchRequest{
			Batches: []*trialv1.TrialProfilerMetricsBatch{
				tb,
			},
		})
		assert.NilError(t, err, "failed to insert mocked trial profiler metrics")

		// This may need 2 or more attempts; gRPC streaming does not provide any backpressure mechanism, if the client
		// is not ready to receive a message when it is sent to the stream server side, the server will buffer it and
		// any calls to Send will return, allowing the code to chug along merrily and Send more and more data while the
		// client doesn't consume it. Because of this, we give the test a chance to read out stale entries before
		// marking the attempt as a failure. For more detail, see https://github.com/grpc/grpc-go/issues/2159.
		shots := 5
		var resp *apiv1.GetTrialProfilerAvailableSeriesResponse
		for i := 1; i <= shots; i++ {
			resp, err = tlCl.Recv()
			assert.NilError(t, err, "failed to stream metrics")

			if len(expected) == len(resp.Labels) {
				break
			}

			if i == shots {
				assert.Equal(t, len(expected), len(resp.Labels), "incorrect number of labels")
			}
		}

		var actual []string
		for _, l := range resp.Labels {
			actual = append(actual, l.Name)
		}

		sort.Strings(expected)
		sort.Strings(actual)

		assert.DeepEqual(t, expected, actual)
	}
}

func assertStringContains(t *testing.T, actual, expected string) {
	assert.Assert(t, strings.Contains(actual, expected),
		fmt.Sprintf("%s not in %s", expected, actual))
}

// stringWithPrefix is a convenience function that returns a pointer.
func stringWithPrefix(p, s string) *string {
	x := p + s
	return &x
}

// timePlusDuration is a convenience function that returns a pointer.
func timePlusDuration(t time.Time, d time.Duration) *time.Time {
	t2 := t.Add(d)
	return &t2
}

func sequentialIntSlice(start, n int) []int32 {
	fs := make([]int32, n)
	for i := 0; i < n; i++ {
		fs[i] = int32(start + i)
	}
	return fs
}

func randFloatSlice(n int) []float32 {
	fs := make([]float32, n)
	for i := 0; i < n; i++ {
		fs[i] = rand.Float32()
	}
	return fs
}

func pbTimestampSlice(n int) []*timestamppb.Timestamp {
	ts := make([]*timestamppb.Timestamp, n)
	for i := 0; i < n; i++ {
		ts[i] = ptypes.TimestampNow()
		// Round off to millis.
		ts[i].Nanos = int32(math.Round(float64(ts[i].Nanos)/float64(time.Millisecond)) * float64(time.Millisecond))
	}
	return ts
}

func randTrialProfilerSystemMetrics(
	trialID int, name, agentID, gpuUUID string,
) *trialv1.TrialProfilerMetricsBatch {
	return &trialv1.TrialProfilerMetricsBatch{
		Values:     randFloatSlice(5),
		Batches:    sequentialIntSlice(0, 5),
		Timestamps: pbTimestampSlice(5),
		Labels: &trialv1.TrialProfilerMetricLabels{
			TrialId:    int32(trialID),
			Name:       name,
			AgentId:    agentID,
			GpuUuid:    gpuUUID,
			MetricType: trialv1.TrialProfilerMetricLabels_PROFILER_METRIC_TYPE_SYSTEM,
		},
	}
}

func setupTrial(t *testing.T, pgDB *db.PgDB) (*model.Experiment, *model.Trial) {
	experiment := model.ExperimentModel()
	err := pgDB.AddExperiment(experiment)
	assert.NilError(t, err, "failed to insert experiment")

	task := db.RequireMockTask(t, pgDB, experiment.OwnerID)
	trial := &model.Trial{
		TaskID:       task.TaskID,
		JobID:        experiment.JobID,
		ExperimentID: experiment.ID,
		State:        model.ActiveState,
		StartTime:    time.Now(),
	}

	err = pgDB.AddTrial(trial)
	assert.NilError(t, err, "failed to insert trial")
	return experiment, trial
}
