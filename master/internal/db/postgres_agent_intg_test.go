//go:build integration
// +build integration

package db

import (
	"context"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/stretchr/testify/require"
	"github.com/uptrace/bun"

	"github.com/determined-ai/determined/master/pkg/etc"
	"github.com/determined-ai/determined/master/pkg/model"
)

type agentStatsRow struct {
	bun.BaseModel `bun:"table:agent_stats"`

	ResourcePool string
	AgentID      string
	Slots        int
	StartTime    time.Time
	EndTime      *time.Time
}

func agentStatsForRP(t *testing.T, rp string) []*agentStatsRow {
	var res []*agentStatsRow
	require.NoError(t, Bun().NewSelect().Model(&res).
		Where("resource_pool = ?", rp).
		Order("start_time").
		Scan(context.TODO(), &res))
	return res
}

func TestRecordAgentStats(t *testing.T) {
	require.NoError(t, etc.SetRootPath(RootFromDB))
	db, close := MustResolveTestPostgres(t)
	defer close()
	MustMigrateTestPostgres(t, db, MigrationsFromDB)

	lowStartTimeBound := time.Now()

	rp := uuid.New().String()
	agentID := uuid.New().String()
	stats := &model.AgentStats{
		ResourcePool: rp,
		AgentID:      agentID,
		Slots:        1,
	}
	require.NoError(t, db.RecordAgentStats(stats))

	// Check we inserted it properly.
	highStartTimeBound := time.Now()
	actualList := agentStatsForRP(t, rp)
	require.Len(t, actualList, 1)
	actual := actualList[0]
	require.WithinRange(t, actual.StartTime, lowStartTimeBound, highStartTimeBound)
	expected := &agentStatsRow{
		ResourcePool: rp,
		AgentID:      agentID,
		Slots:        1,
		StartTime:    actual.StartTime,
		EndTime:      nil,
	}
	require.Equal(t, expected, actual)

	// Errors when we try to RecordAgentStats again due to zero rows affected.
	require.ErrorContains(t, db.RecordAgentStats(stats), "0 rows affected")

	// End our stat.
	lowEndTimeBound := time.Now()
	require.NoError(t, EndAgentStats(stats))

	highEndTimeBound := time.Now()
	actualList = agentStatsForRP(t, rp)
	require.Len(t, actualList, 1)
	actual = actualList[0]
	require.NotNil(t, actual.EndTime)
	require.WithinRange(t, *actual.EndTime, lowEndTimeBound, highEndTimeBound)

	// If we end our agent stats we can add another.
	require.NoError(t, db.RecordAgentStats(stats))
	require.Len(t, agentStatsForRP(t, rp), 2)
}

func TestEndAllAgentStats(t *testing.T) {
	ctx := context.Background()

	require.NoError(t, etc.SetRootPath(RootFromDB))
	db, cleanup := MustResolveNewPostgresDatabase(t)
	defer cleanup()
	MustMigrateTestPostgres(t, db, MigrationsFromDB)

	setTimesTo := func(id string, startTime time.Time, endTime *time.Time) {
		_, err := Bun().NewUpdate().Model(&agentStatsRow{}).
			Set("start_time = ?", startTime).
			Set("end_time = ?", endTime).
			Where("agent_id = ?", id).
			Exec(ctx)
		require.NoError(t, err)
	}

	rp := uuid.New().String()

	// Start is before our cluster heartbeat.
	a0 := uuid.New().String()
	a0Start := time.Date(2021, 10, 9, 0, 0, 0, 0, time.Local).Truncate(time.Millisecond)
	require.NoError(t, db.RecordAgentStats(&model.AgentStats{AgentID: a0, ResourcePool: rp}))
	setTimesTo(a0, a0Start, nil)

	// Cluster heartbeat between these.
	// TODO(!!!) make cluster heartbeat a timestamptz.
	_, err := db.GetOrCreateClusterID("")
	require.NoError(t, err)
	heartBeatTime := time.Date(2021, 10, 10, 0, 0, 0, 0, time.Local).Truncate(time.Millisecond)
	require.NoError(t, db.UpdateClusterHeartBeat(heartBeatTime.UTC()))

	// Start is after our cluster heartbeat.
	a1 := uuid.New().String()
	a1Start := time.Date(2021, 10, 11, 0, 0, 0, 0, time.Local).Truncate(time.Millisecond)
	require.NoError(t, db.RecordAgentStats(&model.AgentStats{AgentID: a1, ResourcePool: rp}))
	setTimesTo(a1, a1Start, nil)

	// Is ended so start time shouldn't get touched.
	a2 := uuid.New().String()
	a2Start := time.Date(2021, 10, 13, 0, 0, 0, 0, time.Local).Truncate(time.Millisecond)
	a2End := time.Date(2021, 10, 14, 0, 0, 0, 0, time.Local).Truncate(time.Millisecond)
	require.NoError(t, db.RecordAgentStats(&model.AgentStats{AgentID: a2, ResourcePool: rp}))
	setTimesTo(a2, a2Start, &a2End)

	require.NoError(t, db.EndAllAgentStats())

	stats := agentStatsForRP(t, rp)
	require.Len(t, stats, 3)
	require.Equal(t,
		[]*time.Time{&heartBeatTime, &a1Start, &a2End},
		[]*time.Time{stats[0].EndTime, stats[1].EndTime, stats[2].EndTime})
}
