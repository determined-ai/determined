package internal

import (
	"os"
	"path/filepath"
	"time"

	log "github.com/sirupsen/logrus"

	"github.com/determined-ai/determined/master/internal/config"
	"github.com/determined-ai/determined/master/internal/db"
)

var modelDefCache *db.FileCache

const cacheDir = "exp_model_def"

// GetModelDefCache returns FileCache object.
func GetModelDefCache() *db.FileCache {
	config := config.GetMasterConfig()
	if modelDefCache == nil {
		rootDir := filepath.Join(config.Cache.CacheDir, cacheDir)
		err := os.RemoveAll(rootDir)
		if err != nil {
			log.WithError(err).Errorf("failed to initialize model def cache at %s", rootDir)
		}
		maxAge, err := time.ParseDuration(config.Cache.MaxAge)
		if err != nil {
			log.WithError(err).Errorf("failed to parse cache max age for %s", config.Cache.MaxAge)
		}
		modelDefCache = db.NewFileCache(rootDir, maxAge)
	}
	return modelDefCache
}
