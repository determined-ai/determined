//go:build integration
// +build integration

package project

import (
	"context"
	"strings"
	"testing"

	"github.com/stretchr/testify/require"
	"gopkg.in/oauth2.v3/utils/uuid"
	"gotest.tools/assert"

	internaldb "github.com/determined-ai/determined/master/internal/db"
	"github.com/determined-ai/determined/master/pkg/etc"
)

func TestProjectByName(t *testing.T) {
	require.NoError(t, etc.SetRootPath(internaldb.RootFromDB))
	db, closeDB := internaldb.MustResolveTestPostgres(t)
	defer closeDB()
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
		// add archived project to workspace
		_, archivedProjectName := internaldb.RequireMockProjectID(t, db, workspaceID, true)
		_, err := ProjectByName(context.Background(), workspaceName, archivedProjectName)
		require.Error(t, err)
	})
}

func TestGetProjectByKey(t *testing.T) {
	require.NoError(t, etc.SetRootPath(internaldb.RootFromDB))
	db, close := internaldb.MustResolveTestPostgres(t)
	defer close()
	internaldb.MustMigrateTestPostgres(t, db, internaldb.MigrationsFromDB)

	// add a workspace, and project
	workspaceID, _ := internaldb.RequireMockWorkspaceID(t, db, "")
	projectID, _ := internaldb.RequireMockProjectID(t, db, workspaceID, false)

	t.Run("valid project key", func(t *testing.T) {
		key := uuid.Must(uuid.NewRandom()).String()[:MaxProjectKeyLength]
		err := UpdateProjectKey(context.Background(), projectID, key)
		require.NoError(t, err)
		project, err := GetProjectByKey(context.Background(), key)
		require.NoError(t, err)
		assert.Equal(t, projectID, project.ID)
	})

	t.Run("non-existent project key", func(t *testing.T) {
		_, err := GetProjectByKey(context.Background(), "bogus")
		require.Error(t, err)
		require.ErrorContains(t, err, "not found")
	})
}

func TestUpdateProjectKey(t *testing.T) {
	require.NoError(t, etc.SetRootPath(internaldb.RootFromDB))
	db, close := internaldb.MustResolveTestPostgres(t)
	defer close()
	internaldb.MustMigrateTestPostgres(t, db, internaldb.MigrationsFromDB)

	// add a workspace, and project
	workspaceID, _ := internaldb.RequireMockWorkspaceID(t, db, "")
	projectID, _ := internaldb.RequireMockProjectID(t, db, workspaceID, false)

	t.Run("update project key", func(t *testing.T) {
		key := uuid.Must(uuid.NewRandom()).String()[:MaxProjectKeyLength]
		err := UpdateProjectKey(context.Background(), projectID, key)
		require.NoError(t, err)
	})

	t.Run("update project key with invalid project id", func(t *testing.T) {
		key := uuid.Must(uuid.NewRandom()).String()[:MaxProjectKeyLength]
		err := UpdateProjectKey(context.Background(), 0, key)
		require.Error(t, err)
	})

	t.Run("update project key with invalid key", func(t *testing.T) {
		invalidKey := strings.Repeat("a", MaxProjectKeyLength+1)
		err := UpdateProjectKey(context.Background(), projectID, invalidKey)
		require.Error(t, err)
	})
}
