//go:build integration
// +build integration

package experiment

import (
	"context"
	"fmt"
	"log"
	"os"
	"reflect"
	"sort"
	"strings"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/stretchr/testify/require"

	"github.com/determined-ai/determined/master/internal/db"
	"github.com/determined-ai/determined/master/pkg/etc"
	"github.com/determined-ai/determined/master/pkg/model"
	"github.com/determined-ai/determined/proto/pkg/modelv1"
)

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

func TestPgDB_ExperimentCheckpointsToGCRawModelRegistry(t *testing.T) {
	type args struct {
		id             int
		experimentBest int
		trialBest      int
		trialLatest    int
	}

	ctx := context.Background()

	user := db.RequireMockUser(t, db.SingleDB())
	exp := db.RequireMockExperiment(t, db.SingleDB(), user)
	tr, task := db.RequireMockTrial(t, db.SingleDB(), exp)
	a := db.RequireMockAllocation(t, db.SingleDB(), task.TaskID)
	length := 4
	var expectedCheckpoints []uuid.UUID
	for i := 1; i <= length; i++ {
		ckptUUID := uuid.New()
		ckpt := db.MockModelCheckpoint(ckptUUID, a, db.WithSteps(i))
		err := db.AddCheckpointMetadata(ctx, &ckpt, tr.ID)
		require.NoError(t, err)
		err = db.AddTrialValidationMetrics(ctx, ckptUUID, tr, int32(i), int32(i+5), db.SingleDB())
		require.NoError(t, err)

		if i == 2 { // add this checkpoint to the model registry
			err = addCheckpointToModelRegistry(ctx, ckptUUID, user)
			require.NoError(t, err)
		} else {
			expectedCheckpoints = append(expectedCheckpoints, ckptUUID)
		}
	}

	tests := []struct {
		name        string
		fields      db.PgDB
		args        args
		expected    []uuid.UUID
		expectedErr bool
	}{
		{"test-000", *db.SingleDB(), args{exp.ID, 0, 0, 0}, expectedCheckpoints, false},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			actual, err := ExperimentCheckpointsToGCRaw(ctx, tt.args.id, tt.args.experimentBest,
				tt.args.trialBest, tt.args.trialLatest)
			if (err != nil) != tt.expectedErr {
				t.Errorf("db.SingleDB().ExperimentCheckpointsToGCRaw() error = %v, expectedErr %v", err, tt.expectedErr)
				return
			}
			sort.Slice(actual, func(i, j int) bool {
				return actual[i].String() < actual[j].String()
			})
			sort.Slice(tt.expected, func(i, j int) bool {
				return tt.expected[i].String() < tt.expected[j].String()
			})
			if !reflect.DeepEqual(actual, tt.expected) {
				t.Errorf("%v db.SingleDB().ExperimentCheckpointsToGCRaw() = %v, expected %v", tt.args.id, actual, tt.expected)
			}
		})
	}
}

func TestPgDB_ExperimentCheckpointsToGCRaw(t *testing.T) {
	type args struct {
		id             int
		experimentBest int
		trialBest      int
		trialLatest    int
	}

	ctx := context.Background()

	user := db.RequireMockUser(t, db.SingleDB())
	exp := db.RequireMockExperiment(t, db.SingleDB(), user)
	tr, task := db.RequireMockTrial(t, db.SingleDB(), exp)
	a := db.RequireMockAllocation(t, db.SingleDB(), task.TaskID)
	length := 4
	allCheckpoints := make([]uuid.UUID, length)
	for i := 1; i <= length; i++ {
		ckptUUID := uuid.New()
		ckpt := db.MockModelCheckpoint(ckptUUID, a, db.WithSteps(i))
		err := db.AddCheckpointMetadata(ctx, &ckpt, tr.ID)
		require.NoError(t, err)
		err = db.AddTrialValidationMetrics(ctx, ckptUUID, tr, int32(i), int32(i+5), db.SingleDB())
		require.NoError(t, err)
		allCheckpoints[i-1] = ckptUUID
	}

	allCheckpointsExpFirst := append([]uuid.UUID(nil), allCheckpoints[1:]...)
	allCheckpointsExpLast := append([]uuid.UUID(nil), allCheckpoints[:length-1]...)
	allCheckpointsExpFirstLast := append([]uuid.UUID(nil), allCheckpoints[1:length-1]...)

	tests := []struct {
		name        string
		fields      db.PgDB
		args        args
		expected    []uuid.UUID
		expectedErr bool
	}{
		{"test-000", *db.SingleDB(), args{exp.ID, 0, 0, 0}, allCheckpoints, false},
		{"test-001", *db.SingleDB(), args{exp.ID, 0, 0, 1}, allCheckpointsExpLast, false},
		{"test-010", *db.SingleDB(), args{exp.ID, 0, 1, 0}, allCheckpointsExpFirst, false},
		{"test-011", *db.SingleDB(), args{exp.ID, 0, 1, 1}, allCheckpointsExpFirstLast, false},
		{"test-100", *db.SingleDB(), args{exp.ID, 1, 0, 0}, allCheckpointsExpFirst, false},
		{"test-101", *db.SingleDB(), args{exp.ID, 1, 0, 1}, allCheckpointsExpFirstLast, false},
		{"test-110", *db.SingleDB(), args{exp.ID, 1, 1, 0}, allCheckpointsExpFirst, false},
		{"test-111", *db.SingleDB(), args{exp.ID, 1, 1, 1}, allCheckpointsExpFirstLast, false},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			actual, err := ExperimentCheckpointsToGCRaw(ctx, tt.args.id,
				tt.args.experimentBest, tt.args.trialBest, tt.args.trialLatest)
			if (err != nil) != tt.expectedErr {
				t.Errorf("db.SingleDB().ExperimentCheckpointsToGCRaw() error = %v, expectedErr %v", err, tt.expectedErr)
				return
			}
			sort.Slice(actual, func(i, j int) bool {
				return actual[i].String() < actual[j].String()
			})
			sort.Slice(tt.expected, func(i, j int) bool {
				return tt.expected[i].String() < tt.expected[j].String()
			})
			if !reflect.DeepEqual(actual, tt.expected) {
				t.Errorf("%v db.SingleDB().ExperimentCheckpointsToGCRaw() = %v, expected %v", tt.args.id, actual, tt.expected)
			}
		})
	}
}

func addCheckpointToModelRegistry(ctx context.Context, checkpointUUID uuid.UUID, user model.User) error {
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
	if err != nil {
		return fmt.Errorf("inserting a model: %w", err)
	}

	// Register checkpoints
	retCkpt1, err := db.GetCheckpoint(ctx, checkpointUUID.String())
	if err != nil {
		return fmt.Errorf("getting checkpoint: %w", err)
	}

	addmv := modelv1.ModelVersion{
		Model:      pmdl,
		Checkpoint: retCkpt1,
		Name:       "checkpoint exp",
		Comment:    "empty",
	}
	log.Print(user.ID)
	_, err = db.InsertModelVersion(ctx, pmdl.Id, retCkpt1.Uuid, addmv.Name, addmv.Comment,
		emptyMetadata, strings.Join(addmv.Labels, ","), addmv.Notes, user.ID,
	)
	if err != nil {
		return fmt.Errorf("inserting model version: %w", err)
	}

	return nil
}
