//go:build integration
// +build integration

package db

import (
	"crypto/rand"
	"fmt"
	"testing"
	"time"

	"github.com/davecgh/go-spew/spew"
	"github.com/google/uuid"
	"github.com/stretchr/testify/require"
	"gopkg.in/guregu/null.v3"

	"github.com/determined-ai/determined/master/pkg/model"
	"github.com/determined-ai/determined/master/pkg/ptrs"
	"github.com/determined-ai/determined/master/pkg/schemas"
	"github.com/determined-ai/determined/master/pkg/schemas/expconf"
)

func pprintedExpect(expected, got interface{}) string {
	return fmt.Sprintf("expected \n\t%s\ngot\n\t%s", spew.Sdump(expected), spew.Sdump(got))
}

func requireMockUser(t *testing.T, db *PgDB) *model.User {
	user := model.User{
		Username:     uuid.NewString(),
		PasswordHash: null.NewString("", false),
		Active:       true,
	}
	err := db.AddUser(&user, nil)
	require.NoError(t, err, "failed to add user")
	return &user
}

const defaultSearcherMetric = "okness"

func requireMockExperiment(t *testing.T, db *PgDB, user *model.User) *model.Experiment {
	cfg := schemas.WithDefaults(expconf.ExperimentConfigV0{
		RawCheckpointStorage: &expconf.CheckpointStorageConfigV0{
			RawSharedFSConfig: &expconf.SharedFSConfigV0{
				RawHostPath: ptrs.StringPtr("/home/ckpts"),
			},
		},
		RawEntrypoint: &expconf.EntrypointV0{
			RawEntrypoint: ptrs.StringPtr("model.Classifier"),
		},
		RawHyperparameters: map[string]expconf.HyperparameterV0{
			"global_batch_size": {
				RawConstHyperparameter: &expconf.ConstHyperparameterV0{
					RawVal: ptrs.IntPtr(1),
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
			RawMetric: ptrs.StringPtr(defaultSearcherMetric),
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
	}
	err := db.AddExperiment(&exp)
	require.NoError(t, err, "failed to add experiment")
	return &exp
}

func requireMockTrial(t *testing.T, db *PgDB, exp *model.Experiment) *model.Trial {
	rqID := model.NewRequestID(rand.Reader)
	tr := model.Trial{
		TaskID:       model.NewTaskID(),
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
