//go:build integration
// +build integration

package db

import (
	"context"
	"encoding/json"
	"math"
	"math/rand"
	"os"
	"sort"
	"strconv"
	"strings"
	"sync"
	"testing"
	"time"
	"unsafe"

	"github.com/google/uuid"
	"github.com/stretchr/testify/require"
	"github.com/uptrace/bun"
	"google.golang.org/protobuf/types/known/structpb"
	"gopkg.in/yaml.v3" // Can't use ghodss/yaml since NaNs error.

	"github.com/determined-ai/determined/master/pkg/etc"
	"github.com/determined-ai/determined/master/pkg/model"
	"github.com/determined-ai/determined/proto/pkg/checkpointv1"
	"github.com/determined-ai/determined/proto/pkg/commonv1"
	"github.com/determined-ai/determined/proto/pkg/modelv1"
	"github.com/determined-ai/determined/proto/pkg/trialv1"
)

func genTrial(
	ctx context.Context, t *testing.T, db *PgDB, exp *model.Experiment, numMetrics, numSteps int,
) {
	type step struct {
		bun.BaseModel `bun:"table:steps"`
		TrialID       int
		TrialRunID    int
		Metrics       map[string]any
		TotalBatches  int
		EndTime       time.Time
	}

	trialID := RequireMockTrial(t, db, exp).ID
	metrics := make([]step, 0, numSteps)

	endTime := time.Now()
	for i := 0; i < numSteps; i++ {
		m := make(map[string]float64, numMetrics)
		for j := 0; j < numMetrics; j++ {
			m[strconv.Itoa(j)] = rand.Float64() //nolint: gosec
		}

		metrics = append(metrics, step{
			TrialID:    trialID,
			TrialRunID: 1,
			Metrics: map[string]any{
				"avg_metrics": m,
			},
			TotalBatches: i,
			EndTime:      endTime,
		})
	}

	_, err := Bun().NewInsert().Model(&metrics).Exec(ctx)
	require.NoError(t, err)

	type val struct {
		bun.BaseModel `bun:"table:validations"`
		TrialID       int
		TrialRunID    int
		Metrics       map[string]any
		TotalBatches  int
		EndTime       time.Time
	}

	vals := make([]val, 0, numSteps)
	for i := 0; i < numSteps/100; i++ {
		m := make(map[string]float64, numMetrics)
		for j := 0; j < numMetrics; j++ {
			m[strconv.Itoa(j)] = rand.Float64() //nolint: gosec
		}

		vals = append(vals, val{
			TrialID:    trialID,
			TrialRunID: 1,
			Metrics: map[string]any{
				"validation_metrics": m,
			},
			TotalBatches: i,
			EndTime:      endTime,
		})
	}

	_, err = Bun().NewInsert().Model(&vals).Exec(ctx)
	require.NoError(t, err)
}

func TestGenLargeDB(t *testing.T) {
	numTrials := 10000
	numSteps := 25000
	numMetrics := 5
	numWorkers := 10

	ctx := context.Background()
	require.NoError(t, etc.SetRootPath(RootFromDB))
	db := MustResolveTestPostgres(t)
	user := RequireMockUser(t, db)
	exp := RequireMockExperiment(t, db, user)

	mu := sync.Mutex{}
	c := 0

	start := time.Now()
	trialsPerPercent := numTrials/100 + 1

	var wg sync.WaitGroup
	for i := 0; i < numWorkers; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			for j := 0; j < numTrials/numWorkers; j++ {
				genTrial(ctx, t, db, exp, numMetrics, numSteps)

				mu.Lock()
				c++
				if c%trialsPerPercent == 0 {
					t.Logf("%d%% done in %v", c/trialsPerPercent, time.Now().Sub(start))
					start = time.Now()
				}
				mu.Unlock()
			}
		}()
	}
	wg.Wait()
}

func sortUUIDSlice(uuids []uuid.UUID) {
	sort.Slice(uuids, func(i, j int) bool {
		return uuids[i].String() < uuids[j].String()
	})
}

var stepsCompleted int32

func addMetrics(ctx context.Context,
	t *testing.T, db *PgDB, trial *model.Trial, trainMetricsJSON, valMetricsJSON string,
) {
	var trainMetrics []map[string]any
	require.NoError(t, json.Unmarshal([]byte(trainMetricsJSON), &trainMetrics))

	curStep := stepsCompleted
	for _, m := range trainMetrics {
		metrics, err := structpb.NewStruct(m)
		require.NoError(t, err)
		require.NoError(t, db.AddTrainingMetrics(ctx, &trialv1.TrialMetrics{
			TrialId:        int32(trial.ID),
			TrialRunId:     0,
			StepsCompleted: curStep,
			Metrics: &commonv1.Metrics{
				AvgMetrics: metrics,
			},
		}))
		curStep++
	}

	curStep = stepsCompleted
	var valMetrics []map[string]any
	require.NoError(t, json.Unmarshal([]byte(valMetricsJSON), &valMetrics))
	for _, m := range valMetrics {
		metrics, err := structpb.NewStruct(m)
		require.NoError(t, err)
		require.NoError(t, db.AddValidationMetrics(ctx, &trialv1.TrialMetrics{
			TrialId:        int32(trial.ID),
			TrialRunId:     0,
			StepsCompleted: curStep,
			Metrics: &commonv1.Metrics{
				AvgMetrics: metrics,
			},
		}))
		curStep++
	}

	stepsCompleted += int32(len(trainMetrics) + len(valMetrics))
}

func addSomeMetricInPast(ctx context.Context, t *testing.T, trialID int) {
	metric := struct {
		bun.BaseModel `bun:"table:steps"`
		TrialID       int
		TrialRunID    int
		Metrics       map[string]any
		TotalBatches  int
		EndTime       time.Time
	}{
		TrialID:    trialID,
		TrialRunID: 1,
		Metrics: map[string]any{
			"avg_metrics": map[string]any{
				"train_met_from_past": 1.0,
			},
		},
		TotalBatches: 1,
		EndTime:      time.Now().AddDate(0, 0, -1),
	}
	_, err := Bun().NewInsert().Model(&metric).Exec(ctx)
	require.NoError(t, err)

	valMetric := struct {
		bun.BaseModel `bun:"table:validations"`
		TrialID       int
		TrialRunID    int
		Metrics       map[string]any
		TotalBatches  int
		EndTime       time.Time
	}{
		TrialID:    trialID,
		TrialRunID: 1,
		Metrics: map[string]any{
			"validation_metrics": map[string]any{
				"val_met_from_past": 1.0,
			},
		},
		TotalBatches: 1,
		EndTime:      time.Now().AddDate(0, 0, -1),
	}
	_, err = Bun().NewInsert().Model(&valMetric).Exec(ctx)
	require.NoError(t, err)
}

func runSummaryMigration(t *testing.T) {
	bytes, err := os.ReadFile("../../static/migrations/20230405164440_add-summary-metrics.tx.up.sql")
	require.NoError(t, err)

	_, err = Bun().Exec(string(bytes))
	require.NoError(t, err)
}

func nanEqual(t *testing.T, expected, actual map[string]summaryMetrics) {
	e, err := yaml.Marshal(&expected)
	require.NoError(t, err)

	a, err := yaml.Marshal(&actual)
	require.NoError(t, err)

	require.Equal(t, string(e), string(a))
}

func validateSummaryMetrics(ctx context.Context, t *testing.T, trialID int,
	expectedTrain map[string]summaryMetrics,
	expectedVal map[string]summaryMetrics,
) {
	query := `SELECT name,
summary_metrics->'avg_metrics'->name->>'max' AS max,
summary_metrics->'avg_metrics'->name->>'min' AS min,
summary_metrics->'avg_metrics'->name->>'sum' AS sum,
summary_metrics->'avg_metrics'->name->>'last' AS last,
summary_metrics->'avg_metrics'->name->>'count' AS count
FROM trials
CROSS JOIN jsonb_object_keys(summary_metrics->'avg_metrics') AS name
WHERE id = ?;`

	trainRows := []*summaryMetrics{}
	err := Bun().NewRaw(query, trialID).Scan(ctx, &trainRows)
	require.NoError(t, err)

	actualTrain := make(map[string]summaryMetrics)
	for _, v := range trainRows {
		name := v.Name
		v.Name = ""
		actualTrain[name] = *v
	}
	nanEqual(t, expectedTrain, actualTrain)

	valRows := []*summaryMetrics{}
	err = Bun().NewRaw(strings.ReplaceAll(query, "avg_metrics", "validation_metrics"), trialID).
		Scan(ctx, &valRows)
	require.NoError(t, err)

	actualVal := make(map[string]summaryMetrics)
	for _, v := range valRows {
		name := v.Name
		v.Name = ""
		actualVal[name] = *v
	}
	nanEqual(t, expectedVal, actualVal)
}

type summaryMetrics struct {
	Name  string
	Min   float64
	Max   float64
	Sum   float64
	Count int
	Last  any
}

func TestSummaryMetricsMigration(t *testing.T) {
	ctx := context.Background()

	require.NoError(t, etc.SetRootPath(RootFromDB))
	db := MustResolveTestPostgres(t)
	MustMigrateTestPostgres(t, db, MigrationsFromDB)
	user := RequireMockUser(t, db)

	exp := RequireMockExperiment(t, db, user)

	noMetrics := RequireMockTrial(t, db, exp)
	addMetrics(ctx, t, db, noMetrics, `[]`, `[]`)
	expectedNoMetrics := make(map[string]summaryMetrics)
	expectedNoValMetrics := make(map[string]summaryMetrics)

	numericMetrics := RequireMockTrial(t, db, exp)
	addMetrics(ctx, t, db, numericMetrics,
		`[{"a":1.0, "b":-0.5}, {"a":1.5,"b":0.0}, {"a":2.0}]`,
		`[{"val_loss": 1.5}]`,
	)
	expectedNumericMetrics := map[string]summaryMetrics{
		"a": {Min: 1.0, Max: 2.0, Sum: 1.0 + 1.5 + 2.0, Count: 3, Last: "2"},
		"b": {Min: -0.5, Max: 0.0, Sum: -0.5 + 0.0, Count: 2}, // empty last.
	}
	expectedNumericValMetrics := map[string]summaryMetrics{
		"val_loss": {Min: 1.5, Max: 1.5, Sum: 1.5, Count: 1, Last: "1.5"},
	}

	// Feels like we should report "val_loss"
	nonNumericMetrics := RequireMockTrial(t, db, exp)
	addMetrics(ctx, t, db, nonNumericMetrics,
		`[{"a":"a", "b":-0.5}, {"a":"b", "b":0.3, "c":"test"}, {"a":"c", "b":[{"loss":5.0}]}]`,
		`[{"val_loss": "c"}, {"val_gain": "d"}]`,
	)
	expectedNonNumericMetrics := map[string]summaryMetrics{
		"a": {Last: "c"},
		"b": {Last: `[{"loss": 5}]`},
		"c": {},
	}
	expectedNonNumericValMetrics := map[string]summaryMetrics{
		"val_loss": {},
		"val_gain": {Last: "d"},
	}

	infNaNMetrics := RequireMockTrial(t, db, exp)
	addMetrics(ctx, t, db, infNaNMetrics,
		`[{"a":"NaN", "b":"-Infinity"}, {"a":1.0, "b":"Infinity"}]`,
		`[{"a":1.0, "b":"Infinity"}, {"a":"NaN", "b":"-Infinity"}]`,
	)
	// Min is still 1.0 this is due to Postgres treating NaNs as greater than all other NaNs.
	// https://www.postgresql.org/docs/current/datatype-numeric.html
	expectedInfNaNMetrics := map[string]summaryMetrics{
		"a": {Min: 1.0, Max: math.NaN(), Sum: math.NaN(), Count: 2, Last: "1"},
		"b": {Min: math.Inf(-1), Max: math.Inf(+1), Sum: math.NaN(), Count: 2, Last: "Infinity"},
	}
	expectedInfNaNValMetrics := map[string]summaryMetrics{
		"a": {Min: 1.0, Max: math.NaN(), Sum: math.NaN(), Count: 2, Last: "NaN"},
		"b": {Min: math.Inf(-1), Max: math.Inf(+1), Sum: math.NaN(), Count: 2, Last: "-Infinity"},
	}

	runSummaryMigration(t)

	validateSummaryMetrics(ctx, t,
		numericMetrics.ID, expectedNumericMetrics, expectedNumericValMetrics)
	validateSummaryMetrics(ctx, t,
		nonNumericMetrics.ID, expectedNonNumericMetrics, expectedNonNumericValMetrics)
	validateSummaryMetrics(ctx, t,
		noMetrics.ID, expectedNoMetrics, expectedNoValMetrics)
	validateSummaryMetrics(ctx, t,
		infNaNMetrics.ID, expectedInfNaNMetrics, expectedInfNaNValMetrics)

	// Add a metric with an older endtime to ensure metric isn't computed.
	addSomeMetricInPast(ctx, t, noMetrics.ID)

	// Verify metric is recomputed with new metrics added.
	addMetrics(ctx, t, db, numericMetrics,
		`[{"b":-1.0}]`,
		`[{"val_loss": 3.0}]`,
	)
	expectedNumericMetrics = map[string]summaryMetrics{
		"a": {Min: 1.0, Max: 2.0, Sum: 1.0 + 1.5 + 2.0, Count: 3},
		"b": {Min: -1.0, Max: 0.0, Sum: -1.0 + -0.5 + 0.0, Count: 3, Last: "-1"},
	}
	expectedNumericValMetrics = map[string]summaryMetrics{
		"val_loss": {Min: 1.5, Max: 3.0, Sum: 1.5 + 3.0, Count: 2, Last: "3"},
	}

	runSummaryMigration(t)

	validateSummaryMetrics(ctx, t,
		numericMetrics.ID, expectedNumericMetrics, expectedNumericValMetrics)
	validateSummaryMetrics(ctx, t,
		nonNumericMetrics.ID, expectedNonNumericMetrics, expectedNonNumericValMetrics)
	validateSummaryMetrics(ctx, t,
		noMetrics.ID, expectedNoMetrics, expectedNoValMetrics)
	validateSummaryMetrics(ctx, t,
		infNaNMetrics.ID, expectedInfNaNMetrics, expectedInfNaNValMetrics)
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
			tr := RequireMockTrial(t, db, exp)
			allocation := RequireMockAllocation(t, db, tr.TaskID)
			trialIDs = append(trialIDs, tr.ID)

			for k := 0; k < 2; k++ {
				ckpt := uuid.New()
				checkpointIDs = append(checkpointIDs, ckpt)
				// Ensure it works with both checkpoint versions.
				if i == 0 && j == 0 && k == 0 {
					checkpointBun := struct {
						bun.BaseModel `bun:"table:checkpoints"`
						TrialID       int
						TrialRunID    int
						TotalBatches  int
						State         model.State
						UUID          string
						EndTime       time.Time
						Resources     map[string]int64
						Size          int64
					}{
						TrialID:      tr.ID,
						TrialRunID:   1,
						TotalBatches: 1,
						State:        model.ActiveState,
						UUID:         ckpt.String(),
						EndTime:      time.Now().UTC().Truncate(time.Millisecond),
						Resources:    resources[resourcesIndex],
						Size:         resources[resourcesIndex]["TEST"],
					}

					_, err := Bun().NewInsert().Model(&checkpointBun).Exec(ctx)
					require.NoError(t, err)
				} else {
					checkpoint := MockModelCheckpoint(ckpt, tr, allocation)
					checkpoint.Resources = resources[resourcesIndex]
					err := AddCheckpointMetadata(ctx, &checkpoint)
					require.NoError(t, err)
				}

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
	tr := RequireMockTrial(t, db, exp)
	allocation := RequireMockAllocation(t, db, tr.TaskID)

	// Create checkpoints
	ckpt1 := uuid.New()
	checkpoint1 := MockModelCheckpoint(ckpt1, tr, allocation)
	err := AddCheckpointMetadata(ctx, &checkpoint1)
	require.NoError(t, err)
	ckpt2 := uuid.New()
	checkpoint2 := MockModelCheckpoint(ckpt2, tr, allocation)
	err = AddCheckpointMetadata(ctx, &checkpoint2)
	require.NoError(t, err)
	ckpt3 := uuid.New()
	checkpoint3 := MockModelCheckpoint(ckpt3, tr, allocation)
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
		tr := RequireMockTrial(t, db, exp)
		allocation := RequireMockAllocation(t, db, tr.TaskID)
		for k := 0; k < 10; k++ {
			ckpt := uuid.New()
			checkpoints = append(checkpoints, ckpt)

			resources := make(map[string]int64)
			for r := 0; r < 100000; r++ {
				resources[uuid.New().String()] = rand.Int63n(2500) //nolint: gosec
			}

			checkpoint := MockModelCheckpoint(ckpt, tr, allocation)
			checkpoint.Resources = resources

			err := AddCheckpointMetadata(ctx, &checkpoint)
			require.NoError(t, err)
		}
	}

	require.NoError(t, MarkCheckpointsDeleted(ctx, checkpoints))
}
