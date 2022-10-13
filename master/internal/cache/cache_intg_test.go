//go:build integration
// +build integration

package cache

import (
	"os"
	"testing"
	"time"

	"github.com/stretchr/testify/require"

	"github.com/determined-ai/determined/master/pkg/etc"

	"github.com/determined-ai/determined/master/internal/db"
)

func TestCache(t *testing.T) {
	require.NoError(t, etc.SetRootPath(db.RootFromDB))
	dbIns := db.MustResolveTestPostgres(t)
	db.MustMigrateTestPostgres(t, dbIns, db.MigrationsFromDB)

	user := db.RequireMockUser(t, dbIns)
	expID := db.RequireMockExperiment(t, dbIns, user).ID

	testCacheDir := "/tmp/determined-cache"
	cache := NewFileCache(testCacheDir, 1*time.Hour)

	// Test fetch
	files, _, err := cache.getFileTree(expID)
	require.NoError(t, err)
	require.True(t, len(files) > 0)
	path := files[0].Path
	_, err = cache.FileContent(expID, path)
	require.NoError(t, err)

	// Test fetch to nested tree structure
	_, err = cache.FileTreeNested(expID)
	require.NoError(t, err)

	// Test fetch invalid path
	_, err = cache.FileContent(expID, "invalid-path")
	require.Error(t, err)

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
