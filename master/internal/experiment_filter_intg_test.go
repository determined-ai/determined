//go:build integration
// +build integration

package internal

import (
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/stretchr/testify/require"
	"google.golang.org/protobuf/types/known/timestamppb"

	"github.com/determined-ai/determined/master/internal/db"
	"github.com/determined-ai/determined/master/pkg/model"
	"github.com/determined-ai/determined/master/pkg/ptrs"
	"github.com/determined-ai/determined/master/pkg/schemas"
	"github.com/determined-ai/determined/master/pkg/schemas/expconf"
	"github.com/determined-ai/determined/proto/pkg/apiv1"
)

func TestHpNotContainsToSql(t *testing.T) {
	api, _, ctx := setupAPITest(t, nil)

	workResp, err := api.PostWorkspace(ctx, &apiv1.PostWorkspaceRequest{
		Name: uuid.New().String(),
	})
	require.NoError(t, err)
	projResp, err := api.PostProject(ctx, &apiv1.PostProjectRequest{
		WorkspaceId: workResp.Workspace.Id,
		Name:        uuid.New().String(),
	})
	require.NoError(t, err)
	pid := projResp.Project.Id

	// nolint: exhaustruct
	activeConfig0 := schemas.WithDefaults(schemas.Merge(minExpConfig, expconf.ExperimentConfig{
		RawHyperparameters: expconf.Hyperparameters{
			"hyperparameter": expconf.HyperparameterV0{
				RawConstHyperparameter: &expconf.ConstHyperparameterV0{
					RawVal: "foo",
				},
			},
		},
	}))
	job0ID := uuid.New().String()

	startTime := time.Unix(123123123, int64(1329012309*time.Nanosecond))
	endTime := time.Unix(423123123, int64(999813239*time.Nanosecond))
	require.WithinDuration(t,
		endTime, timestamppb.New(endTime).AsTime(), time.Millisecond)

	exp0 := &model.Experiment{
		StartTime: startTime,
		EndTime:   &endTime,
		JobID:     model.JobID(job0ID),
		Archived:  false,
		State:     model.PausedState,
		Notes:     "notes",
		Config:    activeConfig0.AsLegacy(),
		OwnerID:   ptrs.Ptr(model.UserID(1)),
		ProjectID: int(pid),
	}
	require.NoError(t, api.m.db.AddExperiment(exp0, []byte{1, 2, 3}, activeConfig0))

	var fieldOperator operator = "notContains"
	var conjunction filterConjunction = "and"
	location := "LOCATION_TYPE_HYPERPARAMETERS"
	fieldType := "COLUMN_TYPE_TEXT"
	var fieldValue interface{} = "fo"

	filter := experimentFilterRoot{
		FilterGroup: experimentFilter{
			Children: []*experimentFilter{{
				ColumnName: "hp.hyperparameter",
				Kind:       "field",
				Location:   &location,
				Operator:   &fieldOperator,
				Type:       &fieldType,
				Value:      &fieldValue,
			}},
			Conjunction: &conjunction,
			Kind:        "group",
		},
		ShowArchived: true,
	}
	query := db.Bun().NewSelect().
		TableExpr("experiments as e")

	_, queryErr := filter.toSQL(query)
	require.Nil(t, queryErr)

	exists, err := query.Exists(ctx)
	require.Nil(t, err)
	require.False(t, exists)
}
