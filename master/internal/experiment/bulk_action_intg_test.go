//go:build integration
// +build integration

package experiment

import (
	"context"
	"testing"

	"github.com/determined-ai/determined/master/internal/db"
	"github.com/determined-ai/determined/proto/pkg/apiv1"
	"github.com/google/uuid"
	"github.com/stretchr/testify/require"
)

func TestActivateExperiments(t *testing.T) {
	type args struct {
		projectID     int32
		experimentIds []int32
		filters       *apiv1.BulkExperimentFilters
	}

	ctx := context.Background()

	user := db.RequireMockUser(t, db.SingleDB())
	exp := db.RequireMockExperiment(t, db.SingleDB(), user)
	tr, task := db.RequireMockTrial(t, db.SingleDB(), exp)
	a := db.RequireMockAllocation(t, db.SingleDB(), task.TaskID)
	length := 4
	var expectedResults []ExperimentActionResult
	for i := 1; i <= length; i++ {
		ckptUUID := uuid.New()
		ckpt := db.MockModelCheckpoint(ckptUUID, a, db.WithSteps(i))
		err := db.AddCheckpointMetadata(ctx, &ckpt, tr.ID)
		require.NoError(t, err)
		err = db.AddTrialValidationMetrics(ctx, ckptUUID, tr, int32(i), int32(i+5), db.SingleDB())
		require.NoError(t, err)

		expectedResults = append(expectedResults, ExperimentActionResult{
			ID: int32(i),
		})
	}

	tests := []struct {
		name        string
		fields      db.PgDB
		args        args
		expected    []ExperimentActionResult
		expectedErr bool
	}{
		{"test-000", *db.SingleDB(), args{}, expectedResults, false},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			actual, err := ActivateExperiments(ctx, tt.args.projectID, tt.args.experimentIds,
				tt.args.filters)
			if (err != nil) != tt.expectedErr {
				t.Errorf("ActivateExperiments() error = %v, expectedErr %v", err, tt.expectedErr)
				return
			}
			require.ElementsMatch(t, actual, tt.expected)
		})
	}
}

func TestCancelExperiments(t *testing.T) {
	type args struct {
		projectID     int32
		experimentIds []int32
		filters       *apiv1.BulkExperimentFilters
	}

	ctx := context.Background()

	user := db.RequireMockUser(t, db.SingleDB())
	exp := db.RequireMockExperiment(t, db.SingleDB(), user)
	tr, task := db.RequireMockTrial(t, db.SingleDB(), exp)
	a := db.RequireMockAllocation(t, db.SingleDB(), task.TaskID)
	length := 4
	var expectedResults []ExperimentActionResult
	for i := 1; i <= length; i++ {
		ckptUUID := uuid.New()
		ckpt := db.MockModelCheckpoint(ckptUUID, a, db.WithSteps(i))
		err := db.AddCheckpointMetadata(ctx, &ckpt, tr.ID)
		require.NoError(t, err)
		err = db.AddTrialValidationMetrics(ctx, ckptUUID, tr, int32(i), int32(i+5), db.SingleDB())
		require.NoError(t, err)

		expectedResults = append(expectedResults, ExperimentActionResult{
			ID: int32(i),
		})
	}

	tests := []struct {
		name        string
		fields      db.PgDB
		args        args
		expected    []ExperimentActionResult
		expectedErr bool
	}{
		{"test-000", *db.SingleDB(), args{}, expectedResults, false},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			actual, err := CancelExperiments(ctx, tt.args.projectID, tt.args.experimentIds,
				tt.args.filters)
			if (err != nil) != tt.expectedErr {
				t.Errorf("CancelExperiments() error = %v, expectedErr %v", err, tt.expectedErr)
				return
			}
			require.ElementsMatch(t, actual, tt.expected)
		})
	}
}

func TestKillExperiments(t *testing.T) {
	type args struct {
		projectID     int32
		experimentIds []int32
		filters       *apiv1.BulkExperimentFilters
	}

	ctx := context.Background()

	user := db.RequireMockUser(t, db.SingleDB())
	exp := db.RequireMockExperiment(t, db.SingleDB(), user)
	tr, task := db.RequireMockTrial(t, db.SingleDB(), exp)
	a := db.RequireMockAllocation(t, db.SingleDB(), task.TaskID)
	length := 4
	var expectedResults []ExperimentActionResult
	for i := 1; i <= length; i++ {
		ckptUUID := uuid.New()
		ckpt := db.MockModelCheckpoint(ckptUUID, a, db.WithSteps(i))
		err := db.AddCheckpointMetadata(ctx, &ckpt, tr.ID)
		require.NoError(t, err)
		err = db.AddTrialValidationMetrics(ctx, ckptUUID, tr, int32(i), int32(i+5), db.SingleDB())
		require.NoError(t, err)

		expectedResults = append(expectedResults, ExperimentActionResult{
			ID: int32(i),
		})
	}

	tests := []struct {
		name        string
		fields      db.PgDB
		args        args
		expected    []ExperimentActionResult
		expectedErr bool
	}{
		{"test-000", *db.SingleDB(), args{}, expectedResults, false},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			actual, err := KillExperiments(ctx, tt.args.projectID, tt.args.experimentIds,
				tt.args.filters)
			if (err != nil) != tt.expectedErr {
				t.Errorf("KillExperiments() error = %v, expectedErr %v", err, tt.expectedErr)
				return
			}
			require.ElementsMatch(t, actual, tt.expected)
		})
	}
}
