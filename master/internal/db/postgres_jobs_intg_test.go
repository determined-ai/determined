//go:build integration
// +build integration

package db

import (
	"testing"

	"github.com/shopspring/decimal"
	"github.com/stretchr/testify/require"
	"gotest.tools/assert"

	"github.com/determined-ai/determined/master/pkg/model"
)

func TestAddJob(t *testing.T) {
	db := SingleDB()

	t.Run("add job", func(t *testing.T) {
		jobID := model.NewJobID()
		j := model.Job{JobID: jobID, JobType: model.JobTypeCommand}
		err := db.AddJob(&j)
		require.NoError(t, err)
	})

	t.Run("add job with duplicate id", func(t *testing.T) {
		jobID := model.NewJobID()
		j := model.Job{JobID: jobID, JobType: model.JobTypeCommand}
		err := db.AddJob(&j)
		require.NoError(t, err)

		// change job type and re-add job
		j.JobType = model.JobTypeExperiment
		err = db.AddJob(&j)
		require.Error(t, err)
	})

	t.Run("add job with no job type", func(t *testing.T) {
		err := db.AddJob(&model.Job{JobID: model.NewJobID()})
		require.Error(t, err)
	})
}

func TestJobByID(t *testing.T) {
	db := SingleDB()

	t.Run("add and retrieve job", func(t *testing.T) {
		// create and send job
		sendJob := model.Job{
			JobID:   model.NewJobID(),
			JobType: model.JobTypeExperiment,
			QPos:    decimal.NewFromInt(10),
		}
		err := db.AddJob(&sendJob)
		require.NoError(t, err)

		// retrieve job and test for equality
		recvJob, err := db.JobByID(sendJob.JobID)
		require.NoError(t, err)
		assert.Equal(t, sendJob.JobID, recvJob.JobID)
		assert.Equal(t, sendJob.JobType, recvJob.JobType)
		assert.Equal(t, sendJob.OwnerID, recvJob.OwnerID)
		assert.Equal(t, decimal.NewFromInt(10).Equal(recvJob.QPos), true)
	})

	t.Run("retrieve non-existent job", func(t *testing.T) {
		// retrieve job and test for equality
		recvJob, err := db.JobByID(model.NewJobID())
		require.Error(t, err)
		require.Nil(t, recvJob)
	})
}

func TestUpdateJobPosition(t *testing.T) {
	db := SingleDB()

	t.Run("update position", func(t *testing.T) {
		// create and send job
		origPos := decimal.NewFromInt(10)
		newPos := decimal.NewFromInt(5)
		sendJob := model.Job{
			JobID:   model.NewJobID(),
			JobType: model.JobTypeExperiment,
			QPos:    origPos,
		}
		err := db.AddJob(&sendJob)
		require.NoError(t, err)

		// update job position
		err = db.UpdateJobPosition(sendJob.JobID, newPos)
		require.NoError(t, err)

		// retrieve job and confirm pos update
		recvJob, err := db.JobByID(sendJob.JobID)
		require.NoError(t, err)
		assert.Equal(t, newPos.Equal(recvJob.QPos), true)
	})

	t.Run("update position - negative value", func(t *testing.T) {
		// create and send job
		origPos := decimal.NewFromInt(10)
		newPos := decimal.NewFromInt(-5)
		sendJob := model.Job{
			JobID:   model.NewJobID(),
			JobType: model.JobTypeExperiment,
			QPos:    origPos,
		}
		err := db.AddJob(&sendJob)
		require.NoError(t, err)

		// update job position
		err = db.UpdateJobPosition(sendJob.JobID, newPos)
		require.NoError(t, err)

		// retrieve job and confirm pos update
		recvJob, err := db.JobByID(sendJob.JobID)
		require.NoError(t, err)
		assert.Equal(t, newPos.Equal(recvJob.QPos), true)
	})

	t.Run("update position - empty ID", func(t *testing.T) {
		// create and send job
		origPos := decimal.NewFromInt(10)
		newPos := decimal.NewFromInt(5)
		sendJob := model.Job{
			JobID:   model.NewJobID(),
			JobType: model.JobTypeExperiment,
			QPos:    origPos,
		}
		err := db.AddJob(&sendJob)
		require.NoError(t, err)

		// update job position
		err = db.UpdateJobPosition(model.JobID(""), newPos)
		require.Error(t, err)

		// retrieve job and ensure queue pos not updated
		recvJob, err := db.JobByID(sendJob.JobID)
		require.NoError(t, err)
		assert.Equal(t, origPos.Equal(recvJob.QPos), true)
	})

	t.Run("update position - ID does not exist", func(t *testing.T) {
		// create and send job
		origPos := decimal.NewFromInt(10)
		newPos := decimal.NewFromInt(5)
		sendJob := model.Job{
			JobID:   model.NewJobID(),
			JobType: model.JobTypeExperiment,
			QPos:    origPos,
		}
		err := db.AddJob(&sendJob)
		require.NoError(t, err)

		// update job position for a job that doesn't exist
		err = db.UpdateJobPosition(model.NewJobID(), newPos)
		require.NoError(t, err)
	})
}
