//go:build integration
// +build integration

package api

import (
	"bytes"
	"context"
	"encoding/json"
	"io"
	"math/rand"
	"sort"
	"testing"
	"time"

	"github.com/golang/protobuf/ptypes/timestamp"

	structpb "github.com/golang/protobuf/ptypes/struct"

	"google.golang.org/protobuf/encoding/protojson"

	"google.golang.org/protobuf/types/known/timestamppb"

	"github.com/determined-ai/determined/proto/pkg/commonv1"
	"github.com/determined-ai/determined/proto/pkg/trialv1"

	"github.com/determined-ai/determined/master/internal"
	"github.com/determined-ai/determined/master/internal/db"
	"github.com/determined-ai/determined/master/pkg/schemas/expconf"

	"github.com/determined-ai/determined/master/test/testutils"

	"gotest.tools/assert"

	"github.com/determined-ai/determined/master/pkg/model"
	"github.com/determined-ai/determined/proto/pkg/apiv1"
)

func TestTrialDetail(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	_, _, cl, creds, err := testutils.RunMaster(ctx, nil)
	defer cancel()
	assert.NilError(t, err, "failed to start master")

	trialDetailAPITests(creds, t, cl, pgDB)
}

func TestTrialProfilerMetrics(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	_, _, cl, creds, err := testutils.RunMaster(ctx, nil)
	defer cancel()
	assert.NilError(t, err, "failed to start master")

	trialProfilerMetricsTests(creds, t, cl, pgDB)
}

func TestTrialProfilerMetricsAvailableSeries(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	_, _, cl, creds, err := testutils.RunMaster(ctx, nil)
	defer cancel()
	assert.NilError(t, err, "failed to start master")

	trialProfilerMetricsAvailableSeriesTests(creds, t, cl, pgDB)
}

func trialDetailAPITests(
	creds context.Context, t *testing.T, cl apiv1.DeterminedClient, pgDB *db.PgDB,
) {
	type testCase struct {
		name    string
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
			_, activeConfig, trial := setupTrial(t, pgDB)

			step := int32(id * activeConfig.SchedulingUnit())
			metrics := trialv1.TrialMetrics{
				TrialId:        int32(trial.ID),
				StepsCompleted: &step,
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

			ctx, cancel := context.WithTimeout(creds, 10*time.Second)
			defer cancel()
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

	trialWorkloadsAPIHugeMetrics(creds, t, cl, pgDB)
}

func makeMetrics() *structpb.Struct {
	return &structpb.Struct{
		Fields: map[string]*structpb.Value{
			"loss1": {
				Kind: &structpb.Value_NumberValue{
					NumberValue: rand.Float64(), //nolint: gosec
				},
			},
			"loss2": {
				Kind: &structpb.Value_NumberValue{
					NumberValue: rand.Float64(), //nolint: gosec
				},
			},
		},
	}
}

func trialWorkloadsAPIHugeMetrics(
	creds context.Context, t *testing.T, cl apiv1.DeterminedClient, pgDB *db.PgDB,
) {
	_, _, trial := setupTrial(t, pgDB)

	batchMetrics := []*structpb.Struct{}
	const stepSize = 1
	for j := 0; j < stepSize; j++ {
		batchMetrics = append(batchMetrics, makeMetrics())
	}

	step := int32(1)
	metrics := trialv1.TrialMetrics{
		TrialId:        int32(trial.ID),
		StepsCompleted: &step,
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
	ctx, cancel := context.WithTimeout(creds, 30*time.Second)
	defer cancel()
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
	creds context.Context, t *testing.T, cl apiv1.DeterminedClient, pgDB *db.PgDB,
) {
	_, _, trial := setupTrial(t, pgDB)

	// If we begin to stream for metrics.
	ctx, cancel := context.WithTimeout(creds, time.Minute)
	defer cancel()
	labels := trialv1.TrialProfilerMetricLabels{
		TrialId:    int32(trial.ID),
		GpuUuid:    "1",
		Name:       "gpu_util",
		AgentId:    "brads agent",
		MetricType: trialv1.TrialProfilerMetricLabels_PROFILER_METRIC_TYPE_SYSTEM,
	}
	tlCl, err := cl.GetTrialProfilerMetrics(ctx, &apiv1.GetTrialProfilerMetricsRequest{
		Labels: &labels,
		Follow: true,
	})
	assert.NilError(t, err, "failed to initiate trial profiler metrics stream")
	reportTime := timestamppb.Now()

	for i := 0; i < 10; i++ {
		// When we add some metrics that match our stream.
		match := randTrialProfilerSystemMetrics(trial.ID, "gpu_util", "brads agent", "1", reportTime)
		_, err = cl.ReportTrialMetrics(creds, &apiv1.ReportTrialMetricsRequest{
			Metrics: match,
			Group:   "gpu",
		})
		assert.NilError(t, err, "failed to insert mocked trial profiler metrics")

		// And some that do not match our stream.
		notMatch := randTrialProfilerSystemMetrics(trial.ID, "gpu_util", "someone elses agent", "1", reportTime)
		_, err = cl.ReportTrialMetrics(creds, &apiv1.ReportTrialMetricsRequest{
			Metrics: notMatch,
			Group:   "gpu",
		})
		assert.NilError(t, err, "failed to insert mocked unmatched trial profiler metrics")

		// Then when we receive the metrics, they should be the metrics we expect.
		recvMetricsBatch, metricErr := tlCl.Recv()
		assert.NilError(t, metricErr, "failed to stream metrics")

		// Just nil the values since the floats lose a little precision getting thrown
		// around so much.
		expectedBatch := &trialv1.TrialProfilerMetricsBatch{
			Values:     nil,
			Timestamps: []*timestamp.Timestamp{reportTime},
			Labels:     &labels,
		}
		recvMetricsBatch.Batch.Values = nil

		bOrig, b0Err := protojson.Marshal(expectedBatch)
		assert.NilError(t, b0Err, "failed marshal original metrics")

		bRecv, bRecvErr := protojson.Marshal(recvMetricsBatch.Batch)
		assert.NilError(t, bRecvErr, "failed marshal received metrics")

		origEqRecv := bytes.Equal(bOrig, bRecv)
		assert.Assert(t, origEqRecv, "received:\nt\t%s\noriginal:\n\t%s", bRecv, bOrig)
	}

	err = db.UpdateTrial(ctx, trial.ID, model.StoppingCompletedState)
	assert.NilError(t, err, "failed to update trial state")
	err = db.UpdateTrial(ctx, trial.ID, model.CompletedState)
	assert.NilError(t, err, "failed to update trial state")

	_, err = tlCl.Recv()
	assert.Equal(t, err, io.EOF, "log stream didn't terminate with trial")
}

func trialProfilerMetricsAvailableSeriesTests(
	creds context.Context, t *testing.T, cl apiv1.DeterminedClient, pgDB *db.PgDB,
) {
	_, _, trial := setupTrial(t, pgDB)

	ctx, cancel := context.WithTimeout(creds, time.Minute)
	defer cancel()
	tlCl, err := cl.GetTrialProfilerAvailableSeries(ctx, &apiv1.GetTrialProfilerAvailableSeriesRequest{
		TrialId: int32(trial.ID),
		Follow:  true,
	})
	assert.NilError(t, err, "failed to initiate trial profiler series stream")
	reportTime := timestamppb.Now()
	metricNames := []string{
		"gpu_util", "gpu_util", "cpu_util", "cpu_util",
	}
	metricGroups := []string{
		"gpu", "gpu", "cpu", "cpu",
	}
	agentIDs := []string{
		"agent0", "agent1", "agent0", "agent2",
	}
	var testMetrics []*trialv1.TrialMetrics

	for i, n := range metricNames {
		testMetric := randTrialProfilerSystemMetrics(trial.ID, n, agentIDs[i], "1", reportTime)
		testMetrics = append(testMetrics, testMetric)
	}

	var expected []string
	internal.TrialAvailableSeriesBatchWaitTime = 10 * time.Millisecond

	for i, tb := range testMetrics {
		expected = append(expected, metricNames[i])
		_, err = cl.ReportTrialMetrics(creds, &apiv1.ReportTrialMetricsRequest{
			Metrics: tb,
			Group:   metricGroups[i],
		})
		assert.NilError(t, err, "failed to insert mocked trial profiler metrics")

		// This may need 2 or more attempts; gRPC streaming does not provide any backpressure
		// mechanism, if the client is not ready to receive a message when it is sent to the
		// stream server side, the server will buffer it and any calls to Send will return,
		// allowing the code to chug along merrily and Send more and more data while the
		// client doesn't consume it. Because of this, we give the test a chance to read
		// out stale entries before marking the attempt as a failure. For more detail, see
		// https://github.com/grpc/grpc-go/issues/2159.
		shots := 5
		var resp *apiv1.GetTrialProfilerAvailableSeriesResponse
		for i := 1; i <= shots; i++ {
			resp, err = tlCl.Recv()
			assert.NilError(t, err, "failed to stream metrics")

			if len(expected) == len(resp.Labels) {
				break
			}

			if i == shots {
				assert.Equal(
					t, len(expected),
					len(resp.Labels),
					"incorrect number of labels; expected %v, got %v",
					expected,
					resp.Labels,
				)
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

func randTrialProfilerSystemMetrics(
	trialID int, name, agentID, gpuUUID string, timestamp *timestamppb.Timestamp,
) *trialv1.TrialMetrics {
	metrics := structpb.Struct{}
	j, _ := json.Marshal(map[string]interface{}{
		agentID: map[string]interface{}{
			gpuUUID: map[string]float32{
				name: rand.Float32(), //nolint: gosec
			},
		},
	})
	_ = protojson.Unmarshal(j, &metrics)
	return &trialv1.TrialMetrics{
		TrialId:    int32(trialID),
		TrialRunId: int32(0),
		ReportTime: timestamp,
		Metrics: &commonv1.Metrics{
			AvgMetrics: &metrics,
		},
	}
}

func setupTrial(t *testing.T, pgDB *db.PgDB) (
	*model.Experiment, expconf.ExperimentConfig, *model.Trial,
) {
	experiment, activeConfig := model.ExperimentModel()
	err := pgDB.AddExperiment(experiment, []byte{}, activeConfig)
	assert.NilError(t, err, "failed to insert experiment")

	task := db.RequireMockTask(t, pgDB, experiment.OwnerID)
	trial := &model.Trial{
		ExperimentID: experiment.ID,
		State:        model.ActiveState,
		StartTime:    time.Now(),
	}

	err = db.AddTrial(context.TODO(), trial, task.TaskID)
	assert.NilError(t, err, "failed to insert trial")
	return experiment, activeConfig, trial
}
