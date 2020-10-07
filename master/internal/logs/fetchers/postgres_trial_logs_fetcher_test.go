package fetchers

import (
	"github.com/determined-ai/determined/master/pkg/check"
	"github.com/golang/protobuf/ptypes"
	"github.com/golang/protobuf/ptypes/timestamp"
	"testing"
	"time"

	"gotest.tools/assert"

	"github.com/determined-ai/determined/master/internal/db"
	"github.com/determined-ai/determined/master/internal/logs"
	"github.com/determined-ai/determined/master/pkg/model"
	"github.com/determined-ai/determined/proto/pkg/filters"
)

func insertFakeLogs(t *testing.T, db *db.PgDB, trialID int, logs []*model.TrialLog) {
	assert.NilError(t, db.Exec("TRUNCATE TABLE trial_logs"))
	assert.NilError(t, db.Exec("ALTER TABLE trial_logs DISABLE TRIGGER ALL"))
	for i := range logs {
		logs[i].TrialID = trialID
	}
	assert.NilError(t, db.AddTrialLogs(logs))
	assert.NilError(t, db.Exec("ALTER TABLE trial_logs ENABLE TRIGGER ALL"))
}

func TestPostgresTrialLogsFetcher(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping integration test")
	}

	db, err := db.MustConnectPostgres("postgres://postgres:postgres@127.0.0.1:5432/determined?sslmode=disable")
	assert.NilError(t, err)

	type testCase struct {
		name    string
		filters []*filters.Filter
		logs    []*model.TrialLog
		validationError string
		matches int
		checker func(*testing.T) func(record logs.Record) error
	}

	trialID := 1
	agent0, agent1, agent2 := "elated-backward-cat", "sad-testfailed-cat", "neutral-cat"
	rank0, rank1, rank2 := 0, 1, 2
	time0 := time.Now()
	time1, time2 := time0.Add(time.Second), time0.Add(2 * time.Second)
	tests := []testCase{
		{
			name: "categorical text where equals",
			filters: []*filters.Filter{
				{
					Field:     "agent_id",
					Operation: filters.Filter_OPERATION_EQUAL,
					Values: toFilterStringValues([]string{agent0}),
				},
			},
			logs: []*model.TrialLog{
				{
					AgentID: &agent0,
				},
				{
					AgentID: &agent1,
				},
			},
			matches: 1,
			checker: func(t *testing.T) func(record logs.Record) error {
				return func(record logs.Record) error {
					assert.DeepEqual(t, record.(*model.TrialLog).AgentID, &agent0)
					return nil
				}
			},
		},
		{
			name: "categorical text where not equals",
			filters: []*filters.Filter{
				{
					Field:     "agent_id",
					Operation: filters.Filter_OPERATION_NOT_EQUAL,
					Values:    toFilterStringValues([]string{agent1}),
				},
			},
			logs: []*model.TrialLog{
				{
					AgentID: &agent0,
				},
				{
					AgentID: &agent1,
				},
			},
			matches: 1,
			checker: func(t *testing.T) func(record logs.Record) error {
				return func(record logs.Record) error {
					assert.DeepEqual(t, record.(*model.TrialLog).AgentID, &agent0)
					return nil
				}
			},
		},
		{
			name: "categorical text where in",
			filters: []*filters.Filter{
				{
					Field:     "agent_id",
					Operation: filters.Filter_OPERATION_IN,
					Values:    toFilterStringValues([]string{agent0, agent2}),
				},
			},
			logs: []*model.TrialLog{
				{
					AgentID: &agent0,
				},
				{
					AgentID: &agent2,
				},
				{
					AgentID: &agent1,
				},
			},
			matches: 2,
			checker: func(t *testing.T) func(record logs.Record) error {
				return func(record logs.Record) error {
					trialLog := record.(*model.TrialLog)
					assert.Assert(t, trialLog.AgentID != nil, "agent_id was nil")
					assert.NilError(t, check.Contains(*trialLog.AgentID, []interface{}{agent0, agent2}),
						"fetched filtered agent_id")
					return nil
				}
			},
		},
		{
			name: "categorical integer equals",
			filters: []*filters.Filter{
				{
					Field:     "rank_id",
					Operation: filters.Filter_OPERATION_EQUAL,
					Values:    toFilterIntValues([]int32{int32(rank0)}),
				},
			},
			logs: []*model.TrialLog{
				{
					RankID: &rank0,
				},
				{
					RankID: &rank2,
				},
				{
					RankID: &rank1,
				},
			},
			matches: 1,
			checker: func(t *testing.T) func(record logs.Record) error {
				return func(record logs.Record) error {
					assert.DeepEqual(t, record.(*model.TrialLog).RankID, &rank0)
					return nil
				}
			},
		},
		{
			name: "categorical integer where in",
			filters: []*filters.Filter{
				{
					Field:     "rank_id",
					Operation: filters.Filter_OPERATION_IN,
					Values:    toFilterIntValues([]int32{int32(rank0), int32(rank1)}),
				},
			},
			logs: []*model.TrialLog{
				{
					RankID: &rank0,
				},
				{
					RankID: &rank1,
				},
				{
					RankID: &rank2,
				},
			},
			matches: 2,
			checker: func(t *testing.T) func(record logs.Record) error {
				return func(record logs.Record) error {
					trialLog := record.(*model.TrialLog)
					assert.Assert(t, trialLog.RankID != nil, "rank_id was nil")
					assert.NilError(t, check.Contains(*trialLog.RankID, []interface{}{rank0, rank1}),
						"fetched filtered rank_id")
					return nil
				}
			},
		},
		{
			name: "ordered where greater than",
			filters: []*filters.Filter{
				{
					Field:     "timestamp",
					Operation: filters.Filter_OPERATION_GREATER,
					Values:    toFilterTimestampValues([]time.Time{time1}),
				},
			},
			logs: []*model.TrialLog{
				{
					Timestamp: &time0,
				},
				{
					Timestamp: &time1,
				},
				{
					Timestamp: &time2,
				},
			},
			matches: 1,
			checker: func(t *testing.T) func(record logs.Record) error {
				return func(record logs.Record) error {
					trialLog := record.(*model.TrialLog)
					assert.Assert(t, trialLog.Timestamp.After(time1), "timestamp wasn't filtered")
					return nil
				}
			},
		},
		{
			name: "ordered where less than",
			filters: []*filters.Filter{
				{
					Field:     "timestamp",
					Operation: filters.Filter_OPERATION_LESS,
					Values:    toFilterTimestampValues([]time.Time{time2}),
				},
			},
			logs: []*model.TrialLog{
				{
					Timestamp: &time0,
				},
				{
					Timestamp: &time1,
				},
				{
					Timestamp: &time2,
				},
			},
			matches: 2,
			checker: func(t *testing.T) func(record logs.Record) error {
				return func(record logs.Record) error {
					trialLog := record.(*model.TrialLog)
					assert.Assert(t, trialLog.Timestamp.Before(time2), "timestamp wasn't filtered")
					return nil
				}
			},
		},
		{
			name: "timestamp fields only accept timestamps",
			filters: []*filters.Filter{
				{
					Field:     "timestamp",
					Operation: filters.Filter_OPERATION_LESS,
					Values:    toFilterStringValues([]string{"12:12:12 not a time"}),
				},
			},
			validationError: "unsupported values *filters.Filter_StringValues for filter timestamp",
		},
		{
			name: "ordered fields only accept ordering operations",
			filters: []*filters.Filter{
				{
					Field:     "timestamp",
					Operation: filters.Filter_OPERATION_IN,
					Values:    toFilterTimestampValues([]time.Time{time0}),
				},
			},
			validationError: "unsupported operation OPERATION_IN in filter timestamp",
		},
		{
			name: "string fields only accept strings",
			filters: []*filters.Filter{
				{
					Field:     "agent_id",
					Operation: filters.Filter_OPERATION_EQUAL,
					Values:    toFilterIntValues([]int32{1, 2, 3}),
				},
			},
			validationError: "unsupported values *filters.Filter_IntValues for filter agent_id",
		},
		{
			name: "categorical fields only accept categorical operations",
			filters: []*filters.Filter{
				{
					Field:     "agent_id",
					Operation: filters.Filter_OPERATION_LESS,
					Values:    toFilterStringValues([]string{"ok"}),
				},
			},
			validationError: "unsupported operation OPERATION_LESS in filter agent_id",
		},
		{
			name: "int fields only accept ints",
			filters: []*filters.Filter{
				{
					Field:     "rank_id",
					Operation: filters.Filter_OPERATION_IN,
					Values:    toFilterStringValues([]string{"12:12:12 not a time"}),
				},
			},
			validationError: "unsupported values *filters.Filter_StringValues for filter rank_id",
		},
		{
			name: "missing values",
			filters: []*filters.Filter{
				{
					Field:     "timestamp",
					Operation: filters.Filter_OPERATION_LESS,
					Values:    toFilterTimestampValues(nil),
				},
			},
			validationError: "operation OPERATION_LESS in filter timestamp requires arguments",
		},
	}

	runTestCase := func(t *testing.T, tc testCase) {
		t.Run(tc.name, func(t *testing.T) {
			insertFakeLogs(t, db, trialID, tc.logs)

			fetcher, err := NewPostgresTrialLogsFetcher(db, 1, 0, tc.filters)
			if tc.validationError != "" {
				assert.Assert(t, err != nil, "expected validation error but found none")
				assert.ErrorContains(t, err, tc.validationError)
				return
			} else {
				assert.NilError(t, err, "could not create fetcher")
			}

			batch, err := fetcher.Fetch(10, false)
			assert.NilError(t, err, "could not fetch batch")
			assert.Equal(t, tc.matches, batch.Size(), "incorrect number of matches")

			err = batch.ForEach(tc.checker(t))
			assert.NilError(t, err, "could not check logs")
		})
	}

	for _, tc := range tests {
		runTestCase(t, tc)
	}
}

func toFilterStringValues(vals []string) *filters.Filter_StringValues {
	return &filters.Filter_StringValues{StringValues: &filters.StringValues{Values: vals}}
}

func toFilterIntValues(vals []int32) *filters.Filter_IntValues {
	return &filters.Filter_IntValues{IntValues: &filters.IntValues{Values: vals}}
}

func toFilterTimestampValues(vals []time.Time) *filters.Filter_TimestampValues {
	var tss []*timestamp.Timestamp
	for _, t := range vals {
		ts, err := ptypes.TimestampProto(t)
		if err != nil {
			panic(err)
		}
		tss = append(tss, ts)
	}
	return &filters.Filter_TimestampValues{TimestampValues: &filters.TimestampValues{Values: tss}}
}
