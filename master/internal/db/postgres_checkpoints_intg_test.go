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

func TestGetDeleteCheckpointsInModelRegistry(t *testing.T) {
	etc.SetRootPath(rootFromDB)
	db := MustResolveTestPostgres(t)
	MustMigrateTestPostgres(t, db, migrationsFromDB)
	user := requireMockUser(t, db)
	exp := requireMockExperiment(t, db, user)
	tr := requireMockTrial(t, db, exp)

	// Create checkpoints
	checkpoint1 := requireMockCheckpoint(t, db, tr, 1)
	checkpoint2 := requireMockCheckpoint(t, db, tr, 2)
	checkpoint3 := requireMockCheckpoint(t, db, tr, 3)

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

	//Register checkpoint_1 and checkpoint_2 in ModelRegistry
	var retCkpt1 checkpointv1.Checkpoint
	err = db.QueryProto("get_checkpoint", &retCkpt1, checkpoint1.Uuid)
	var retCkpt2 checkpointv1.Checkpoint
	err = db.QueryProto("get_checkpoint", &retCkpt2, checkpoint2.Uuid)

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
		"insert_model_version", &mv, pmdl.Id, retCkpt1.Uuid, addmv.Name, addmv.Comment,
		emptyMetadata, strings.Join(addmv.Labels, ","), addmv.Notes, user.ID,
	)
	require.NoError(t, err)

	// Send a list of delete checkpoints uuids the user wants to delete and check if it's in model registry.
	requestedDeleteCheckpoints := []string{checkpoint1.Uuid, checkpoint3.Uuid}
	expectedDeleteInModelRegistryCheckpoints := make(map[string]int)
	expectedDeleteInModelRegistryCheckpoints[checkpoint1.Uuid] = 1
	dCheckpointsInRegistry, err := db.GetDeleteCheckpointsInModelRegistry(requestedDeleteCheckpoints)
	require.NoError(t, err)
	require.Equal(t, expectedDeleteInModelRegistryCheckpoints, dCheckpointsInRegistry)
}
