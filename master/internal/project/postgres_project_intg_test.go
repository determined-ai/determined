//go:build integration
// +build integration

package project

import (
	"context"
	"strings"
	"testing"

	"github.com/google/uuid"
	"github.com/stretchr/testify/require"
	"google.golang.org/protobuf/types/known/wrapperspb"
	"gotest.tools/assert"

	internaldb "github.com/determined-ai/determined/master/internal/db"
	"github.com/determined-ai/determined/master/pkg/etc"
	"github.com/determined-ai/determined/master/pkg/model"
	"github.com/determined-ai/determined/proto/pkg/projectv1"
)

func TestProjectByName(t *testing.T) {
	require.NoError(t, etc.SetRootPath(internaldb.RootFromDB))
	testDB, closeDB := internaldb.MustResolveTestPostgres(t)
	defer closeDB()
	internaldb.MustMigrateTestPostgres(t, testDB, internaldb.MigrationsFromDB)

	// add a workspace, and project
	workspaceID, workspaceName := internaldb.RequireMockWorkspaceID(t, testDB, "")
	projectID, projectName := internaldb.RequireMockProjectID(t, testDB, workspaceID, false)

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
		_, archivedProjectName := internaldb.RequireMockProjectID(t, testDB, workspaceID, true)
		_, err := ProjectByName(context.Background(), workspaceName, archivedProjectName)
		require.Error(t, err)
	})
}

func TestGetProjectByKey(t *testing.T) {
	require.NoError(t, etc.SetRootPath(internaldb.RootFromDB))
	testDB, closeDB := internaldb.MustResolveTestPostgres(t)
	defer closeDB()
	internaldb.MustMigrateTestPostgres(t, testDB, internaldb.MigrationsFromDB)

	// add a workspace, and project
	user := model.User{
		Username: uuid.Must(uuid.NewRandom()).String(),
		Admin:    true,
	}
	_, err := internaldb.HackAddUser(
		context.Background(),
		&user,
	)
	require.NoError(t, err)
	workspaceID, _ := internaldb.RequireMockWorkspaceID(t, testDB, "")
	projectID, _ := internaldb.RequireMockProjectID(t, testDB, workspaceID, false)

	t.Run("valid project key", func(t *testing.T) {
		key := uuid.Must(uuid.NewRandom()).String()[:MaxProjectKeyLength]
		_, err := UpdateProject(
			context.Background(),
			int32(projectID),
			user,
			&projectv1.PatchProject{Key: &wrapperspb.StringValue{Value: key}},
		)
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
	testDB, closeDB := internaldb.MustResolveTestPostgres(t)
	defer closeDB()
	internaldb.MustMigrateTestPostgres(t, testDB, internaldb.MigrationsFromDB)

	// add a workspace, and project
	workspaceID, _ := internaldb.RequireMockWorkspaceID(t, testDB, "")
	projectID, _ := internaldb.RequireMockProjectID(t, testDB, workspaceID, false)

	user := model.User{
		Username: uuid.Must(uuid.NewRandom()).String(),
		Admin:    true,
	}
	_, err := internaldb.HackAddUser(
		context.Background(),
		&user,
	)
	require.NoError(t, err)

	t.Run("update project key", func(t *testing.T) {
		key := uuid.Must(uuid.NewRandom()).String()[:MaxProjectKeyLength]
		_, err := UpdateProject(
			context.Background(),
			int32(projectID),
			user,
			&projectv1.PatchProject{Key: &wrapperspb.StringValue{Value: key}},
		)
		require.NoError(t, err)
	})

	t.Run("update project key with invalid project id", func(t *testing.T) {
		key := uuid.Must(uuid.NewRandom()).String()[:MaxProjectKeyLength]
		_, err := UpdateProject(
			context.Background(),
			0,
			user,
			&projectv1.PatchProject{Key: &wrapperspb.StringValue{Value: key}},
		)
		require.Error(t, err)
	})

	t.Run("update project key with invalid key", func(t *testing.T) {
		invalidKey := strings.Repeat("a", MaxProjectKeyLength+1)
		_, err := UpdateProject(
			context.Background(),
			int32(projectID),
			user,
			&projectv1.PatchProject{Key: &wrapperspb.StringValue{Value: invalidKey}},
		)
		require.Error(t, err)
	})
}
