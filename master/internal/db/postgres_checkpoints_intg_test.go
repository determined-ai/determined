//go:build integration
// +build integration

package db

import (
	"context"
	"math/rand"
	"sort"
	"strings"
	"testing"
	"time"
	"unsafe"

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

func TestUpdateCheckpointSize(t *testing.T) {
	ctx := context.Background()

	require.NoError(t, etc.SetRootPath(RootFromDB))
	db := MustResolveTestPostgres(t)
	MustMigrateTestPostgres(t, db, MigrationsFromDB)
	user := RequireMockUser(t, db)

	var resources []map[string]int64
	for i := 0; i < 8; i++ {
		resources = append(resources, map[string]int64{"TEST": int64(i) + 1})
	}

	// Create two experiments with two trials each with two checkpoints.
	var experimentIDs []int
	var trialIDs []int
	var checkpointIDs []uuid.UUID

	resourcesIndex := 0
	for i := 0; i < 2; i++ {
		exp := RequireMockExperiment(t, db, user)
		experimentIDs = append(experimentIDs, exp.ID)

		for j := 0; j < 2; j++ {
			tr, task := RequireMockTrial(t, db, exp)
			allocation := RequireMockAllocation(t, db, task.TaskID)
			trialIDs = append(trialIDs, tr.ID)

			for k := 0; k < 2; k++ {
				ckpt := uuid.New()
				checkpointIDs = append(checkpointIDs, ckpt)

				checkpoint := MockModelCheckpoint(ckpt, allocation)
				checkpoint.Resources = resources[resourcesIndex]
				err := AddCheckpointMetadata(ctx, &checkpoint)
				require.NoError(t, err)

				resourcesIndex++
			}
		}
	}

	type expected struct {
		checkpointSizes []int64

		trialCounts []int
		trialSizes  []int64

		experimentCounts []int
		experimentSizes  []int64
	}

	verifySizes := func(e expected) {
		for i, checkpointID := range checkpointIDs {
			var size int64
			err := Bun().NewSelect().Table("checkpoints_view").
				Column("size").
				Where("uuid = ?", checkpointID).
				Scan(context.Background(), &size)
			require.NoError(t, err)
			require.Equal(t, e.checkpointSizes[i], size)
		}

		for i, trialID := range trialIDs {
			actual := struct {
				CheckpointSize  int64
				CheckpointCount int
			}{}
			err := Bun().NewSelect().Table("trials").
				Column("checkpoint_size").
				Column("checkpoint_count").
				Where("id = ?", trialID).
				Scan(context.Background(), &actual)
			require.NoError(t, err)

			require.Equal(t, e.trialCounts[i], actual.CheckpointCount)
			require.Equal(t, e.trialSizes[i], actual.CheckpointSize)
		}

		for i, experimentID := range experimentIDs {
			actual := struct {
				CheckpointSize  int64
				CheckpointCount int
			}{}
			err := Bun().NewSelect().Table("experiments").
				Column("checkpoint_size").
				Column("checkpoint_count").
				Where("id = ?", experimentID).
				Scan(context.Background(), &actual)
			require.NoError(t, err)

			require.Equal(t, e.experimentCounts[i], actual.CheckpointCount)
			require.Equal(t, e.experimentSizes[i], actual.CheckpointSize)
		}
	}

	e := expected{
		checkpointSizes: []int64{1, 2, 3, 4, 5, 6, 7, 8},

		trialCounts: []int{2, 2, 2, 2},
		trialSizes:  []int64{1 + 2, 3 + 4, 5 + 6, 7 + 8},

		experimentCounts: []int{4, 4},
		experimentSizes:  []int64{1 + 2 + 3 + 4, 5 + 6 + 7 + 8},
	}
	verifySizes(e)

	require.NoError(t, MarkCheckpointsDeleted(ctx, checkpointIDs[:2]))
	e.trialCounts = []int{0, 2, 2, 2}
	e.trialSizes = []int64{0, 3 + 4, 5 + 6, 7 + 8}
	e.experimentCounts = []int{2, 4}
	e.experimentSizes = []int64{3 + 4, 5 + 6 + 7 + 8}
	verifySizes(e)

	require.NoError(t, MarkCheckpointsDeleted(ctx, checkpointIDs[3:5]))
	e.trialCounts = []int{0, 1, 1, 2}
	e.trialSizes = []int64{0, 3, 6, 7 + 8}
	e.experimentCounts = []int{1, 3}
	e.experimentSizes = []int64{3, 6 + 7 + 8}
	verifySizes(e)

	require.NoError(t, MarkCheckpointsDeleted(ctx, checkpointIDs))
	e.trialCounts = []int{0, 0, 0, 0}
	e.trialSizes = []int64{0, 0, 0, 0}
	e.experimentCounts = []int{0, 0}
	e.experimentSizes = []int64{0, 0}
	verifySizes(e)
}

func TestDeleteCheckpoints(t *testing.T) {
	ctx := context.Background()

	require.NoError(t, etc.SetRootPath(RootFromDB))
	db := MustResolveTestPostgres(t)
	MustMigrateTestPostgres(t, db, MigrationsFromDB)
	user := RequireMockUser(t, db)
	exp := RequireMockExperiment(t, db, user)
	_, task := RequireMockTrial(t, db, exp)
	allocation := RequireMockAllocation(t, db, task.TaskID)

	// Create checkpoints
	ckpt1 := uuid.New()
	checkpoint1 := MockModelCheckpoint(ckpt1, allocation)
	err := AddCheckpointMetadata(ctx, &checkpoint1)
	require.NoError(t, err)
	ckpt2 := uuid.New()
	checkpoint2 := MockModelCheckpoint(ckpt2, allocation)
	err = AddCheckpointMetadata(ctx, &checkpoint2)
	require.NoError(t, err)
	ckpt3 := uuid.New()
	checkpoint3 := MockModelCheckpoint(ckpt3, allocation)
	err = AddCheckpointMetadata(ctx, &checkpoint3)
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
	require.NoError(t, err)
	dbCheckpointsUUIDs := []uuid.UUID{
		*checkpointsByUUIDs[0].UUID, *checkpointsByUUIDs[1].UUID, *checkpointsByUUIDs[2].UUID,
	}
	sortUUIDSlice(reqCheckpointUUIDs)
	sortUUIDSlice(dbCheckpointsUUIDs)
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

	require.NoError(t, MarkCheckpointsDeleted(ctx, []uuid.UUID{validDeleteCheckpoint}))

	var numDStateCheckpoints int

	err = db.sql.QueryRowx(`SELECT count(c.uuid) AS numC from checkpoints_view AS c WHERE
	c.uuid::text = $1 AND c.state = 'DELETED';`, validDeleteCheckpoint).Scan(&numDStateCheckpoints)
	require.NoError(t, err)
	require.Equal(t, numValidDCheckpoints, numDStateCheckpoints,
		"didn't correctly delete the valid checkpoints")
}

func BenchmarkUpdateCheckpointSize(b *testing.B) {
	ctx := context.Background()
	t := (*testing.T)(unsafe.Pointer(b)) //nolint: gosec // Hack to still use methods that take t.
	require.NoError(t, etc.SetRootPath(RootFromDB))
	db := MustResolveTestPostgres(t)
	MustMigrateTestPostgres(t, db, MigrationsFromDB)
	user := RequireMockUser(t, db)

	var checkpoints []uuid.UUID
	exp := RequireMockExperiment(t, db, user)
	for j := 0; j < 10; j++ {
		t.Logf("Adding trial #%d", j)
		_, task := RequireMockTrial(t, db, exp)
		allocation := RequireMockAllocation(t, db, task.TaskID)
		for k := 0; k < 10; k++ {
			ckpt := uuid.New()
			checkpoints = append(checkpoints, ckpt)

			resources := make(map[string]int64)
			for r := 0; r < 100000; r++ {
				resources[uuid.New().String()] = rand.Int63n(2500) //nolint: gosec
			}

			checkpoint := MockModelCheckpoint(ckpt, allocation)
			checkpoint.Resources = resources

			err := AddCheckpointMetadata(ctx, &checkpoint)
			require.NoError(t, err)
		}
	}

	require.NoError(t, MarkCheckpointsDeleted(ctx, checkpoints))
}
