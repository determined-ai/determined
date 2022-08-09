//go:build integration
// +build integration

package db

import (
	"context"
	"crypto/rand"
	"encoding/json"
	"fmt"
	"testing"
	"time"

	"github.com/davecgh/go-spew/spew"
	"github.com/google/uuid"
	"github.com/stretchr/testify/require"
	"google.golang.org/protobuf/types/known/structpb"

	"github.com/determined-ai/determined/master/pkg/model"
	"github.com/determined-ai/determined/master/pkg/ptrs"
	"github.com/determined-ai/determined/proto/pkg/checkpointv1"
	"github.com/determined-ai/determined/proto/pkg/modelv1"
	"github.com/determined-ai/determined/proto/pkg/trialv1"
)

func pprintedExpect(expected, got interface{}) string {
	return fmt.Sprintf("expected \n\t%s\ngot\n\t%s", spew.Sdump(expected), spew.Sdump(got))
}

func requireMockTrial(t *testing.T, db *PgDB, exp *model.Experiment) *model.Trial {
	task := RequireMockTask(t, db, exp.OwnerID)
	rqID := model.NewRequestID(rand.Reader)
	tr := model.Trial{
		TaskID:       task.TaskID,
		RequestID:    &rqID,
		ExperimentID: exp.ID,
		State:        model.ActiveState,
		StartTime:    time.Now(),
		HParams:      model.JSONObj{"global_batch_size": 1},
		JobID:        exp.JobID,
	}
	err := db.AddTrial(&tr)
	require.NoError(t, err, "failed to add trial")
	return &tr
}

func requireMockAllocation(t *testing.T, db *PgDB, tID model.TaskID) *model.Allocation {
	a := model.Allocation{
		AllocationID: model.AllocationID(fmt.Sprintf("%s-1", tID)),
		TaskID:       tID,
		StartTime:    ptrs.Ptr(time.Now().UTC()),
		State:        ptrs.Ptr(model.AllocationStateTerminated),
	}
	err := db.AddAllocation(&a)
	require.NoError(t, err, "failed to add allocation")
	return &a
}

func requireMockModel(t *testing.T, db *PgDB, user model.User) *modelv1.Model {
	now := time.Now()
	m := model.Model{
		Name:            uuid.NewString(),
		Description:     "",
		Metadata:        map[string]interface{}{},
		CreationTime:    now,
		LastUpdatedTime: now,
		Labels:          []string{},
		Username:        user.Username,
	}
	b, err := json.Marshal(m.Metadata)
	require.NoError(t, err)

	var pmdl modelv1.Model
	err = db.QueryProto("insert_model", &pmdl, m.Name, m.Description, b, m.Labels, "", user.ID)
	require.NoError(t, err)
	return &pmdl
}

func requireMockMetrics(
	t *testing.T, db *PgDB, tr *model.Trial, stepsCompleted int, metricValue float64,
) *trialv1.TrialMetrics {
	m := trialv1.TrialMetrics{
		TrialId:        int32(tr.ID),
		StepsCompleted: int32(stepsCompleted),
		Metrics: &checkpointv1.Metrics{
			AvgMetrics: &structpb.Struct{
				Fields: map[string]*structpb.Value{
					defaultSearcherMetric: {
						Kind: &structpb.Value_NumberValue{
							NumberValue: metricValue,
						},
					},
				},
			},
			BatchMetrics: []*structpb.Struct{},
		},
	}
	err := db.AddValidationMetrics(context.TODO(), &m)
	require.NoError(t, err)
	return &m
}

func requireMockCheckpoint(
	t *testing.T, db *PgDB, task *model.Task, allocation *model.Allocation, checkpoint_id int,
) *model.CheckpointV2 {
	uuid_val := uuid.New()

	ckpt := model.CheckpointV2{
		UUID:         uuid_val,
		TaskID:       task.TaskID,
		AllocationID: allocation.AllocationID,
		ReportTime:   time.Now().UTC().Truncate(time.Millisecond),
		State:        model.ActiveState,
		Resources:    map[string]int64{"ok": 1.0},
		Metadata:     model.JSONObjFromMapStringInt64(map[string]int64{"ok": 1.0}),
	}
	err := db.AddCheckpointMetadata(context.TODO(), &ckpt)
	require.NoError(t, err)
	return &ckpt
}
