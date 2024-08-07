//go:build integration
// +build integration

package experiment

import (
	"context"
	"testing"

	"github.com/google/uuid"
	"github.com/stretchr/testify/require"
	"google.golang.org/protobuf/types/known/wrapperspb"

	"github.com/determined-ai/determined/master/internal/db"
	"github.com/determined-ai/determined/master/pkg/model"
	"github.com/determined-ai/determined/proto/pkg/apiv1"
	"github.com/determined-ai/determined/proto/pkg/experimentv1"
)

func TestGetExperimentsEditableByUser(t *testing.T) {
	nameExt := uuid.New()
	type args struct {
		projectID     int32
		experimentIDs []int32
		filters       *apiv1.BulkExperimentFilters
	}

	ctx := context.Background()

	testUser := db.RequireMockUser(t, db.SingleDB())
	testWorkspaceID, _ := db.RequireMockWorkspaceID(t, db.SingleDB(), "TestGetExperimentsEditableByUser-"+nameExt.String())
	testProjectID, _ := db.RequireMockProjectID(t, db.SingleDB(), testWorkspaceID, false)

	testModelStates := []model.State{
		model.ActiveState,
		model.CanceledState,
		model.CompletedState,
		model.ErrorState,
		model.PausedState,
		model.StoppingCanceledState,
		model.DeletingState,
		model.RunningState,
	}

	allExperimentIds := make([]int, len(testModelStates))
	editableExperiments := map[model.State]*model.Experiment{}
	for i, state := range testModelStates {
		exp := db.RequireMockExperimentParams(
			t, db.SingleDB(), testUser,
			db.MockExperimentParams{
				State: &state,
			},
			testProjectID,
		)
		editableExperiments[state] = exp
		allExperimentIds[i] = exp.ID
	}

	defer func(ids []int) {
		_ = db.SingleDB().DeleteExperiments(ctx, ids)
	}(allExperimentIds)

	tests := []struct {
		name        string
		args        args
		expected    []int32
		expectedErr bool
	}{
		{
			name: "no filters",
			args: args{
				projectID: int32(testProjectID),
				experimentIDs: []int32{
					int32(editableExperiments[model.ActiveState].ID),
					int32(editableExperiments[model.CanceledState].ID),
					int32(editableExperiments[model.CompletedState].ID),
					int32(editableExperiments[model.ErrorState].ID),
					int32(editableExperiments[model.PausedState].ID),
					int32(editableExperiments[model.StoppingCanceledState].ID),
					int32(editableExperiments[model.DeletingState].ID),
					int32(editableExperiments[model.RunningState].ID),
				},
			},
			expected: []int32{
				int32(editableExperiments[model.ActiveState].ID),
				int32(editableExperiments[model.CanceledState].ID),
				int32(editableExperiments[model.CompletedState].ID),
				int32(editableExperiments[model.ErrorState].ID),
				int32(editableExperiments[model.PausedState].ID),
				int32(editableExperiments[model.StoppingCanceledState].ID),
				int32(editableExperiments[model.DeletingState].ID),
				int32(editableExperiments[model.RunningState].ID),
			},
		},
		{
			name: "no results",
			args: args{
				projectID: int32(testProjectID),
				filters: &apiv1.BulkExperimentFilters{
					Description: "no-match",
					Name:        "no-match",
					Labels:      []string{"no-match"},
					Archived: &wrapperspb.BoolValue{
						Value: true,
					},
				},
			},
			expected: []int32{},
		},
		{
			name: "exclude finished",
			args: args{
				projectID: int32(testProjectID),
				filters: &apiv1.BulkExperimentFilters{
					ExcludedExperimentIds: []int32{
						int32(editableExperiments[model.CanceledState].ID),
						int32(editableExperiments[model.CompletedState].ID),
						int32(editableExperiments[model.ErrorState].ID),
					},
				},
			},
			expected: []int32{
				int32(editableExperiments[model.ActiveState].ID),
				int32(editableExperiments[model.PausedState].ID),
				int32(editableExperiments[model.StoppingCanceledState].ID),
				int32(editableExperiments[model.DeletingState].ID),
				int32(editableExperiments[model.RunningState].ID),
			},
		},
		{
			name: "include only finished",
			args: args{
				projectID: int32(testProjectID),
				filters: &apiv1.BulkExperimentFilters{
					States: []experimentv1.State{
						experimentv1.State_STATE_CANCELED,
						experimentv1.State_STATE_COMPLETED,
						experimentv1.State_STATE_ERROR,
					},
				},
			},
			expected: []int32{
				int32(editableExperiments[model.CanceledState].ID),
				int32(editableExperiments[model.CompletedState].ID),
				int32(editableExperiments[model.ErrorState].ID),
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			actual, err := getExperimentsEditableByUser(
				ctx, &testUser, tt.args.projectID, tt.args.experimentIDs, tt.args.filters,
			)
			if (err != nil) != tt.expectedErr {
				t.Errorf("getExperimentsEditableByUser() error = %v, expectedErr %v", err, tt.expectedErr)
				return
			}
			require.ElementsMatch(t, tt.expected, actual)
		})
	}
}
