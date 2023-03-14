//go:build integration
// +build integration

package db

import (
	"context"
	"sort"
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

func sortUUIDSlice(uuids []uuid.UUID) {
	sort.Slice(uuids, func(i, j int) bool {
		return uuids[i].String() < uuids[j].String()
	})
}

func TestDeleteCheckpoints(t *testing.T) {
	require.NoError(t, etc.SetRootPath(RootFromDB))
	db := MustResolveTestPostgres(t)
	MustMigrateTestPostgres(t, db, MigrationsFromDB)
	user := RequireMockUser(t, db)
	exp := RequireMockExperiment(t, db, user)
	tr := RequireMockTrial(t, db, exp)
	allocation := RequireMockAllocation(t, db, tr.TaskID)

	// Create checkpoints
	ckpt1 := uuid.New()
	checkpoint1 := MockModelCheckpoint(ckpt1, tr, allocation)
	err := db.AddCheckpointMetadata(context.TODO(), &checkpoint1)
	require.NoError(t, err)
	ckpt2 := uuid.New()
	checkpoint2 := MockModelCheckpoint(ckpt2, tr, allocation)
	err = db.AddCheckpointMetadata(context.TODO(), &checkpoint2)
	require.NoError(t, err)
	ckpt3 := uuid.New()
	checkpoint3 := MockModelCheckpoint(ckpt3, tr, allocation)
	err = db.AddCheckpointMetadata(context.TODO(), &checkpoint3)
	require.NoError(t, err)

	// Insert a model.
	now := time.Now()
	mdl := model.Model{
		Name:            uuid.NewString(),
		Description:     "some important model",
		CreationTime:    now,
		LastUpdatedTime: now,
		Labels:          []string{"some other label"},
		Username:        user.Username,
		WorkspaceID:     1,
	}
	mdlNotes := "some notes2"
	var pmdl modelv1.Model
	err = db.QueryProto(
		"insert_model", &pmdl, mdl.Name, mdl.Description, emptyMetadata,
		strings.Join(mdl.Labels, ","), mdlNotes, user.ID, mdl.WorkspaceID,
	)

	require.NoError(t, err)

	// Register checkpoint_1 and checkpoint_2 in ModelRegistry
	var retCkpt1 checkpointv1.Checkpoint
	err = db.QueryProto("get_checkpoint", &retCkpt1, checkpoint1.UUID)
	require.NoError(t, err)
	var retCkpt2 checkpointv1.Checkpoint
	err = db.QueryProto("get_checkpoint", &retCkpt2, checkpoint2.UUID)
	require.NoError(t, err)

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

	// Test CheckpointsByUUIDs
	reqCheckpointUUIDs := []uuid.UUID{checkpoint1.UUID, checkpoint2.UUID, checkpoint3.UUID}
	checkpointsByUUIDs, err := db.CheckpointByUUIDs(reqCheckpointUUIDs)
	dbCheckpointsUUIDs := []uuid.UUID{
		*checkpointsByUUIDs[0].UUID, *checkpointsByUUIDs[1].UUID, *checkpointsByUUIDs[2].UUID,
	}
	sortUUIDSlice(reqCheckpointUUIDs)
	sortUUIDSlice(dbCheckpointsUUIDs)
	require.NoError(t, err)
	require.Equal(t, reqCheckpointUUIDs, dbCheckpointsUUIDs)

	// Test GetModelIDsAssociatedWithCheckpoint
	expmodelIDsCheckpoint := []int32{pmdl.Id}
	modelIDsCheckpoint, err := GetModelIDsAssociatedWithCheckpoint(context.TODO(), checkpoint1.UUID)
	require.NoError(t, err)
	require.Equal(t, expmodelIDsCheckpoint, modelIDsCheckpoint)
	// Send a list of delete checkpoints uuids the user wants to delete and
	// check if it's in model registry.
	requestedDeleteCheckpoints := []uuid.UUID{checkpoint1.UUID, checkpoint3.UUID}
	expectedDeleteInModelRegistryCheckpoints := make(map[uuid.UUID]bool)
	expectedDeleteInModelRegistryCheckpoints[checkpoint1.UUID] = true
	dCheckpointsInRegistry, err := db.GetRegisteredCheckpoints(requestedDeleteCheckpoints)
	require.NoError(t, err)
	require.Equal(t, expectedDeleteInModelRegistryCheckpoints, dCheckpointsInRegistry)

	validDeleteCheckpoint := checkpoint3.UUID
	numValidDCheckpoints := 1

	require.NoError(t, db.MarkCheckpointsDeleted([]uuid.UUID{validDeleteCheckpoint}))

	var numDStateCheckpoints int

	err = db.sql.QueryRowx(`SELECT count(c.uuid) AS numC from checkpoints_view AS c WHERE
	c.uuid::text = $1 AND c.state = 'DELETED';`, validDeleteCheckpoint).Scan(&numDStateCheckpoints)
	require.NoError(t, err)
	require.Equal(t, numValidDCheckpoints, numDStateCheckpoints,
		"didn't correctly delete the valid checkpoints")
}
