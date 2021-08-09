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
	"strconv"
	"strings"
	"testing"
	"time"

	structpb "github.com/golang/protobuf/ptypes/struct"

	"google.golang.org/protobuf/encoding/protojson"

	"google.golang.org/protobuf/types/known/timestamppb"

	"github.com/determined-ai/determined/proto/pkg/trialv1"

	"github.com/determined-ai/determined/master/internal"
	"github.com/determined-ai/determined/master/internal/db"
	"github.com/determined-ai/determined/master/internal/elastic"

	"github.com/determined-ai/determined/master/test/testutils"

	"github.com/golang/protobuf/ptypes"
	"gotest.tools/assert"

	"github.com/determined-ai/determined/master/pkg/model"
	"github.com/determined-ai/determined/proto/pkg/apiv1"
)

func TestTrialLogAPIPostgres(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	_, _, cl, creds, err := testutils.RunMaster(ctx, nil)
	defer cancel()
	assert.NilError(t, err, "failed to start master")

	trialLogAPITests(t, creds, cl, pgDB, func() error {
		return nil
	})
}

func TestTrialLogAPIElastic(t *testing.T) {
	cfg, err := testutils.DefaultMasterConfig()
	assert.NilError(t, err, "failed to create master config")
	cfg.Logging = testutils.DefaultElasticConfig()

	ctx, cancel := context.WithCancel(context.Background())
	_, _, cl, creds, err := testutils.RunMaster(ctx, cfg)
	defer cancel()
	assert.NilError(t, err, "failed to start master")

	trialLogAPITests(t, creds, cl, es, func() error {
		return es.WaitForIngest(elastic.CurrentLogstashIndex())
	})
}

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
	t *testing.T, creds context.Context, cl apiv1.DeterminedClient, db *db.PgDB,
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
			experiment := testutils.ExperimentModel()
			err := db.AddExperiment(experiment)
			assert.NilError(t, err, "failed to insert experiment")

			trial := testutils.TrialModel(experiment.ID, testutils.WithTrialState(model.ActiveState))
			err = db.AddTrial(trial)
			assert.NilError(t, err, "failed to insert trial")

			metrics := trialv1.TrainingMetrics{
				TrialId:      int32(trial.ID),
				TotalBatches: int32(id * experiment.Config.SchedulingUnit()),
			}

			m := structpb.Struct{}
			b, err := json.Marshal(map[string]interface{}{"metrics": tc.metrics})
			assert.NilError(t, err, "failed to marshal metrics")
			err = protojson.Unmarshal(b, &m)
			assert.NilError(t, err, "failed to unmarshal metrics")
			metrics.Metrics = &m

			err = db.AddTrainingMetrics(context.Background(), &metrics)
			assert.NilError(t, err, "failed to insert step")

			ctx, _ := context.WithTimeout(creds, 10*time.Second)
			req := apiv1.GetTrialRequest{TrialId: int32(trial.ID)}

			tlCl, err := cl.GetTrial(ctx, &req)
			assert.NilError(t, err, "failed to fetch api details")
			assert.Equal(t, len(tlCl.Workloads), 1, "mismatching workload length")
		})
	}

	for idx, tc := range testCases {
		runTestCase(t, tc, idx)
	}

}

func trialProfilerMetricsTests(
	t *testing.T, creds context.Context, cl apiv1.DeterminedClient, db *db.PgDB,
) {
	// Given an experiment.
	experiment := testutils.ExperimentModel()
	err := db.AddExperiment(experiment)
	assert.NilError(t, err, "failed to insert experiment")

	// With a trial.
	trial := testutils.TrialModel(experiment.ID, testutils.WithTrialState(model.ActiveState))
	err = db.AddTrial(trial)
	assert.NilError(t, err, "failed to insert trial")

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
	t *testing.T, creds context.Context, cl apiv1.DeterminedClient, db *db.PgDB,
) {
	experiment := testutils.ExperimentModel()
	err := db.AddExperiment(experiment)
	assert.NilError(t, err, "failed to insert experiment")

	trial := testutils.TrialModel(experiment.ID, testutils.WithTrialState(model.ActiveState))
	err = db.AddTrial(trial)
	assert.NilError(t, err, "failed to insert trial")

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

func trialLogAPITests(
	t *testing.T, creds context.Context, cl apiv1.DeterminedClient, backend internal.TrialLogBackend,
	awaitBackend func() error,
) {
	type testCase struct {
		name    string
		req     *apiv1.TrialLogsRequest
		logs    []*model.TrialLog
		matches []string
	}

	agent0, agent1 := "elated-backward-cat", "sad-testfailed-cat"
	rank0, rank1 := 0, 1
	time0 := time.Now().UTC().Add(-time.Minute)
	pTime0, err := ptypes.TimestampProto(time0)
	assert.NilError(t, err, "failed to make proto time")
	tests := []testCase{
		{
			name: "agent_id in list",
			req: &apiv1.TrialLogsRequest{
				Limit:  2,
				Follow: false,
				AgentIds: []string{
					agent0,
				},
			},
			logs: []*model.TrialLog{
				{
					AgentID:   &agent0,
					Log:       stringWithPrefix("a log from ", agent0),
					Timestamp: &time0,
				},
				{
					AgentID:   &agent1,
					Log:       stringWithPrefix("a log from ", agent1),
					Timestamp: timePlusDuration(time0, time.Second),
				},
				{
					AgentID:   &agent0,
					Log:       stringWithPrefix("another log from ", agent0),
					Timestamp: timePlusDuration(time0, 2*time.Second),
				},
				{
					AgentID:   &agent1,
					Log:       stringWithPrefix("another log from ", agent1),
					Timestamp: timePlusDuration(time0, 3*time.Second),
				},
			},
			matches: []string{
				"a log from " + agent0,
				"another log from " + agent0,
			},
		},
		{
			name: "rank_id in list",
			req: &apiv1.TrialLogsRequest{
				Limit:  2,
				Follow: false,
				RankIds: []int32{
					int32(rank0),
				},
			},
			logs: []*model.TrialLog{
				{
					RankID:    &rank0,
					Log:       stringWithPrefix("a log from ", strconv.Itoa(rank0)),
					Timestamp: &time0,
				},
				{
					RankID:    &rank1,
					Log:       stringWithPrefix("a log from ", strconv.Itoa(rank1)),
					Timestamp: timePlusDuration(time0, time.Second),
				},
				{
					RankID:    &rank0,
					Log:       stringWithPrefix("another log from ", strconv.Itoa(rank0)),
					Timestamp: timePlusDuration(time0, 2*time.Second),
				},
				{
					RankID:    &rank1,
					Log:       stringWithPrefix("another log from ", strconv.Itoa(rank1)),
					Timestamp: timePlusDuration(time0, 3*time.Second),
				},
			},
			matches: []string{
				"a log from " + strconv.Itoa(rank0),
				"another log from " + strconv.Itoa(rank0),
			},
		},
		{
			name: "timestamp_before",
			req: &apiv1.TrialLogsRequest{
				Limit:           1,
				Follow:          false,
				TimestampBefore: pTime0,
			},
			logs: []*model.TrialLog{
				{
					AgentID:   &agent1,
					Timestamp: timePlusDuration(time0, time.Second),
					Log:       stringWithPrefix("", "a log at time0 and a second"),
				},
				{
					AgentID:   &agent0,
					Timestamp: timePlusDuration(time0, -time.Second),
					Log:       stringWithPrefix("", "a log at time0 less a second"),
				},
			},
			matches: []string{"a log at time0 less a second"},
		},
		{
			name: "timestamp_after",
			req: &apiv1.TrialLogsRequest{
				Limit:          1,
				Follow:         false,
				TimestampAfter: pTime0,
			},
			logs: []*model.TrialLog{
				{
					AgentID:   &agent0,
					Timestamp: &time0,
					Log:       stringWithPrefix("", "a log at time0"),
				},
				{
					AgentID:   &agent1,
					Timestamp: timePlusDuration(time0, time.Second),
					Log:       stringWithPrefix("", "a log at time0 and a second"),
				},
				{
					AgentID:   &agent0,
					Timestamp: timePlusDuration(time0, -time.Second),
					Log:       stringWithPrefix("", "a log at time0 less a second"),
				},
			},
			matches: []string{"a log at time0 and a second"},
		},
		{
			name: "order by desc",
			req: &apiv1.TrialLogsRequest{
				Limit:   3,
				Follow:  false,
				OrderBy: apiv1.OrderBy_ORDER_BY_DESC,
			},
			logs: []*model.TrialLog{
				{
					AgentID:   &agent1,
					Timestamp: timePlusDuration(time0, time.Second),
					Log:       stringWithPrefix("", "a log at time0 and a second"),
				},
				{
					AgentID:   &agent0,
					Timestamp: &time0,
					Log:       stringWithPrefix("", "a log at time0"),
				},
				{
					AgentID:   &agent0,
					Timestamp: timePlusDuration(time0, -time.Second),
					Log:       stringWithPrefix("", "a log at time0 less a second"),
				},
			},
			matches: []string{
				"a log at time0 and a second",
				"a log at time0",
				"a log at time0 less a second",
			},
		},
	}

	runTestCase := func(t *testing.T, tc testCase) {
		t.Run(tc.name, func(t *testing.T) {
			experiment := testutils.ExperimentModel()
			err := pgDB.AddExperiment(experiment)
			assert.NilError(t, err, "failed to insert experiment")

			trial := testutils.TrialModel(experiment.ID, testutils.WithTrialState(model.ActiveState))
			err = pgDB.AddTrial(trial)
			assert.NilError(t, err, "failed to insert trial")

			for i := range tc.logs {
				tc.logs[i].TrialID = trial.ID
			}
			err = backend.AddTrialLogs(tc.logs)
			assert.NilError(t, err, "failed to insert mocked trial logs")

			assert.NilError(t, awaitBackend(), "failed to wait for logging backend")

			tc.req.TrialId = int32(trial.ID)
			ctx, _ := context.WithTimeout(creds, time.Minute)
			tlCl, err := cl.TrialLogs(ctx, tc.req)
			i := 0
			ids := map[string]bool{}
			for {
				resp, err := tlCl.Recv()
				if err == io.EOF {
					assert.Equal(t, i, len(tc.matches))
					return
				}
				// context.DeadlineExceeded likely means the stream did not terminate as expected.
				assert.NilError(t, err, "failed to receive trial logs")
				assert.Assert(t, i < len(tc.matches), "received too many logs")
				assertStringContains(t, resp.Message, tc.matches[i])

				// assert IDs are unique
				assert.Equal(t, ids[resp.Id], false, "log ID was not unique: %s", resp.Id)
				ids[resp.Id] = true
				i++
			}
		})
	}

	for _, tc := range tests {
		runTestCase(t, tc)
	}
}

func TestTrialLogFollowingPostgres(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	_, _, cl, creds, err := testutils.RunMaster(ctx, nil)
	defer cancel()
	assert.NilError(t, err, "failed to start master")

	trialLogFollowingTests(t, ctx, creds, cl, pgDB)
}

func TestTrialLogFollowingElastic(t *testing.T) {
	cfg, err := testutils.DefaultMasterConfig()
	assert.NilError(t, err, "failed to create master config")
	cfg.Logging = testutils.DefaultElasticConfig()

	ctx, cancel := context.WithCancel(context.Background())
	_, _, cl, creds, err := testutils.RunMaster(ctx, cfg)
	defer cancel()
	assert.NilError(t, err, "failed to start master")

	trialLogFollowingTests(t, ctx, creds, cl, es)
}

func trialLogFollowingTests(
	t *testing.T, ctx context.Context, creds context.Context, cl apiv1.DeterminedClient,
	backend internal.TrialLogBackend,
) {
	experiment := testutils.ExperimentModel()
	err := pgDB.AddExperiment(experiment)
	assert.NilError(t, err, "failed to insert experiment")

	trial := testutils.TrialModel(experiment.ID, testutils.WithTrialState(model.ActiveState))
	err = pgDB.AddTrial(trial)
	assert.NilError(t, err, "failed to insert trial")

	ctx, _ = context.WithTimeout(creds, time.Minute)
	tlCl, err := cl.TrialLogs(ctx, &apiv1.TrialLogsRequest{
		TrialId: int32(trial.ID),
		Follow:  true,
	})

	// Write logs and turn around and make sure following receives them.
	time0 := time.Now().UTC().Add(-time.Minute)
	for trialLogID := 0; trialLogID < 5; trialLogID++ {
		message := fmt.Sprintf("log %d", trialLogID)
		err = backend.AddTrialLogs([]*model.TrialLog{
			{
				TrialID:   trial.ID,
				Log:       &message,
				Timestamp: timePlusDuration(time0, time.Duration(trialLogID)*time.Second),
			},
		})
		assert.NilError(t, err, "failed to insert mocked trial logs")

		resp, err := tlCl.Recv()
		assert.NilError(t, err, "failed to stream logs")

		// context.DeadlineExceeded likely means the stream did not terminate as expected.
		assert.NilError(t, err, "failed to receive trial logs")
		assertStringContains(t, resp.Message, message)
	}

	err = pgDB.UpdateTrial(trial.ID, model.StoppingCompletedState)
	assert.NilError(t, err, "failed to update trial state")
	err = pgDB.UpdateTrial(trial.ID, model.CompletedState)
	assert.NilError(t, err, "failed to update trial state")

	_, err = tlCl.Recv()
	assert.Equal(t, err, io.EOF, "log stream didn't terminate with trial")
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
