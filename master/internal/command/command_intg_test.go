//go:build integration
// +build integration

package command

import (
	"context"
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/determined-ai/determined/master/internal/db"
	"github.com/determined-ai/determined/master/pkg/etc"
)

func TestCommandPersistAndEvictContextDirectoryFromMemory(t *testing.T) {
	require.NoError(t, etc.SetRootPath("../static/srv"))

	// real db.
	pgDB := db.MustSetupTestPostgres(t)
	task := db.RequireMockTask(t, pgDB, nil)

	expected := []byte{0, 1, 2, 7}
	c := command{
		db:               pgDB,
		taskID:           task.TaskID,
		contextDirectory: expected,
	}
	require.NoError(t, c.persistAndEvictContextDirectoryFromMemory())
	require.Nil(t, c.contextDirectory)

	actual, err := db.NonExperimentTasksContextDirectory(context.TODO(), task.TaskID)
	require.NoError(t, err)
	require.Equal(t, expected, actual)

	task = db.RequireMockTask(t, pgDB, nil)
	c = command{
		db:               pgDB,
		taskID:           task.TaskID,
		contextDirectory: nil, // don't error with nil context directory.
	}
	require.NoError(t, c.persistAndEvictContextDirectoryFromMemory())
	require.Nil(t, c.contextDirectory)

	actual, err = db.NonExperimentTasksContextDirectory(context.TODO(), task.TaskID)
	require.NoError(t, err)
	require.Len(t, actual, 0)
}
