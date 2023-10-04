//go:build integration
// +build integration

package db

import (
	"testing"
	"time"

	"github.com/stretchr/testify/require"

	"github.com/determined-ai/determined/master/pkg/etc"
	"github.com/determined-ai/determined/master/pkg/model"
	"github.com/determined-ai/determined/master/pkg/ptrs"
)

func TestClusterAPI(t *testing.T) {
	require.NoError(t, etc.SetRootPath(RootFromDB))

	db := MustResolveTestPostgres(t)
	MustMigrateTestPostgres(t, db, MigrationsFromDB)

	_, err := db.GetOrCreateClusterID("")
	require.NoError(t, err, "failed to get or create cluster id")

	// Add a mock user
	user := RequireMockUser(t, db)

	// Add a job
	jID := model.NewJobID()
	jIn := &model.Job{
		JobID:   jID,
		JobType: model.JobTypeExperiment,
		OwnerID: &user.ID,
	}

	err = db.AddJob(jIn)
	require.NoError(t, err, "failed to add job")

	// Add a task
	tID := model.NewTaskID()
	tIn := &model.Task{
		TaskID:    tID,
		JobID:     &jID,
		TaskType:  model.TaskTypeTrial,
		StartTime: time.Now().UTC().Truncate(time.Millisecond),
	}

	err = db.AddTask(tIn)
	require.NoError(t, err, "failed to add task")

	// Add an allocation
	aID := model.AllocationID(string(tID) + "-1")
	aIn := &model.Allocation{
		AllocationID: aID,
		TaskID:       tID,
		Slots:        8,
		ResourcePool: "somethingelse",
		StartTime:    ptrs.Ptr(time.Now().UTC().Truncate(time.Millisecond)),
	}

	err = db.AddAllocation(aIn)
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
	require.NoError(t, db.CloseOpenAllocations(nil))

	// Retrieve the open allocation and check if end time is set to cluster_heartbeat
	aOut, err := db.AllocationByID(aIn.AllocationID)
	require.NoError(t, err)
	require.NotNil(t, aOut, "aOut is Nil")
	require.NotNil(t, aOut.EndTime, "aOut.EndTime is Nil")
	require.Equal(t, *aOut.EndTime, clusterHeartbeat,
		"Expected end time of open allocation is = %q but it is = %q instead",
		clusterHeartbeat.String(), aOut.EndTime.String())
}
