//go:build integration
// +build integration

package project

import (
	"context"
	"testing"

	"github.com/stretchr/testify/require"
	"gotest.tools/assert"

	internaldb "github.com/determined-ai/determined/master/internal/db"
	"github.com/determined-ai/determined/master/pkg/etc"
)

func TestProjectByName(t *testing.T) {
	require.NoError(t, etc.SetRootPath(internaldb.RootFromDB))
	db := internaldb.MustResolveTestPostgres(t)
	internaldb.MustMigrateTestPostgres(t, db, internaldb.MigrationsFromDB)

	// add a workspace, and project
	workspaceID, workspaceName := internaldb.RequireMockWorkspaceID(t, db, "")
	projectID, projectName := internaldb.RequireMockProjectID(t, db, workspaceID, false)

	t.Run("valid project name", func(t *testing.T) {
		actualProjectID, err := ProjectByName(context.Background(), workspaceName, projectName)
		require.NoError(t, err)
		assert.Equal(t, projectID, actualProjectID)
	})

	t.Run("invalid project name", func(t *testing.T) {
		_, err := ProjectByName(context.Background(), workspaceName, "bogus")
		require.Error(t, err)
	})

	t.Run("archived project", func(t *testing.T) {
		// add archvied project to workspace
		_, archivedProjectName := internaldb.RequireMockProjectID(t, db, workspaceID, true)
		_, err := ProjectByName(context.Background(), workspaceName, archivedProjectName)
		require.Error(t, err)
	})
}
