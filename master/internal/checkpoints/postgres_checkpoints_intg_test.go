//go:build integration
// +build integration

package checkpoints

import (
	"context"
	"log"
	"math/rand"
	"os"
	"sort"
	"strings"
	"testing"
	"time"
	"unsafe"

	"github.com/google/uuid"
	"github.com/stretchr/testify/require"
	"golang.org/x/exp/maps"

	"github.com/determined-ai/determined/master/internal/db"
	"github.com/determined-ai/determined/master/pkg/etc"
	"github.com/determined-ai/determined/master/pkg/model"
	"github.com/determined-ai/determined/proto/pkg/modelv1"
)

var emptyMetadata = []byte(`{}`)

func sortUUIDSlice(uuids []uuid.UUID) {
	sort.Slice(uuids, func(i, j int) bool {
		return uuids[i].String() < uuids[j].String()
	})
}

func TestMain(m *testing.M) {
	pgDB, err := db.ResolveTestPostgres()
	if err != nil {
		log.Panicln(err)
	}

	err = db.MigrateTestPostgres(pgDB, "file://../../static/migrations", "up")
	if err != nil {
		log.Panicln(err)
	}

	err = etc.SetRootPath("../../static/srv")
	if err != nil {
		log.Panicln(err)
	}

	os.Exit(m.Run())
}

func TestCheckpointByUUID(t *testing.T) {
	ctx := context.Background()

	user := db.RequireMockUser(t, db.SingleDB())

	exp := db.RequireMockExperiment(t, db.SingleDB(), user)
	tr, task := db.RequireMockTrial(t, db.SingleDB(), exp)
	allocation := db.RequireMockAllocation(t, db.SingleDB(), task.TaskID)

	ckpt1 := uuid.New()
	checkpoint := db.MockModelCheckpoint(ckpt1, allocation)
	err := db.AddCheckpointMetadata(ctx, &checkpoint, tr.ID)
	require.NoError(t, err)

	result, err := CheckpointByUUID(ctx, ckpt1)
	require.NoError(t, err)
	require.Equal(t, checkpoint.UUID, *result.UUID)

	// confirm if UUID is not found there is no error
	ckpt2 := uuid.New()
	result, err = CheckpointByUUID(ctx, ckpt2)
	require.NoError(t, err)
	require.Nil(t, result)
}

func TestCheckpointByUUIDs(t *testing.T) {
	ctx := context.Background()

	user := db.RequireMockUser(t, db.SingleDB())

	exp := db.RequireMockExperiment(t, db.SingleDB(), user)
	tr, task := db.RequireMockTrial(t, db.SingleDB(), exp)
	allocation := db.RequireMockAllocation(t, db.SingleDB(), task.TaskID)

	// Create checkpoints
	ckpt1 := uuid.New()
	checkpoint1 := db.MockModelCheckpoint(ckpt1, allocation)
	err := db.AddCheckpointMetadata(ctx, &checkpoint1, tr.ID)
	require.NoError(t, err)

	ckpt2 := uuid.New()
	checkpoint2 := db.MockModelCheckpoint(ckpt2, allocation)
	err = db.AddCheckpointMetadata(ctx, &checkpoint2, tr.ID)
	require.NoError(t, err)

	ckpt3 := uuid.New()
	checkpoint3 := db.MockModelCheckpoint(ckpt3, allocation)
	err = db.AddCheckpointMetadata(ctx, &checkpoint3, tr.ID)
	require.NoError(t, err)

	// Test CheckpointsByUUIDs
	reqCheckpointUUIDs := []uuid.UUID{checkpoint1.UUID, checkpoint2.UUID, checkpoint3.UUID}
	checkpointsByUUIDs, err := CheckpointByUUIDs(ctx, reqCheckpointUUIDs)
	require.NoError(t, err)
	dbCheckpointsUUIDs := []uuid.UUID{
		*checkpointsByUUIDs[0].UUID, *checkpointsByUUIDs[1].UUID, *checkpointsByUUIDs[2].UUID,
	}
	sortUUIDSlice(reqCheckpointUUIDs)
	sortUUIDSlice(dbCheckpointsUUIDs)
	require.ElementsMatch(t, reqCheckpointUUIDs, dbCheckpointsUUIDs)
}

func TestGetModelIDsAssociatedWithCheckpoint(t *testing.T) {
	ctx := context.Background()

	user := db.RequireMockUser(t, db.SingleDB())

	exp := db.RequireMockExperiment(t, db.SingleDB(), user)
	tr, task := db.RequireMockTrial(t, db.SingleDB(), exp)
	allocation := db.RequireMockAllocation(t, db.SingleDB(), task.TaskID)

	ckpt := uuid.New()
	checkpoint := db.MockModelCheckpoint(ckpt, allocation)
	err := db.AddCheckpointMetadata(ctx, &checkpoint, tr.ID)
	require.NoError(t, err)

	// Insert a model.
	now := time.Now()
	mdl := db.Model{
		Name:            uuid.NewString(),
		Description:     "some important model",
		CreationTime:    now,
		LastUpdatedTime: now,
		Labels:          []string{"some other label"},
		UserID:          user.ID,
		WorkspaceID:     1,
	}
	mdlNotes := "some notes1"
	pmdl, err := db.InsertModel(ctx, mdl.Name, mdl.Description, emptyMetadata,
		strings.Join(mdl.Labels, ","), mdlNotes, user.ID, mdl.WorkspaceID)
	require.NoError(t, err)

	retCkpt1, err := db.GetCheckpoint(ctx, checkpoint.UUID.String())
	require.NoError(t, err)

	mv := modelv1.ModelVersion{
		Model:      pmdl,
		Checkpoint: retCkpt1,
		Name:       "checkpoint 1",
		Comment:    "empty",
	}
	_, err = db.InsertModelVersion(ctx, pmdl.Id, retCkpt1.Uuid, mv.Name, mv.Comment,
		emptyMetadata, strings.Join(mv.Labels, ","), mv.Notes, user.ID,
	)
	require.NoError(t, err)

	expmodelIDsCheckpoint := []int32{pmdl.Id}
	modelIDsCheckpoint, err := GetModelIDsAssociatedWithCheckpoint(ctx, checkpoint.UUID)
	require.NoError(t, err)
	require.ElementsMatch(t, expmodelIDsCheckpoint, modelIDsCheckpoint)
}

func TestGetRegisteredCheckpoints(t *testing.T) {
	ctx := context.Background()

	user := db.RequireMockUser(t, db.SingleDB())

	exp := db.RequireMockExperiment(t, db.SingleDB(), user)
	tr, task := db.RequireMockTrial(t, db.SingleDB(), exp)
	allocation := db.RequireMockAllocation(t, db.SingleDB(), task.TaskID)

	ckpt1 := uuid.New()
	checkpoint1 := db.MockModelCheckpoint(ckpt1, allocation)
	err := db.AddCheckpointMetadata(ctx, &checkpoint1, tr.ID)
	require.NoError(t, err)

	ckpt2 := uuid.New()
	checkpoint2 := db.MockModelCheckpoint(ckpt2, allocation)
	err = db.AddCheckpointMetadata(ctx, &checkpoint2, tr.ID)
	require.NoError(t, err)

	ckpt3 := uuid.New()
	checkpoint3 := db.MockModelCheckpoint(ckpt3, allocation)
	err = db.AddCheckpointMetadata(ctx, &checkpoint3, tr.ID)
	require.NoError(t, err)

	// Insert a model.
	now := time.Now()
	mdl := db.Model{
		Name:            uuid.NewString(),
		Description:     "some important model",
		CreationTime:    now,
		LastUpdatedTime: now,
		Labels:          []string{"some other label"},
		UserID:          user.ID,
		WorkspaceID:     1,
	}
	mdlNotes := "some notes2"
	pmdl, err := db.InsertModel(ctx, mdl.Name, mdl.Description, emptyMetadata,
		strings.Join(mdl.Labels, ","), mdlNotes, user.ID, mdl.WorkspaceID,
	)
	require.NoError(t, err)

	retCkpt1, err := db.GetCheckpoint(ctx, checkpoint1.UUID.String())
	require.NoError(t, err)

	mv1 := modelv1.ModelVersion{
		Model:      pmdl,
		Checkpoint: retCkpt1,
		Name:       "checkpoint 1",
		Comment:    "empty",
	}
	_, err = db.InsertModelVersion(ctx, pmdl.Id, retCkpt1.Uuid, mv1.Name, mv1.Comment,
		emptyMetadata, strings.Join(mv1.Labels, ","), mv1.Notes, user.ID,
	)
	require.NoError(t, err)

	retCkpt2, err := db.GetCheckpoint(ctx, checkpoint2.UUID.String())
	require.NoError(t, err)

	mv2 := modelv1.ModelVersion{
		Model:      pmdl,
		Checkpoint: retCkpt2,
		Name:       "checkpoint 2",
		Comment:    "empty",
	}
	_, err = db.InsertModelVersion(ctx, pmdl.Id, retCkpt2.Uuid, mv2.Name, mv2.Comment,
		emptyMetadata, strings.Join(mv2.Labels, ","), mv2.Notes, user.ID,
	)
	require.NoError(t, err)

	checkpoints := []uuid.UUID{checkpoint1.UUID, checkpoint3.UUID}
	expectedRegisteredCheckpoints := make(map[uuid.UUID]bool)
	expectedRegisteredCheckpoints[checkpoint1.UUID] = true
	dCheckpointsInRegistry, err := GetRegisteredCheckpoints(ctx, checkpoints)
	require.NoError(t, err)
	require.Equal(t, expectedRegisteredCheckpoints, dCheckpointsInRegistry)
}

func TestUpdateCheckpointSize(t *testing.T) {
	ctx := context.Background()

	user := db.RequireMockUser(t, db.SingleDB())

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
		exp := db.RequireMockExperiment(t, db.SingleDB(), user)
		experimentIDs = append(experimentIDs, exp.ID)

		for j := 0; j < 2; j++ {
			tr, task := db.RequireMockTrial(t, db.SingleDB(), exp)
			allocation := db.RequireMockAllocation(t, db.SingleDB(), task.TaskID)
			trialIDs = append(trialIDs, tr.ID)

			for k := 0; k < 2; k++ {
				ckpt := uuid.New()
				checkpointIDs = append(checkpointIDs, ckpt)

				checkpoint := db.MockModelCheckpoint(ckpt, allocation)
				checkpoint.Resources = resources[resourcesIndex]
				err := db.AddCheckpointMetadata(ctx, &checkpoint, tr.ID)
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
			err := db.Bun().NewSelect().Table("checkpoints_view").
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
			err := db.Bun().NewSelect().Table("trials").
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
			err := db.Bun().NewSelect().Table("experiments").
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

	user := db.RequireMockUser(t, db.SingleDB())
	exp := db.RequireMockExperiment(t, db.SingleDB(), user)
	tr, task := db.RequireMockTrial(t, db.SingleDB(), exp)
	allocation := db.RequireMockAllocation(t, db.SingleDB(), task.TaskID)

	// Create checkpoints
	ckpt1 := uuid.New()
	checkpoint1 := db.MockModelCheckpoint(ckpt1, allocation)
	err := db.AddCheckpointMetadata(ctx, &checkpoint1, tr.ID)
	require.NoError(t, err)

	ckpt2 := uuid.New()
	checkpoint2 := db.MockModelCheckpoint(ckpt2, allocation)
	err = db.AddCheckpointMetadata(ctx, &checkpoint2, tr.ID)
	require.NoError(t, err)

	ckpt3 := uuid.New()
	checkpoint3 := db.MockModelCheckpoint(ckpt3, allocation)
	err = db.AddCheckpointMetadata(ctx, &checkpoint3, tr.ID)
	require.NoError(t, err)

	// Insert a model.
	now := time.Now()
	mdl := db.Model{
		Name:            uuid.NewString(),
		Description:     "some important model",
		CreationTime:    now,
		LastUpdatedTime: now,
		Labels:          []string{"some other label"},
		UserID:          user.ID,
		WorkspaceID:     1,
	}
	mdlNotes := "some notes3"
	pmdl, err := db.InsertModel(ctx, mdl.Name, mdl.Description, emptyMetadata,
		strings.Join(mdl.Labels, ","), mdlNotes, user.ID, mdl.WorkspaceID,
	)

	require.NoError(t, err)

	// Register checkpoint_1 and checkpoint_2 in ModelRegistry
	retCkpt1, err := db.GetCheckpoint(ctx, checkpoint1.UUID.String())
	require.NoError(t, err)

	retCkpt2, err := db.GetCheckpoint(ctx, checkpoint2.UUID.String())
	require.NoError(t, err)

	mv := modelv1.ModelVersion{
		Model:      pmdl,
		Checkpoint: retCkpt1,
		Name:       "checkpoint 1",
		Comment:    "empty",
	}
	_, err = db.InsertModelVersion(ctx, pmdl.Id, retCkpt1.Uuid, mv.Name, mv.Comment,
		emptyMetadata, strings.Join(mv.Labels, ","), mv.Notes, user.ID,
	)
	require.NoError(t, err)

	mv = modelv1.ModelVersion{
		Model:      pmdl,
		Checkpoint: retCkpt2,
		Name:       "checkpoint 2",
		Comment:    "empty",
	}
	_, err = db.InsertModelVersion(ctx, pmdl.Id, retCkpt2.Uuid, mv.Name, mv.Comment,
		emptyMetadata, strings.Join(mv.Labels, ","), mv.Notes, user.ID,
	)
	require.NoError(t, err)

	validDeleteCheckpoint := checkpoint3.UUID
	numValidDCheckpoints := 1

	require.NoError(t, MarkCheckpointsDeleted(ctx, []uuid.UUID{validDeleteCheckpoint}))

	var numDStateCheckpoints int
	err = db.Bun().NewSelect().
		TableExpr("checkpoints_view AS c").
		ColumnExpr("count(c.uuid) AS numC").
		Where("c.uuid::text = ? AND c.state = 'DELETED'", validDeleteCheckpoint).
		Scan(ctx, &numDStateCheckpoints)
	require.NoError(t, err)

	require.Equal(t, numValidDCheckpoints, numDStateCheckpoints,
		"didn't correctly delete the valid checkpoints")
}

func BenchmarkUpdateCheckpointSize(b *testing.B) {
	ctx := context.Background()
	t := (*testing.T)(unsafe.Pointer(b)) //nolint: gosec // Hack to still use methods that take t.

	user := db.RequireMockUser(t, db.SingleDB())

	var checkpoints []uuid.UUID
	exp := db.RequireMockExperiment(t, db.SingleDB(), user)
	for j := 0; j < 10; j++ {
		t.Logf("Adding trial #%d", j)
		tr, task := db.RequireMockTrial(t, db.SingleDB(), exp)
		allocation := db.RequireMockAllocation(t, db.SingleDB(), task.TaskID)
		for k := 0; k < 10; k++ {
			ckpt := uuid.New()
			checkpoints = append(checkpoints, ckpt)

			resources := make(map[string]int64)
			for r := 0; r < 100000; r++ {
				resources[uuid.New().String()] = rand.Int63n(2500) //nolint: gosec
			}

			checkpoint := db.MockModelCheckpoint(ckpt, allocation)
			checkpoint.Resources = resources

			err := db.AddCheckpointMetadata(ctx, &checkpoint, tr.ID)
			require.NoError(t, err)
		}
	}

	require.NoError(t, MarkCheckpointsDeleted(ctx, checkpoints))
}

func TestPgDB_GroupCheckpointUUIDsByExperimentID(t *testing.T) {
	// Setup some fake data for us to work with.
	expToCkptUUIDs := make(map[int][]uuid.UUID)
	user := db.RequireMockUser(t, db.SingleDB())
	for i := 0; i < 3; i++ {
		exp := db.RequireMockExperiment(t, db.SingleDB(), user)
		tr, tk := db.RequireMockTrial(t, db.SingleDB(), exp)

		var ids []uuid.UUID
		for j := 0; j < 3; j++ {
			id := uuid.New()
			err := db.AddCheckpointMetadata(context.TODO(), &model.CheckpointV2{
				UUID:   id,
				TaskID: tk.TaskID,
			}, tr.ID)
			require.NoError(t, err)
			ids = append(ids, id)
		}

		expToCkptUUIDs[exp.ID] = ids
	}

	type testCase struct {
		name    string
		input   []uuid.UUID
		want    map[int][]uuid.UUID
		wantErr bool
	}

	tests := []testCase{
		{
			name:  "empty is ok",
			input: []uuid.UUID{},
			want:  make(map[int][]uuid.UUID),
		},
		{
			// TODO: A missing checkpoint probably shouldn't be silently removed from the grouping.
			name:  "missing checkpoint returns an error (but it doesn't, yet)",
			input: []uuid.UUID{uuid.New()},
			want:  make(map[int][]uuid.UUID),
		},
	}

	expID := maps.Keys(expToCkptUUIDs)[0]
	ckptUUIDs := expToCkptUUIDs[expID]
	tests = append(tests, testCase{
		name:  "grouping checkpoints but they all belong to one experiment",
		input: expToCkptUUIDs[expID],
		want:  map[int][]uuid.UUID{expID: ckptUUIDs},
	})

	tests = append(tests, testCase{
		name:  "grouping checkpoints across many experiments",
		input: flatten(maps.Values(expToCkptUUIDs)),
		want:  expToCkptUUIDs,
	})

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			groupings, err := GroupCheckpointUUIDsByExperimentID(context.Background(), tt.input)
			if tt.wantErr {
				require.Error(t, err)
			}
			require.NoError(t, err)

			// Unpack the response into a sane format---this API just isn't very usable.
			got := make(map[int][]uuid.UUID)
			for _, g := range groupings {
				ckptStrs := strings.Split(g.CheckpointUUIDSStr, ",")
				var ckpts []uuid.UUID
				for _, ckptStr := range ckptStrs {
					ckpt, err := uuid.Parse(ckptStr)
					if err != nil {
						require.NoError(t, err)
					}
					ckpts = append(ckpts, ckpt)
				}
				got[g.ExperimentID] = append(got[g.ExperimentID], ckpts...)
			}

			require.ElementsMatch(t, maps.Keys(tt.want), maps.Keys(got))
			for wantID, wantCkpts := range tt.want {
				require.ElementsMatch(t, wantCkpts, got[wantID])
			}
		})
	}
}

func flatten[T any](in [][]T) []T {
	var out []T
	for _, i := range in {
		out = append(out, i...)
	}
	return out
}
