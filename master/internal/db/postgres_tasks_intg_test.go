//go:build integration
// +build integration

package db

import (
	"fmt"
	"reflect"
	"testing"
	"time"

	"github.com/davecgh/go-spew/spew"
	"github.com/determined-ai/determined/master/pkg/etc"
	"github.com/determined-ai/determined/master/pkg/model"
	"github.com/determined-ai/determined/master/pkg/ptrs"
	"github.com/google/uuid"
	"github.com/stretchr/testify/require"
	"gopkg.in/guregu/null.v3"
)

// TestJobTaskAndAllocationAPI, in lieu of an ORM, ensures that the mappings into and out of the
// database are total. We should look into an ORM in the near to medium term future.
func TestJobTaskAndAllocationAPI(t *testing.T) {
	etc.SetRootPath(rootFromDB)
	db := MustResolveTestPostgres(t)
	MustMigrateTestPostgres(t, db, migrationsFromDB)

	// Add a mock user.
	user := model.User{
		Username:     uuid.NewString(),
		PasswordHash: null.NewString("", false),
		Admin:        false,
		Active:       true,
	}
	err := db.AddUser(&user, nil)
	require.NoError(t, err, "failed to add user")

	// Add a job.
	jID := model.NewJobID()
	jIn := &model.Job{
		JobID:   jID,
		JobType: model.JobTypeExperiment,
		OwnerID: &user.ID,
	}
	err = db.AddJob(jIn)
	require.NoError(t, err, "failed to add job")

	// Retrieve it back and make sure the mapping is exhaustive.
	jOut, err := db.JobByID(jID)
	require.NoError(t, err, "failed to retrieve job")
	require.True(t, reflect.DeepEqual(jIn, jOut), pprintedExpect(jIn, jOut))

	// Add a task.
	tID := model.NewTaskID()
	tIn := &model.Task{
		TaskID:    tID,
		JobID:     jID,
		TaskType:  model.TaskTypeTrial,
		StartTime: time.Now().UTC().Truncate(time.Millisecond),
	}
	err = db.AddTask(tIn)
	require.NoError(t, err, "failed to add task")

	// Retrieve it back and make sure the mapping is exhaustive.
	tOut, err := db.TaskByID(tID)
	require.NoError(t, err, "failed to retrieve task")
	require.True(t, reflect.DeepEqual(tIn, tOut), pprintedExpect(tIn, tOut))

	// Complete it.
	tIn.EndTime = ptrs.TimePtr(time.Now().UTC().Truncate(time.Millisecond))
	err = db.CompleteTask(tID, *tIn.EndTime)
	require.NoError(t, err, "failed to mark task completed")

	// Re-retrieve it back and make sure the mapping is still exhaustive.
	tOut, err = db.TaskByID(tID)
	require.NoError(t, err, "failed to re-retrieve task")
	require.True(t, reflect.DeepEqual(tIn, tOut), pprintedExpect(tIn, tOut))

	// And an allocation.
	aID := model.NewAllocationID(string(tID) + "-1")
	aIn := &model.Allocation{
		AllocationID: aID,
		TaskID:       tID,
		Slots:        8,
		AgentLabel:   "something",
		ResourcePool: "somethingelse",
		StartTime:    time.Now().UTC().Truncate(time.Millisecond),
	}
	err = db.AddAllocation(aIn)
	require.NoError(t, err, "failed to add allocation")

	// Retrieve it back and make sure the mapping is exhaustive.
	aOut, err := db.AllocationByID(aIn.AllocationID)
	require.NoError(t, err, "failed to retrieve allocation")
	require.True(t, reflect.DeepEqual(aIn, aOut), pprintedExpect(aIn, aOut))

	// Complete it.
	aIn.EndTime = ptrs.TimePtr(time.Now().UTC().Truncate(time.Millisecond))
	err = db.CompleteAllocation(aIn)
	require.NoError(t, err, "failed to mark allocation completed")

	// Re-retrieve it back and make sure the mapping is still exhaustive.
	aOut, err = db.AllocationByID(aIn.AllocationID)
	require.NoError(t, err, "failed to re-retrieve allocation")
	require.True(t, reflect.DeepEqual(aIn, aOut), pprintedExpect(aIn, aOut))
}

func pprintedExpect(expected, got interface{}) string {
	return fmt.Sprintf("expected \n\t%s\ngot\n\t%s", spew.Sdump(expected), spew.Sdump(got))
}
