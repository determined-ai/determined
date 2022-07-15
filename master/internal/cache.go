package internal

import (
	"path/filepath"
	"sync"
	"time"

	"github.com/determined-ai/determined/master/internal/cache"
	"github.com/determined-ai/determined/master/internal/config"
)

var (
	modelDefCache      *cache.FileCache
	modelDefCacheMutex sync.Mutex
)

const (
	cacheDir    = "exp_model_def"
	cacheMaxAge = 24 * time.Hour
)

// GetModelDefCache returns FileCache object.
func GetModelDefCache() *cache.FileCache {
	modelDefCacheMutex.Lock()
	defer modelDefCacheMutex.Unlock()
	if modelDefCache == nil {
		config := config.GetMasterConfig()
		rootDir := filepath.Join(config.Cache.CacheDir, cacheDir)
		modelDefCache = cache.NewFileCache(rootDir, cacheMaxAge)
	}
	return modelDefCache
}
