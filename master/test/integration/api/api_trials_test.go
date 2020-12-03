// +build integration

package api

import (
	"context"
	"fmt"
	"io"
	"strconv"
	"testing"
	"time"

	"github.com/determined-ai/determined/master/test/testutils"

	"github.com/golang/protobuf/ptypes"
	"gotest.tools/assert"

	"github.com/determined-ai/determined/master/pkg/model"
	"github.com/determined-ai/determined/proto/pkg/apiv1"
)

func TestTrialLogAPI(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	_, _, cl, creds, err := testutils.RunMaster(ctx, nil)
	defer cancel()
	assert.NilError(t, err, "failed to start master")

	type testCase struct {
		name    string
		req     *apiv1.TrialLogsRequest
		logs    []*model.TrialLog
		matches []string
	}

	agent0, agent1 := "elated-backward-cat", "sad-testfailed-cat"
	rank0, rank1 := 0, 1
	time0 := time.Now().UTC()
	time0Plus1, time0Minus1 := time0.Add(time.Second), time0.Add(-time.Second)
	pTime0, err := ptypes.TimestampProto(time0)
	assert.NilError(t, err, "failed to make proto time")
	tests := []testCase{
		{
			name: "agent_id in list",
			req: &apiv1.TrialLogsRequest{
				Offset: 0,
				Limit:  2,
				Follow: false,
				AgentIds: []string{
					agent0,
				},
			},
			logs: []*model.TrialLog{
				{
					AgentID: &agent0,
					Message: "a log from " + agent0,
				},
				{
					AgentID: &agent1,
					Message: "a log from " + agent1,
				},
				{
					AgentID: &agent0,
					Message: "another log from " + agent0,
				},
				{
					AgentID: &agent1,
					Message: "another log from " + agent1,
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
				Offset: 0,
				Limit:  2,
				Follow: false,
				RankIds: []int32{
					int32(rank0),
				},
			},
			logs: []*model.TrialLog{
				{
					RankID:  &rank0,
					Message: "a log from " + strconv.Itoa(rank0),
				},
				{
					RankID:  &rank1,
					Message: "a log from " + strconv.Itoa(rank1),
				},
				{
					RankID:  &rank0,
					Message: "another log from " + strconv.Itoa(rank0),
				},
				{
					RankID:  &rank1,
					Message: "another log from " + strconv.Itoa(rank1),
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
				Offset:          0,
				Limit:           1,
				Follow:          false,
				TimestampBefore: pTime0,
			},
			logs: []*model.TrialLog{
				{
					AgentID:   &agent1,
					Timestamp: &time0Plus1,
					Message:   "a log at time0 and a second",
				},
				{
					AgentID:   &agent0,
					Timestamp: &time0Minus1,
					Message:   "a log at time0 less a second",
				},
			},
			matches: []string{"a log at time0 less a second"},
		},
		{
			name: "timestamp_after",
			req: &apiv1.TrialLogsRequest{
				Offset:         0,
				Limit:          1,
				Follow:         false,
				TimestampAfter: pTime0,
			},
			logs: []*model.TrialLog{
				{
					AgentID:   &agent0,
					Timestamp: &time0,
					Message:   "a log at time0",
				},
				{
					AgentID:   &agent1,
					Timestamp: &time0Plus1,
					Message:   "a log at time0 and a second",
				},
				{
					AgentID:   &agent0,
					Timestamp: &time0Minus1,
					Message:   "a log at time0 less a second",
				},
			},
			matches: []string{"a log at time0 and a second"},
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
			err = pgDB.AddTrialLogs(tc.logs)
			assert.NilError(t, err, "failed to insert mocked trial logs")

			tc.req.TrialId = int32(trial.ID)
			ctx, _ := context.WithTimeout(creds, 100*time.Millisecond)
			tlCl, err := cl.TrialLogs(ctx, tc.req)
			i := 0
			for {
				resp, err := tlCl.Recv()
				if err == io.EOF {
					assert.Assert(t, i == len(tc.matches))
					return
				}
				// context.DeadlineExceeded likely means the stream did not terminate as expected.
				assert.NilError(t, err, "failed to receive trial logs")
				assert.Assert(t, i < len(tc.matches), "received too many logs")
				assert.Equal(t, tc.matches[i], resp.Message)
				i++
			}
		})
	}

	for _, tc := range tests {
		runTestCase(t, tc)
	}
}

func TestTrialLogFollowing(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	_, _, cl, creds, err := testutils.RunMaster(ctx, nil)
	defer cancel()
	assert.NilError(t, err, "failed to start master")

	experiment := testutils.ExperimentModel()
	err = pgDB.AddExperiment(experiment)
	assert.NilError(t, err, "failed to insert experiment")

	trial := testutils.TrialModel(experiment.ID, testutils.WithTrialState(model.ActiveState))
	err = pgDB.AddTrial(trial)
	assert.NilError(t, err, "failed to insert trial")

	ctx, _ = context.WithTimeout(creds, time.Second)
	tlCl, err := cl.TrialLogs(ctx, &apiv1.TrialLogsRequest{
		TrialId: int32(trial.ID),
		Follow:  true,
	})

	// Write logs and turn around and make sure following receives them.
	for trialLogID := 0; trialLogID < 5; trialLogID++ {
		message := fmt.Sprintf("log %d", trialLogID)
		err = pgDB.AddTrialLogs([]*model.TrialLog{
			{
				TrialID: trial.ID,
				Message: message,
			},
		})
		assert.NilError(t, err, "failed to insert mocked trial logs")

		resp, err := tlCl.Recv()
		assert.NilError(t, err, "failed to stream logs")

		// context.DeadlineExceeded likely means the stream did not terminate as expected.
		assert.NilError(t, err, "failed to receive trial logs")
		assert.Equal(t, message, resp.Message)
		assert.Equal(t, trialLogID, int(resp.Id))
	}

	err = pgDB.UpdateTrial(trial.ID, model.CompletedState)
	assert.NilError(t, err, "failed to update trial state")

	_, err = tlCl.Recv()
	assert.Equal(t, err, io.EOF, "log stream didn't terminate with trial")
}
