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
	"gopkg.in/guregu/null.v3"

	"github.com/determined-ai/determined/master/pkg/model"
	"github.com/determined-ai/determined/master/pkg/ptrs"
	"github.com/determined-ai/determined/master/pkg/schemas"
	"github.com/determined-ai/determined/master/pkg/schemas/expconf"
	"github.com/determined-ai/determined/proto/pkg/modelv1"
	"github.com/determined-ai/determined/proto/pkg/trialv1"
)

func pprintedExpect(expected, got interface{}) string {
	return fmt.Sprintf("expected \n\t%s\ngot\n\t%s", spew.Sdump(expected), spew.Sdump(got))
}

func requireMockUser(t *testing.T, db *PgDB) model.User {
	user := model.User{
		Username:     uuid.NewString(),
		PasswordHash: null.NewString("", false),
		Active:       true,
	}
	err := db.AddUser(&user, nil)
	require.NoError(t, err, "failed to add user")
	return user
}

const defaultSearcherMetric = "okness"

func requireMockExperiment(t *testing.T, db *PgDB, user model.User) *model.Experiment {
	cfg := schemas.WithDefaults(expconf.ExperimentConfigV0{
		RawCheckpointStorage: &expconf.CheckpointStorageConfigV0{
			RawSharedFSConfig: &expconf.SharedFSConfigV0{
				RawHostPath: ptrs.Ptr("/home/ckpts"),
			},
		},
		RawEntrypoint: &expconf.EntrypointV0{
			RawEntrypoint: ptrs.Ptr("model.Classifier"),
		},
		RawHyperparameters: map[string]expconf.HyperparameterV0{
			"global_batch_size": {
				RawConstHyperparameter: &expconf.ConstHyperparameterV0{
					RawVal: ptrs.Ptr(1),
				},
			},
		},
		RawSearcher: &expconf.SearcherConfigV0{
			RawSingleConfig: &expconf.SingleConfigV0{
				RawMaxLength: &expconf.LengthV0{
					Unit:  expconf.Batches,
					Units: 1,
				},
			},
			RawMetric: ptrs.Ptr(defaultSearcherMetric),
		},
	}).(expconf.ExperimentConfigV0)

	exp := model.Experiment{
		JobID:                model.NewJobID(),
		State:                model.ActiveState,
		Config:               cfg,
		ModelDefinitionBytes: []byte{1, 0, 1, 0, 1, 0},
		StartTime:            time.Now().Add(-time.Hour),
		OwnerID:              &user.ID,
		Username:             user.Username,
		ProjectID:            1,
	}
	err := db.AddExperiment(&exp)
	require.NoError(t, err, "failed to add experiment")
	return &exp
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
	t *testing.T, db *PgDB, tr *model.Trial, latestBatch int, metricValue float64,
) *trialv1.TrialMetrics {
	m := trialv1.TrialMetrics{
		TrialId:     int32(tr.ID),
		LatestBatch: int32(latestBatch),
		Metrics: &structpb.Struct{
			Fields: map[string]*structpb.Value{
				defaultSearcherMetric: {
					Kind: &structpb.Value_NumberValue{
						NumberValue: metricValue,
					},
				},
			},
		},
		BatchMetrics: []*structpb.Struct{},
	}
	err := db.AddValidationMetrics(context.TODO(), &m)
	require.NoError(t, err)
	return &m
}

func requireMockCheckpoint(
	t *testing.T, db *PgDB, tr *model.Trial, latestBatch int,
) *trialv1.CheckpointMetadata {
	ckpt := trialv1.CheckpointMetadata{
		TrialId:           int32(tr.ID),
		Uuid:              uuid.NewString(),
		Resources:         map[string]int64{"ok": 1.0},
		Framework:         "some framework",
		Format:            "some format",
		DeterminedVersion: "1.0.0",
		LatestBatch:       int32(latestBatch),
	}
	err := db.AddCheckpointMetadata(context.TODO(), &ckpt)
	require.NoError(t, err)
	return &ckpt
}
