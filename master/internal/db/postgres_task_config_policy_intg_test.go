//go:build integration
// +build integration

package db

import (
	"context"
	"testing"

	"github.com/determined-ai/determined/master/pkg/etc"
	"github.com/determined-ai/determined/master/pkg/model"
	"github.com/determined-ai/determined/master/pkg/ptrs"
	"github.com/determined-ai/determined/master/pkg/schemas/expconf"
	"github.com/google/uuid"
	log "github.com/sirupsen/logrus"

	"github.com/stretchr/testify/require"
)

func TestDeleteConfigPolicies(t *testing.T) {
	ctx := context.Background()
	require.NoError(t, etc.SetRootPath(RootFromDB))
	pgDB, cleanup := MustResolveNewPostgresDatabase(t)
	defer cleanup()
	MustMigrateTestPostgres(t, pgDB, MigrationsFromDB)

	user := RequireMockUser(t, pgDB)
	workspaceIDs := []int32{}

	defer func() {
		err := CleanupMockWorkspace(workspaceIDs)
		if err != nil {
			log.Errorf("error when cleaning up mock workspaces")
		}
	}()

	tests := []struct {
		name               string
		global             bool
		workloadType       model.WorkloadType
		hasInvariantConfig bool
		hasConstraints     bool
		err                error
	}{
		{
			name:               "exp config no constraint",
			global:             false,
			workloadType:       model.ExperimentType,
			hasInvariantConfig: true,
			hasConstraints:     false,
			err:                nil,
		},
		{
			name:               "ntsc config no constraint",
			global:             false,
			workloadType:       model.NTSCType,
			hasInvariantConfig: true,
			hasConstraints:     false,
			err:                nil,
		},
		{
			name:               "exp config and constraint",
			global:             false,
			workloadType:       model.ExperimentType,
			hasInvariantConfig: true,
			hasConstraints:     true,
			err:                nil,
		},
		{
			name:               "ntsc config and constraint",
			global:             false,
			workloadType:       model.NTSCType,
			hasInvariantConfig: true,
			hasConstraints:     true,
			err:                nil,
		},
		{
			name:               "global exp config no constraint",
			global:             true,
			workloadType:       model.ExperimentType,
			hasInvariantConfig: true,
			hasConstraints:     false,
			err:                nil,
		},
		{
			name:               "global config no constraint",
			global:             true,
			workloadType:       model.NTSCType,
			hasInvariantConfig: true,
			hasConstraints:     false,
			err:                nil,
		},
		{
			name:               "global exp config and constraint",
			global:             true,
			workloadType:       model.ExperimentType,
			hasInvariantConfig: true,
			hasConstraints:     true,
			err:                nil,
		},
		{
			name:               "global ntsc config and constraint",
			global:             true,
			workloadType:       model.NTSCType,
			hasInvariantConfig: true,
			hasConstraints:     true,
			err:                nil,
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			switch test.workloadType {
			case model.ExperimentType:
				w, expTCP, _ := CreateMockTaskConfigPolicies(ctx, t, pgDB, user, test.global)
				workspaceIDs = append(workspaceIDs, int32(w.ID))
				if !test.hasInvariantConfig {
					expTCP.InvariantConfig = nil
				}
				if !test.hasConstraints {
					expTCP.Constraints = nil
				}
				err := DeleteConfigPolicies(ctx, expTCP.WorkspaceID, test.workloadType)
				require.Equal(t, test.err, err)
			case model.NTSCType:
				w, _, ntscTCP := CreateMockTaskConfigPolicies(ctx, t, pgDB, user, test.global)
				workspaceIDs = append(workspaceIDs, int32(w.ID))
				err := DeleteConfigPolicies(ctx, ntscTCP.WorkspaceID, test.workloadType)
				require.Equal(t, test.err, err)
			}
		})
	}
}

func CreateMockTaskConfigPolicies(ctx context.Context, t *testing.T,
	pgDB *PgDB, user model.User, global bool) (*model.Workspace,
	*model.ExperimentTaskConfigPolicies, *model.NTSCTaskConfigPolicies) {

	var scope *int
	var w model.Workspace
	if !global {
		w = model.Workspace{Name: uuid.NewString(), UserID: user.ID}
		_, err := Bun().NewInsert().Model(&w).Exec(ctx)
		require.NoError(t, err)
		scope = &w.ID
	}

	experimentTCP := &model.ExperimentTaskConfigPolicies{
		WorkspaceID:     scope,
		LastUpdatedBy:   user.ID,
		WorkloadType:    model.ExperimentType,
		InvariantConfig: DefaultExperimentConfig(),
		Constraints:     &model.Constraints{PriorityLimit: ptrs.Ptr(10)},
	}
	err := SetExperimentConfigPolicies(ctx, experimentTCP)
	require.NoError(t, err)

	ntscTCP := &model.NTSCTaskConfigPolicies{
		WorkspaceID:   scope,
		LastUpdatedBy: user.ID,
		WorkloadType:  model.ExperimentType,
		InvariantConfig: &model.CommandConfig{
			Description: "random description",
			Resources: model.ResourcesConfig{
				Slots:    4,
				MaxSlots: ptrs.Ptr(8),
			},
		},
		Constraints: &model.Constraints{PriorityLimit: ptrs.Ptr[int](10),
			ResourceConstraints: &model.ResourceConstraints{
				MaxSlots: ptrs.Ptr(10),
			}},
	}
	err = SetNTSCConfigPolicies(ctx, ntscTCP)
	require.NoError(t, err)

	return &w, experimentTCP, ntscTCP
}

func DefaultExperimentConfig() *expconf.ExperimentConfigV0 {
	return &expconf.ExperimentConfigV0{
		RawCheckpointStorage: &expconf.CheckpointStorageConfigV0{
			RawSharedFSConfig: &expconf.SharedFSConfigV0{
				RawHostPath: ptrs.Ptr("path/to/config")},
		},
		RawHyperparameters: expconf.HyperparametersV0{},
		RawName:            expconf.Name{RawString: ptrs.Ptr(uuid.NewString())},
		RawReproducibility: &expconf.ReproducibilityConfigV0{
			RawExperimentSeed: ptrs.Ptr[uint32](10),
		},
		RawSearcher: &expconf.SearcherConfigV0{
			RawMetric: ptrs.Ptr("training_test"),
			RawSingleConfig: &expconf.SingleConfigV0{
				RawMaxLength: &expconf.LengthV0{
					Unit:  expconf.Batches,
					Units: uint64(2),
				},
			},
		},
	}
}
