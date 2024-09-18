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
	testProjectID2, _ := db.RequireMockProjectID(t, db.SingleDB(), testWorkspaceID, false)

	statePtr := func(s model.State) *model.State { return &s }
	testModels := map[string]db.MockExperimentParams{
		"project_1_state_active": {
			ProjectID: &testProjectID,
			State:     statePtr(model.ActiveState),
		},
		"project_1_state_canceled": {
			ProjectID: &testProjectID,
			State:     statePtr(model.CanceledState),
		},
		"project_1_state_completed": {
			ProjectID: &testProjectID,
			State:     statePtr(model.CompletedState),
		},
		"project_1_state_error": {
			ProjectID: &testProjectID,
			State:     statePtr(model.ErrorState),
		},
		"project_1_state_paused": {
			ProjectID: &testProjectID,
			State:     statePtr(model.PausedState),
		},
		"project_1_state_stoppingCanceled": {
			ProjectID: &testProjectID,
			State:     statePtr(model.StoppingCanceledState),
		},
		"project_1_state_deleting": {
			ProjectID: &testProjectID,
			State:     statePtr(model.DeletingState),
		},
		"project_1_state_running": {
			ProjectID: &testProjectID,
			State:     statePtr(model.RunningState),
		},
		"project_2_state_active": {
			ProjectID: &testProjectID2,
			State:     statePtr(model.ActiveState),
		},
		"project_2_state_paused": {
			ProjectID: &testProjectID2,
			State:     statePtr(model.PausedState),
		},
	}

	allExperimentIds := make([]int, len(testModels))
	editableExperiments := map[string]*model.Experiment{}
	i := 0
	for name, model := range testModels {
		exp := db.RequireMockExperimentParams(
			t, db.SingleDB(), testUser,
			model,
			*model.ProjectID,
		)
		editableExperiments[name] = exp
		allExperimentIds[i] = exp.ID
		i++
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
					int32(editableExperiments["project_1_state_active"].ID),
					int32(editableExperiments["project_1_state_canceled"].ID),
					int32(editableExperiments["project_1_state_completed"].ID),
					int32(editableExperiments["project_1_state_error"].ID),
					int32(editableExperiments["project_1_state_paused"].ID),
					int32(editableExperiments["project_1_state_stoppingCanceled"].ID),
					int32(editableExperiments["project_1_state_deleting"].ID),
					int32(editableExperiments["project_1_state_running"].ID),
				},
			},
			expected: []int32{
				int32(editableExperiments["project_1_state_active"].ID),
				int32(editableExperiments["project_1_state_canceled"].ID),
				int32(editableExperiments["project_1_state_completed"].ID),
				int32(editableExperiments["project_1_state_error"].ID),
				int32(editableExperiments["project_1_state_paused"].ID),
				int32(editableExperiments["project_1_state_stoppingCanceled"].ID),
				int32(editableExperiments["project_1_state_deleting"].ID),
				int32(editableExperiments["project_1_state_running"].ID),
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
			name: "exclude finished by ID",
			args: args{
				projectID: int32(testProjectID),
				filters: &apiv1.BulkExperimentFilters{
					ExcludedExperimentIds: []int32{
						int32(editableExperiments["project_1_state_canceled"].ID),
						int32(editableExperiments["project_1_state_completed"].ID),
						int32(editableExperiments["project_1_state_error"].ID),
					},
				},
			},
			expected: []int32{
				int32(editableExperiments["project_1_state_active"].ID),
				int32(editableExperiments["project_1_state_paused"].ID),
				int32(editableExperiments["project_1_state_stoppingCanceled"].ID),
				int32(editableExperiments["project_1_state_deleting"].ID),
				int32(editableExperiments["project_1_state_running"].ID),
			},
		},
		{
			name: "include only finished by state",
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
				int32(editableExperiments["project_1_state_canceled"].ID),
				int32(editableExperiments["project_1_state_completed"].ID),
				int32(editableExperiments["project_1_state_error"].ID),
			},
		},
		{
			name: "active in project 2",
			args: args{
				projectID: int32(testProjectID2),
				filters: &apiv1.BulkExperimentFilters{
					ProjectId: int32(testProjectID2),
					States: []experimentv1.State{
						experimentv1.State_STATE_ACTIVE,
					},
				},
			},
			expected: []int32{
				int32(editableExperiments["project_2_state_active"].ID),
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

func TestIntegrationKillAllExperimentsInProject(t *testing.T) {
	ctx := context.Background()
	nameExt := uuid.New()
	testUser := db.RequireMockUser(t, db.SingleDB())

	// override experimentsEditableByUser to bypass dependency on context with attached auth
	defer func(originalFunc func(context.Context, int32, []int32, *apiv1.BulkExperimentFilters) ([]int32, error)) {
		experimentsEditableByUser = originalFunc
	}(experimentsEditableByUser)
	experimentsEditableByUser = func(
		ctx context.Context,
		projectID int32,
		experimentIDs []int32,
		filters *apiv1.BulkExperimentFilters,
	) ([]int32, error) {
		exps, err := getExperimentsEditableByUser(
			ctx, &testUser, projectID, experimentIDs, filters,
		)
		return exps, err
	}

	testWorkspaceID, _ := db.RequireMockWorkspaceID(t, db.SingleDB(), "TestIntegrationKillExperiment-"+nameExt.String())
	testProjectID, _ := db.RequireMockProjectID(t, db.SingleDB(), testWorkspaceID, false)
	otherProjectID, _ := db.RequireMockProjectID(t, db.SingleDB(), testWorkspaceID, false)

	statePtr := func(s model.State) *model.State { return &s }
	testModels := map[string]db.MockExperimentParams{
		// already in terminal state, will not be affected
		"project_1_state_completed": {
			ProjectID: &testProjectID,
			State:     statePtr(model.CompletedState),
		},
		// expect this to be killed
		"project_1_state_active": {
			ProjectID: &testProjectID,
			State:     statePtr(model.ActiveState),
		},
		// in a different project, will not be affected
		"project_2_state_active": {
			ProjectID: &otherProjectID,
			State:     statePtr(model.ActiveState),
		},
	}

	allExperimentIds := make([]int, 0, len(testModels))
	editableExperiments := map[string]*model.Experiment{}
	for name, model := range testModels {
		exp := db.RequireMockExperimentParams(
			t, db.SingleDB(), testUser,
			model,
			*model.ProjectID,
		)
		editableExperiments[name] = exp
		allExperimentIds = append(allExperimentIds, exp.ID)
	}
	defer func(ids []int) {
		_ = db.SingleDB().DeleteExperiments(ctx, ids)
	}(allExperimentIds)

	for _, expID := range allExperimentIds {
		exp := experimentMock{}
		if expID == editableExperiments["project_1_state_active"].ID {
			exp.On("KillExperiment").Return(nil)
		}
		require.NoError(t, ExperimentRegistry.Add(expID, &exp))
		defer exp.AssertExpectations(t)
		defer ExperimentRegistry.Delete(expID) //nolint:errcheck
	}

	filters := &apiv1.BulkExperimentFilters{
		ProjectId: int32(testProjectID),
	}

	killed, err := KillExperiments(ctx, int32(testProjectID), []int32{}, filters)
	require.NoError(t, err)
	require.ElementsMatch(t, killed, []ExperimentActionResult{
		{
			ID: int32(editableExperiments["project_1_state_active"].ID),
		},
	})
}
