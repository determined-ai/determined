//go:build integration
// +build integration

package task

import (
	"context"
	"testing"

	"github.com/stretchr/testify/require"
	"gopkg.in/oauth2.v3/utils/uuid"

	"github.com/determined-ai/determined/master/internal/db"
	"github.com/determined-ai/determined/master/pkg/etc"
	"github.com/determined-ai/determined/master/pkg/model"
)

func TestPersistAllocationWorkspaceInfo(t *testing.T) {
	ctx := context.Background()
	require.NoError(t, etc.SetRootPath(db.RootFromDB))
	pgDB, cleanup := db.MustResolveTestPostgres(t)
	defer cleanup()
	db.MustMigrateTestPostgres(t, pgDB, db.MigrationsFromDB)

	type TestCase struct {
		description string
		isTrial     bool
	}
	testCases := []TestCase{
		{description: "NTSC Task", isTrial: false},
		{description: "Trial", isTrial: true},
	}

	for _, tc := range testCases {
		t.Run(tc.description, func(t *testing.T) {
			workspaceID := model.DefaultWorkspaceID
			workspaceName := model.DefaultWorkspaceName

			// Create dummy allocation & experiment ids.
			allocID := model.AllocationID(uuid.Must(uuid.NewRandom()).String())
			user := db.RequireMockUser(t, pgDB)
			exp := db.RequireMockExperiment(t, pgDB, user)

			var err error
			switch tc.isTrial {
			case true:
				err = InsertTrialAllocationWorkspaceRecord(
					ctx,
					exp.ID,
					allocID,
				)
			case false:
				err = InsertNTSCAllocationWorkspaceRecord(
					ctx,
					allocID,
					workspaceID,
					workspaceName,
				)
			}
			require.NoError(t, err)

			// Check if entry in allocation_workspace_info table is correct.
			var persistedInfo model.AllocationWorkspaceRecord
			err = db.Bun().NewSelect().
				Model(&persistedInfo).
				Where("allocation_id = ?", allocID).
				Scan(ctx)

			require.NoError(t, err)
			require.Equal(t, allocID, persistedInfo.AllocationID)
			require.Equal(t, workspaceID, persistedInfo.WorkspaceID)
			require.Equal(t, workspaceName, persistedInfo.WorkspaceName)
			if tc.isTrial {
				require.Equal(t, exp.ID, persistedInfo.ExperimentID)
			}
		})
	}
}
