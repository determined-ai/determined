package internal

import (
	"path/filepath"
	"time"

	log "github.com/sirupsen/logrus"

	"github.com/determined-ai/determined/master/internal/cache"
	"github.com/determined-ai/determined/master/internal/config"
)

var modelDefCache *cache.FileCache

const cacheDir = "exp_model_def"

// GetModelDefCache returns FileCache object.
func GetModelDefCache() *cache.FileCache {
	if modelDefCache == nil {
		config := config.GetMasterConfig()
		rootDir := filepath.Join(config.Cache.CacheDir, cacheDir)
		maxAge, err := time.ParseDuration(config.Cache.MaxAge)
		if err != nil {
			log.WithError(err).Errorf("failed to parse cache max age for %s", config.Cache.MaxAge)
		}
		modelDefCache = cache.NewFileCache(rootDir, maxAge)
	}
	return modelDefCache
}
