//go:build integration
// +build integration

package db

import (
	"strings"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/stretchr/testify/require"

	"github.com/determined-ai/determined/master/pkg/etc"
	"github.com/determined-ai/determined/master/pkg/model"
	"github.com/determined-ai/determined/proto/pkg/checkpointv1"
	"github.com/determined-ai/determined/proto/pkg/modelv1"
)

func TestDeleteCheckpoints(t *testing.T) {
	etc.SetRootPath(RootFromDB)
	db := MustResolveTestPostgres(t)
	MustMigrateTestPostgres(t, db, MigrationsFromDB)
	user := RequireMockUser(t, db)
	task := RequireMockTask(t, db, &user.ID)
	allocation := requireMockAllocation(t, db, task.TaskID)

	// Create checkpoints
	checkpoint1 := requireMockCheckpoint(t, db, task, allocation, 1)
	checkpoint2 := requireMockCheckpoint(t, db, task, allocation, 2)
	checkpoint3 := requireMockCheckpoint(t, db, task, allocation, 3)

	// Insert a model.
	now := time.Now()
	mdl := model.Model{
		Name:            uuid.NewString(),
		Description:     "some important model",
		CreationTime:    now,
		LastUpdatedTime: now,
		Labels:          []string{"some other label"},
		Username:        user.Username,
	}
	mdlNotes := "some notes"
	var pmdl modelv1.Model
	err := db.QueryProto(
		"insert_model", &pmdl, mdl.Name, mdl.Description, emptyMetadata,
		strings.Join(mdl.Labels, ","), mdlNotes, user.ID,
	)

	require.NoError(t, err)

	// Register checkpoint_1 and checkpoint_2 in ModelRegistry
	var retCkpt1 checkpointv1.Checkpoint
	err = db.QueryProto("get_checkpoint", &retCkpt1, checkpoint1.UUID)
	var retCkpt2 checkpointv1.Checkpoint
	err = db.QueryProto("get_checkpoint", &retCkpt2, checkpoint2.UUID)

	addmv := modelv1.ModelVersion{
		Model:      &pmdl,
		Checkpoint: &retCkpt1,
		Name:       "checkpoint 1",
		Comment:    "empty",
	}
	var mv modelv1.ModelVersion
	err = db.QueryProto(
		"insert_model_version", &mv, pmdl.Id, retCkpt1.Uuid, addmv.Name, addmv.Comment,
		emptyMetadata, strings.Join(addmv.Labels, ","), addmv.Notes, user.ID,
	)
	require.NoError(t, err)

	addmv = modelv1.ModelVersion{
		Model:      &pmdl,
		Checkpoint: &retCkpt2,
		Name:       "checkpoint 2",
		Comment:    "empty",
	}
	err = db.QueryProto(
		"insert_model_version", &mv, pmdl.Id, retCkpt2.Uuid, addmv.Name, addmv.Comment,
		emptyMetadata, strings.Join(addmv.Labels, ","), addmv.Notes, user.ID,
	)
	require.NoError(t, err)

	// Send a list of delete checkpoints uuids the user wants to delete and check if it's in model registry.
	requestedDeleteCheckpoints := []uuid.UUID{checkpoint1.UUID, checkpoint3.UUID}
	expectedDeleteInModelRegistryCheckpoints := make(map[uuid.UUID]bool)
	expectedDeleteInModelRegistryCheckpoints[checkpoint1.UUID] = true
	dCheckpointsInRegistry, err := db.GetRegisteredCheckpoints(requestedDeleteCheckpoints)
	require.NoError(t, err)
	require.Equal(t, expectedDeleteInModelRegistryCheckpoints, dCheckpointsInRegistry)

	validDeleteCheckpoint := checkpoint3.UUID
	numValidDCheckpoints := 1

	db.MarkCheckpointsDeleted([]uuid.UUID{validDeleteCheckpoint})

	var numDStateCheckpoints int

	db.sql.QueryRowx(`SELECT count(c.uuid) AS numC from checkpoints_view AS c WHERE
	c.uuid::text = $1 AND c.state = 'DELETED';`, validDeleteCheckpoint).Scan(&numDStateCheckpoints)
	require.Equal(t, numValidDCheckpoints, numDStateCheckpoints, "didn't correctly delete the valid checkpoints")
}
