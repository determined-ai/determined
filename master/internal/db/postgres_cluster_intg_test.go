//go:build integration
// +build integration

package db

import (
	"context"
	"fmt"
	"strings"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/stretchr/testify/require"
	"github.com/uptrace/bun"
	"github.com/uptrace/bun/dialect/pgdialect"

	"github.com/determined-ai/determined/master/pkg/etc"
	"github.com/determined-ai/determined/master/pkg/model"
	"github.com/determined-ai/determined/master/pkg/ptrs"
)

func TestClusterAPI(t *testing.T) {
	require.NoError(t, etc.SetRootPath(RootFromDB))

	db, closeDB := MustResolveTestPostgres(t)
	defer closeDB()
	MustMigrateTestPostgres(t, db, MigrationsFromDB)

	_, err := db.GetOrCreateClusterID("")
	require.NoError(t, err, "failed to get or create cluster id")

	_, tIn := CreateMockJobAndTask(t, db)
	tID := tIn.TaskID
	// Add an allocation
	aID := model.AllocationID(string(tID) + "-1")
	aIn := &model.Allocation{
		AllocationID: aID,
		TaskID:       tID,
		Slots:        8,
		ResourcePool: "somethingelse",
		StartTime:    ptrs.Ptr(time.Now().UTC().Truncate(time.Millisecond)),
	}

	err = AddAllocation(context.TODO(), aIn)
	require.NoError(t, err, "failed to add allocation")

	// Add a cluster heartbeat after allocation, so it is as if the master died with it open.
	currentTime := time.Now().UTC().Truncate(time.Millisecond)
	require.NoError(t, db.UpdateClusterHeartBeat(currentTime))

	var clusterHeartbeat time.Time
	err = db.sql.QueryRow("SELECT cluster_heartbeat FROM cluster_id").Scan(&clusterHeartbeat)
	require.NoError(t, err, "error reading cluster_heartbeat from cluster_id table")

	require.Equal(t, currentTime, clusterHeartbeat,
		"Retrieved cluster heartbeat doesn't match the correct time")

	// Don't complete the above allocation and call CloseOpenAllocations
	require.NoError(t, CloseOpenAllocations(context.TODO(), nil))

	// Retrieve the open allocation and check if end time is set to cluster_heartbeat
	aOut, err := AllocationByID(context.TODO(), aIn.AllocationID)
	require.NoError(t, err)
	require.NotNil(t, aOut, "aOut is Nil")
	require.NotNil(t, aOut.EndTime, "aOut.EndTime is Nil")
	require.Equal(t, *aOut.EndTime, clusterHeartbeat,
		"Expected end time of open allocation is = %q but it is = %q instead",
		clusterHeartbeat.String(), aOut.EndTime.String())
}

func CreateMockJobAndTask(t *testing.T, db *PgDB) (*model.Job, *model.Task) {
	// Add a mock user
	user := RequireMockUser(t, db)

	// Add a job
	jID := model.NewJobID()
	jIn := &model.Job{
		JobID:   jID,
		JobType: model.JobTypeExperiment,
		OwnerID: &user.ID,
	}

	err := AddJob(jIn)
	require.NoError(t, err, "failed to add job")

	// Add a task
	tID := model.NewTaskID()
	tIn := &model.Task{
		TaskID:    tID,
		JobID:     &jID,
		TaskType:  model.TaskTypeTrial,
		StartTime: time.Now().UTC().Truncate(time.Millisecond),
	}

	err = AddTask(context.TODO(), tIn)
	require.NoError(t, err, "failed to add task")

	return jIn, tIn
}

type allocAggTest struct {
	name       string
	tzQuery    string
	timeOfDay  string
	numSlots   int
	seconds    int
	queuedTime int
}

func TestUpdateResourceAllocationAggregation(t *testing.T) {
	require.NoError(t, etc.SetRootPath(RootFromDB))

	db, closeDB := MustResolveTestPostgres(t)
	defer closeDB()
	MustMigrateTestPostgres(t, db, MigrationsFromDB)

	ctx := context.Background()
	bunDB := bun.NewDB(db.sql.DB, pgdialect.New())

	today := time.Now()

	tests := []allocAggTest{
		{
			name:       "UTC basic add",
			tzQuery:    `SET TIME ZONE 'Etc/UTC'`,
			timeOfDay:  "04:20:00AM",
			numSlots:   5,
			seconds:    5,
			queuedTime: 2,
		},
		{
			name:       "Positive UTC offset",
			tzQuery:    `SET TIME ZONE 'Europe/Athens'`,
			timeOfDay:  "11:30:00PM",
			numSlots:   5,
			seconds:    10,
			queuedTime: 2,
		},
		{
			name:       "Negative UTC offset",
			tzQuery:    `SET TIME ZONE 'America/Montreal'`,
			timeOfDay:  "01:20:00AM",
			numSlots:   7,
			seconds:    2,
			queuedTime: 2,
		},
	}

	type resourceAggregate struct {
		bun.BaseModel `bun:"table:resource_aggregates"`
		Date          time.Time `bun:"date"`
		Seconds       float64   `bun:"seconds"`
	}

	for ind, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			offset := -1 * (480 - (24 * ind))
			recentDate := today.Add(time.Hour * time.Duration(offset))
			formattedDate, prevSeconds := setupUpdateResourceAllocationAggregation(ctx, t, db,
				bunDB, recentDate, test)
			err := db.UpdateResourceAllocationAggregation()
			require.NoError(t, err)

			ra := resourceAggregate{}
			err = bunDB.NewSelect().
				Model(&ra).
				Where("date = ? AND aggregation_type = ? AND aggregation_key = ?", formattedDate,
					"total", "total").
				Scan(ctx)
			require.NoError(t, err)

			var expectedSeconds interface{} = float64(test.numSlots*test.seconds) + prevSeconds
			var seconds interface{} = ra.Seconds
			var diffTooLarge interface{} = fmt.Sprintf("expected time: %v \nactual time: %v \n",
				expectedSeconds, seconds)
			require.InEpsilon(t, expectedSeconds, seconds, 0.5, diffTooLarge)

			err = bunDB.NewSelect().
				Model(&ra).
				Where("date = ? AND aggregation_type = ? AND aggregation_key = ?", formattedDate,
					"queued", "total").
				Scan(ctx)
			require.NoError(t, err)

			var expectedQueuedTime interface{} = test.queuedTime
			seconds = ra.Seconds
			diffTooLarge = fmt.Sprintf("expected queued time: %v \nactual queued "+
				" time: %v \n",
				expectedQueuedTime, seconds)
			require.InEpsilon(t, expectedQueuedTime, seconds, 0.5, diffTooLarge)
		})
	}
}

func setupUpdateResourceAllocationAggregation(ctx context.Context, t *testing.T, db *PgDB,
	bunDB *bun.DB, recentDate time.Time, test allocAggTest,
) (string, float64) {
	// (Setup) Set the timezone.
	_, err := bunDB.NewRaw(test.tzQuery).Exec(ctx)
	require.NoError(t, err)

	_, tIn := CreateMockJobAndTask(t, db)
	tID := tIn.TaskID

	formattedDate := recentDate.Format(time.DateOnly)
	yearMonthDay := strings.Split(formattedDate, "-")
	startDate := fmt.Sprintf("%s/%s %s %s +00", yearMonthDay[1], yearMonthDay[2], test.timeOfDay,
		yearMonthDay[0])

	// The total aggregated seconds from previously added allocations on the day recentDate.
	var prevSeconds float64
	err = bunDB.NewRaw(`
	WITH d AS (
    SELECT
        tsrange(
            ?::timestamp,
            (?::timestamp + interval '1 day')
        ) AS period
	),
	allocs_in_range AS (
    SELECT
        extract(
            EPOCH
            FROM
            upper(d.period * alloc.range) - lower(d.period * alloc.range)
        ) * alloc.slots::float AS seconds
    FROM
        (
            SELECT
				slots,
                tsrange(start_time, greatest(start_time, end_time)) AS range
            FROM
                allocations
            WHERE
                start_time IS NOT NULL
        ) AS alloc,
        d
    WHERE
        d.period && alloc.range
	)
	SELECT coalesce(sum(allocs_in_range.seconds), 0) FROM allocs_in_range
	`, formattedDate, formattedDate).Scan(ctx, &prevSeconds)
	require.NoError(t, err)

	// (Setup) the allocation's start and end time.
	startTime, err := time.Parse("01/02 03:04:05PM 2006 -07", startDate)
	require.NoError(t, err)
	endTime := startTime.Add(time.Second * time.Duration(test.seconds))

	allocID := uuid.NewString()
	alloc := model.Allocation{
		AllocationID: *model.NewAllocationID(&allocID),
		TaskID:       tID,
		Slots:        test.numSlots,
		ResourcePool: uuid.NewString(),
		StartTime:    &startTime,
		EndTime:      &endTime,
	}

	// The function UpdateResourceAllocationAggregation starts aggregating allocations from the day
	// after the highest date in the resource_aggregates table. Therefore, we need to ensure that
	// the highest date in resource_aggregates is less than the date on which we are adding the
	// allocation in this setup.
	_, err = bunDB.NewDelete().
		Table("resource_aggregates").
		Where("date >= ?", formattedDate).
		Exec(ctx)
	require.NoError(t, err)

	_, err = bunDB.NewInsert().Model(&alloc).Exec(ctx)
	require.NoError(t, err)

	queuedEndTime := startTime.Add(time.Second * time.Duration(test.queuedTime))
	taskStats := model.TaskStats{
		AllocationID: model.AllocationID(allocID),
		EventType:    "QUEUED",
		StartTime:    &startTime,
		EndTime:      &queuedEndTime,
	}
	_, err = bunDB.NewInsert().Model(&taskStats).Exec(ctx)
	require.NoError(t, err)

	return formattedDate, prevSeconds
}
