//go:build integration
// +build integration

package internal

import (
	"encoding/json"
	"testing"
	"time"

	"github.com/golang/protobuf/ptypes/wrappers"
	"github.com/google/uuid"
	"github.com/stretchr/testify/require"
	"google.golang.org/protobuf/types/known/structpb"

	"github.com/determined-ai/determined/master/internal/db"
	"github.com/determined-ai/determined/master/pkg/model"
	"github.com/determined-ai/determined/master/pkg/schemas"
	"github.com/determined-ai/determined/proto/pkg/apiv1"
	"github.com/determined-ai/determined/proto/pkg/apiv2"
	"github.com/determined-ai/determined/proto/pkg/experimentv1"
	"github.com/determined-ai/determined/proto/pkg/searchv2"
)

func TestGetSearchConfig(t *testing.T) {
	api, curUser, ctx := setupAPITest(t, nil)

	exp := createTestExp(t, api, curUser)
	expectedBytes, err := db.SingleDB().ExperimentConfigRaw(exp.ID)
	require.NoError(t, err)
	expected := make(map[string]any)
	require.NoError(t, json.Unmarshal(expectedBytes, &expected))

	resp, err := api.GetSearch(ctx, &apiv2.GetSearchRequest{
		SearchId: int32(exp.ID),
	})
	require.NoError(t, err)

	cases := []struct {
		name   string
		config *structpb.Struct
	}{
		{"GetSearchResponse.Config", resp.Config},
		{"GetSearchResponse.Search.Config", resp.Search.Config}, //nolint:staticcheck
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			require.Equal(t, expected, c.config.AsMap())
		})
	}
}

func TestGetPutDeleteSearchTags(t *testing.T) {
	api, curUser, ctx := setupAPITest(t, nil)
	_, projectIDInt := createProjectAndWorkspace(ctx, t, api)
	projectID := int32(projectIDInt)

	activeConfig := schemas.WithDefaults(minExpConfig)
	exp := &model.Experiment{
		JobID:     model.JobID(uuid.New().String()),
		State:     model.CompletedState,
		OwnerID:   &curUser.ID,
		ProjectID: projectIDInt,
		StartTime: time.Now(),
		Config:    activeConfig.AsLegacy(),
	}
	require.NoError(t, api.m.db.AddExperiment(exp, []byte{10, 11, 12}, activeConfig))

	// Runs are required for delete tags to work
	task := &model.Task{TaskType: model.TaskTypeTrial, TaskID: model.NewTaskID()}
	require.NoError(t, db.AddTask(ctx, task))
	require.NoError(t, db.AddTrial(ctx, &model.Trial{
		State:        model.PausedState,
		ExperimentID: exp.ID,
		StartTime:    time.Now(),
	}, task.TaskID))

	// No tags initially
	getResp, err := api.GetSearchTags(ctx, &apiv2.GetSearchTagsRequest{
		ProjectId: projectID,
	})
	require.NoError(t, err)
	require.Empty(t, getResp.Tags)

	// Put new tag
	testTag := "testTag"
	putResp, err := api.PutSearchTag(ctx, &apiv2.PutSearchTagRequest{
		SearchId: int32(exp.ID),
		Tag:      testTag,
	})
	require.NoError(t, err)
	require.Len(t, putResp.Tags, 1)
	require.Equal(t, putResp.Tags[0], testTag)

	// Tags should be present
	getResp, err = api.GetSearchTags(ctx, &apiv2.GetSearchTagsRequest{
		ProjectId: projectID,
	})
	require.NoError(t, err)
	require.Len(t, getResp.Tags, 1)

	// Delete tag
	deleteResp, err := api.DeleteSearchTag(ctx, &apiv2.DeleteSearchTagRequest{
		SearchId: int32(exp.ID),
		Tag:      testTag,
	})
	require.NoError(t, err)
	require.Empty(t, deleteResp.Tags)

	// No more tags in project
	getResp, err = api.GetSearchTags(ctx, &apiv2.GetSearchTagsRequest{
		ProjectId: projectID,
	})
	require.NoError(t, err)
	require.Empty(t, getResp.Tags)
}

func TestBulkFilterConversion(t *testing.T) {
	cases := []struct {
		name     string
		input    *apiv2.BulkSearchFilters
		expected *apiv1.BulkExperimentFilters
	}{
		{
			name: "Full Object",
			input: &apiv2.BulkSearchFilters{
				Description: "desc",
				Name:        "name",
				Labels:      []string{"a", "b"},
				Archived: &wrappers.BoolValue{
					Value: false,
				},
				States: []searchv2.State{
					searchv2.State_STATE_ACTIVE,
					searchv2.State_STATE_COMPLETED,
				},
				UserIds:           []int32{1, 2, 3},
				ProjectId:         1,
				ExcludedSearchIds: []int32{8, 9, 10},
			},
			expected: &apiv1.BulkExperimentFilters{
				Description: "desc",
				Name:        "name",
				Labels:      []string{"a", "b"},
				Archived: &wrappers.BoolValue{
					Value: false,
				},
				States: []experimentv1.State{
					experimentv1.State_STATE_ACTIVE,
					experimentv1.State_STATE_COMPLETED,
				},
				UserIds:               []int32{1, 2, 3},
				ProjectId:             1,
				ExcludedExperimentIds: []int32{8, 9, 10},
			},
		},
		{
			name: "Partial Object",
			input: &apiv2.BulkSearchFilters{
				Description: "desc",
				Name:        "name",
				States: []searchv2.State{
					searchv2.State_STATE_ACTIVE,
					searchv2.State_STATE_COMPLETED,
				},
			},
			expected: &apiv1.BulkExperimentFilters{
				Description: "desc",
				Name:        "name",
				States: []experimentv1.State{
					experimentv1.State_STATE_ACTIVE,
					experimentv1.State_STATE_COMPLETED,
				},
			},
		},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			result := bulkFiltersSearchToExperiment(c.input)
			require.Equal(t, c.expected, result)
		})
	}
}
