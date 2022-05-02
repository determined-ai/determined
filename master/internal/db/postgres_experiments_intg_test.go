//go:build integration
// +build integration

package db

import (
	"context"
	"fmt"
	"strings"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/stretchr/testify/require"
	"google.golang.org/protobuf/types/known/structpb"

	"github.com/determined-ai/determined/master/pkg/etc"
	"github.com/determined-ai/determined/master/pkg/model"
	"github.com/determined-ai/determined/master/pkg/ptrs"
	"github.com/determined-ai/determined/master/pkg/schemas"
	"github.com/determined-ai/determined/master/pkg/schemas/expconf"
	"github.com/determined-ai/determined/proto/pkg/apiv1"
	"github.com/determined-ai/determined/proto/pkg/checkpointv1"
	"github.com/determined-ai/determined/proto/pkg/experimentv1"
	"github.com/determined-ai/determined/proto/pkg/trialv1"
)

func TestGetExperiments(t *testing.T) {
	etc.SetRootPath(rootFromDB)
	db := MustResolveTestPostgres(t)
	MustMigrateTestPostgres(t, db, migrationsFromDB)

	// Add a mock user.
	user := requireMockUser(t, db)
	const (
		adesc     = "a description"
		alabel    = "a label"
		aname     = "a name"
		orderByID = "id ASC"
	)

	tests := []struct {
		name           string
		numResults     int
		exps           []model.Experiment
		require        func(*testing.T, *experimentv1.Experiment)
		stateFilter    string
		archivedFilter string
		labelFilter    string
		descFilter     string
		nameFilter     string
		offset, limit  int
	}{
		{
			name:       "empty result",
			numResults: 0,
			exps:       func() []model.Experiment { return nil }(),
		},
		{
			name:       "unfiltered result",
			numResults: 3,
			exps: func() []model.Experiment {
				var exps []model.Experiment
				for i := 0; i < 3; i++ {
					exps = append(exps, mockModelExperiment(user, mockExpconf()))
				}
				return exps
			}(),
		},
		{
			name:       "filter by state",
			numResults: 3,
			exps: func() []model.Experiment {
				var exps []model.Experiment
				for i := 0; i < 3; i++ {
					exp := mockModelExperiment(user, mockExpconf())
					exp.State = model.CanceledState
					exps = append(exps, exp)
				}
				return exps
			}(),
			require: func(t *testing.T, e *experimentv1.Experiment) {
				require.Equal(t, experimentv1.State_STATE_CANCELED, e.State)
			},
			stateFilter: string(model.CanceledState),
		},
		{
			name:       "filter by archived",
			numResults: 3,
			exps: func() []model.Experiment {
				var exps []model.Experiment
				for i := 0; i < 3; i++ {
					exp := mockModelExperiment(user, mockExpconf())
					exp.Archived = true
					exps = append(exps, exp)
				}
				return exps
			}(),
			require: func(t *testing.T, e *experimentv1.Experiment) {
				require.Equal(t, true, e.Archived)
			},
			archivedFilter: "true",
		},
		{
			name:       "filter by labels",
			numResults: 3,
			exps: func() []model.Experiment {
				cfg := mockExpconf()
				cfg.SetLabels(expconf.LabelsV0{alabel: true})

				var exps []model.Experiment
				for i := 0; i < 3; i++ {
					exps = append(exps, mockModelExperiment(user, cfg))
				}
				return exps
			}(),
			require: func(t *testing.T, e *experimentv1.Experiment) {
				require.Len(t, e.Labels, 1)
				require.Equal(t, alabel, e.Labels[0])
			},
			labelFilter: alabel,
		},
		{
			name:       "filter by description",
			numResults: 3,
			exps: func() []model.Experiment {
				cfg := mockExpconf()
				cfg.SetDescription(ptrs.Ptr(adesc))

				var exps []model.Experiment
				for i := 0; i < 3; i++ {
					exps = append(exps, mockModelExperiment(user, cfg))
				}
				return exps
			}(),
			require: func(t *testing.T, e *experimentv1.Experiment) {
				require.Contains(t, adesc, e.Description)
			},
			descFilter: adesc,
		},
		{
			name:       "filter by name",
			numResults: 3,
			exps: func() []model.Experiment {
				cfg := mockExpconf()
				cfg.SetName(expconf.Name{RawString: ptrs.Ptr(aname)})

				var exps []model.Experiment
				for i := 0; i < 3; i++ {
					exps = append(exps, mockModelExperiment(user, cfg))
				}
				return exps
			}(),
			require: func(t *testing.T, e *experimentv1.Experiment) {
				require.Contains(t, aname, e.Name)
			},
			nameFilter: aname,
		},
		{
			// Offset 1 in and expect 2 back.
			name:       "filter by name, with offset",
			numResults: 2,
			exps: func() []model.Experiment {
				cfg := mockExpconf()
				cfg.SetName(expconf.Name{RawString: ptrs.Ptr(aname + "1")})

				var exps []model.Experiment
				for i := 0; i < 3; i++ {
					exps = append(exps, mockModelExperiment(user, cfg))
				}
				return exps
			}(),
			require: func(t *testing.T, e *experimentv1.Experiment) {
				require.Contains(t, aname+"1", e.Name)
			},
			nameFilter: aname + "1",
			offset:     1,
		},
		{
			// Limit 2 and expect 2 back.
			name:       "filter by name, with limit",
			numResults: 2,
			exps: func() []model.Experiment {
				cfg := mockExpconf()
				cfg.SetName(expconf.Name{RawString: ptrs.Ptr(aname + "1")})

				var exps []model.Experiment
				for i := 0; i < 6; i++ {
					exps = append(exps, mockModelExperiment(user, cfg))
				}
				return exps
			}(),
			require: func(t *testing.T, e *experimentv1.Experiment) {
				require.Contains(t, aname+"1", e.Name)
			},
			nameFilter: aname + "1",
			limit:      2,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Add mock experiments.
			for _, exp := range tt.exps {
				err := db.AddExperiment(&exp)
				require.NoError(t, err, "failed to add experiment")
			}
			userIDFilterExpr := strings.Trim(
				strings.Join(strings.Split(fmt.Sprint([]int32{int32(user.ID)}), " "), ","),
				"[]",
			)
			resp := &apiv1.GetExperimentsResponse{}
			err := db.QueryProtof(
				"get_experiments",
				[]interface{}{orderByID},
				resp,
				tt.stateFilter,
				tt.archivedFilter,
				user.Username, // Always filter by a random user so the state is inconsequential.
				userIDFilterExpr,
				tt.labelFilter,
				tt.descFilter,
				tt.nameFilter,
				0,
				tt.offset,
				tt.limit,
			)
			require.NoError(t, err)
			require.Len(t, resp.Experiments, tt.numResults)
			if tt.require != nil {
				for _, e := range resp.Experiments {
					tt.require(t, e)
				}
			}
		})
	}
}

func mockModelExperiment(user model.User, expConf expconf.ExperimentConfigV0) model.Experiment {
	return model.Experiment{
		JobID:                model.NewJobID(),
		State:                model.ActiveState,
		Config:               expConf,
		ModelDefinitionBytes: []byte{1, 0, 1, 0, 1, 0},
		StartTime:            time.Now().Add(-time.Hour),
		OwnerID:              &user.ID,
		Username:             user.Username,
		Archived:             false,
		ProjectID:            1,
	}
}

func mockExpconf() expconf.ExperimentConfig {
	return schemas.WithDefaults(expconf.ExperimentConfigV0{
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
}

func TestCheckpointMetadata(t *testing.T) {
	etc.SetRootPath(rootFromDB)
	db := MustResolveTestPostgres(t)
	MustMigrateTestPostgres(t, db, migrationsFromDB)

	tests := []struct {
		name          string
		hasValidation bool
	}{
		{
			name:          "checkpoints associated validations",
			hasValidation: true,
		},
		{
			name:          "checkpoints not associated validations",
			hasValidation: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			user := requireMockUser(t, db)
			exp := requireMockExperiment(t, db, user)
			tr := requireMockTrial(t, db, exp)

			ckptUuid := uuid.NewString()
			latestBatch := int32(10)
			ckpt := &trialv1.CheckpointMetadata{
				TrialId:           int32(tr.ID),
				Uuid:              ckptUuid,
				Resources:         map[string]int64{"ok": 1.0},
				Framework:         "some framework",
				Format:            "some format",
				DeterminedVersion: "1.0.0",
				LatestBatch:       latestBatch,
			}
			err := db.AddCheckpointMetadata(context.TODO(), ckpt)
			require.NoError(t, err)

			var m *trialv1.TrialMetrics
			const metricValue = 1.0
			if tt.hasValidation {
				m = &trialv1.TrialMetrics{
					TrialId:     int32(tr.ID),
					LatestBatch: latestBatch,
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
				err := db.AddValidationMetrics(context.TODO(), m)
				require.NoError(t, err)
			}

			requireCheckpointOk := func(
				expected *trialv1.CheckpointMetadata,
				actual checkpointv1.Checkpoint,
			) {
				require.Equal(t, expected, &trialv1.CheckpointMetadata{
					TrialId:           actual.TrialId,
					Uuid:              actual.Uuid,
					Resources:         actual.Resources,
					Framework:         actual.Framework,
					Format:            actual.Format,
					DeterminedVersion: actual.DeterminedVersion,
					LatestBatch:       actual.BatchNumber,
				})
				if tt.hasValidation {
					require.Equal(t, metricValue, actual.SearcherMetric.Value)
					require.Equal(t, checkpointv1.State_STATE_COMPLETED, actual.ValidationState)
					require.NotNil(t, actual.Metrics)
				} else {
					require.Nil(t, actual.SearcherMetric)
					require.Equal(t, checkpointv1.State_STATE_UNSPECIFIED, actual.ValidationState)
					require.Nil(t, actual.Metrics)
				}
			}

			var retCkpt checkpointv1.Checkpoint
			err = db.QueryProto("get_checkpoint", &retCkpt, ckptUuid)
			require.NoError(t, err, "failed to get checkpoint")
			requireCheckpointOk(ckpt, retCkpt)

			var retCkpts []*checkpointv1.Checkpoint
			err = db.QueryProto("get_checkpoints_for_trial", &retCkpts, tr.ID)
			require.NoError(t, err)
			require.Len(t, retCkpts, 1)
			requireCheckpointOk(ckpt, *retCkpts[0])

			retCkpts = nil
			err = db.QueryProto("get_checkpoints_for_experiment", &retCkpts, exp.ID)
			require.NoError(t, err)
			require.Len(t, retCkpts, 1)
			requireCheckpointOk(ckpt, *retCkpts[0])
		})
	}
}
