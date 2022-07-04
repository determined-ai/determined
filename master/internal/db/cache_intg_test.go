//go:build integration
// +build integration

package db

import (
	"github.com/stretchr/testify/require"

	"github.com/determined-ai/determined/master/pkg/etc"

	"os"
	"testing"
	"time"
)

func TestCache(t *testing.T) {
	etc.SetRootPath(rootFromDB)
	db := MustResolveTestPostgres(t)
	MustMigrateTestPostgres(t, db, migrationsFromDB)

	user := requireMockUser(t, db)
	expID := requireMockExperiment(t, db, user).ID

	testCacheDir := "/tmp/determined-cache"
	cache := NewFileCache(testCacheDir, 1*time.Hour)

	// Test fetch
	files, err := cache.getFileTree(expID)
	require.NoError(t, err)
	require.True(t, len(files) > 0)
	path := files[0].Path
	_, err = cache.GetFileContent(expID, path)
	require.NoError(t, err)

	// Test fetch to nested tree structure
	files, err = cache.GetFileTreeNested(expID)
	require.NoError(t, err)

	// Test prune, first verify the file exists, then modify cached time to make cache expire
	// after prune, file no longer exist
	_, err = os.ReadFile(cache.genPath(expID, path))
	require.NoError(t, err)
	cache.caches[expID].cachedTime = time.Now().Add(-2 * time.Hour)
	cache.prune()
	_, err = os.ReadFile(cache.genPath(expID, path))
	require.Error(t, err)

	err = os.RemoveAll(testCacheDir)
	require.NoError(t, err)
}
