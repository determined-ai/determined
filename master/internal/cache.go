package internal

import (
	"context"
	"io/fs"
	"net/http"
	"os"
	"path/filepath"
	"strconv"
	"sync"
	"time"

	log "github.com/sirupsen/logrus"

	"google.golang.org/protobuf/types/known/timestamppb"

	"github.com/determined-ai/determined/master/internal/config"
	"github.com/determined-ai/determined/master/internal/db"
	"github.com/determined-ai/determined/master/pkg/archive"
	"github.com/determined-ai/determined/proto/pkg/experimentv1"
)

var modelDefCache *FileCache

const cacheDir = "exp_model_def"
const defaultCacheMaxAge = 24 * time.Hour

type modelDefFolder struct {
	fileTree   []*experimentv1.FileNode
	cachedTime time.Time
	lock       sync.RWMutex
	path       string
}

// FileCache is metadata for files cached at file system.
type FileCache struct {
	rootDir string
	maxAge  time.Duration
	caches  map[int]*modelDefFolder
	lock    sync.Mutex
}

// GetModelDefCache returns FileCache object.
func GetModelDefCache(cacheMaxAge *time.Duration) *FileCache {
	config := config.GetMasterConfig()
	if modelDefCache == nil {
		rootDir := filepath.Join(config.Cache, cacheDir)
		err := os.RemoveAll(rootDir)
		if err != nil {
			log.WithError(err).Errorf("failed to initialize model def cache at %s", rootDir)
		}
		var maxAge time.Duration
		if cacheMaxAge != nil {
			maxAge = *cacheMaxAge
		} else {
			maxAge = defaultCacheMaxAge
		}
		modelDefCache = &FileCache{
			rootDir: rootDir,
			maxAge:  maxAge,
			caches:  make(map[int]*modelDefFolder),
		}
	}
	return modelDefCache
}

func (f *FileCache) getOrCreateFolder(expID int) (*modelDefFolder, error) {
	f.lock.Lock()
	defer f.lock.Unlock()
	value, ok := f.caches[expID]
	if !ok {
		exp := struct {
			ModelDefinition []byte
		}{}
		err := db.Bun().NewSelect().TableExpr(
			"experiments").Column("model_definition").Where("id = ?", expID).Scan(context.TODO(), &exp)
		if err != nil {
			return nil, err
		}
		var fileTree []*experimentv1.FileNode
		arc, err := archive.FromTarGz(exp.ModelDefinition)
		if err != nil {
			return nil, err
		}
		err = os.MkdirAll(f.genPath(expID, ""), fs.ModePerm)
		if err != nil {
			return nil, err
		}
		for _, ar := range arc {
			if ar.IsDir() {
				err = os.MkdirAll(f.genPath(expID, ar.Path), fs.ModePerm)
			} else {
				err = os.WriteFile(f.genPath(expID, ar.Path), ar.Content, fs.ModePerm)
			}
			if err != nil {
				return nil, err
			}
			fileTree = append(fileTree, &experimentv1.FileNode{
				Path:          ar.Path,
				ModifiedTime:  timestamppb.New(ar.ModifiedTime.Time),
				ContentLength: int32(len(ar.Content)),
				IsDir:         ar.IsDir(),
				ContentType:   http.DetectContentType(ar.Content),
			})
		}
		value = &modelDefFolder{
			path:       f.genPath(expID, ""),
			fileTree:   fileTree,
			cachedTime: time.Now(),
		}
		f.caches[expID] = value
		f.prune()
	}
	return value, nil
}

func (f *FileCache) prune() {
	for expID, folder := range f.caches {
		if folder.cachedTime.Add(f.maxAge).Before(time.Now()) {
			err := os.RemoveAll(folder.path)
			if err != nil {
				log.WithError(err).Errorf("failed to prune model definition cache under %s", folder.path)
			}
			delete(f.caches, expID)
		}
	}
}

func (f *FileCache) genPath(expID int, path string) string {
	return filepath.Join(f.rootDir, strconv.Itoa(expID), path)
}

// GetFileTree returns folder tree structure with given experiment id.
func (f *FileCache) GetFileTree(expID int) ([]*experimentv1.FileNode, error) {
	folder, err := f.getOrCreateFolder(expID)
	if err != nil {
		return nil, err
	}
	folder.lock.RLock()
	defer folder.lock.RUnlock()
	return folder.fileTree, nil
}

// GetFileContent returns file with given experiment id and path.
func (f *FileCache) GetFileContent(expID int, path string) ([]byte, error) {
	folder, err := f.getOrCreateFolder(expID)
	if err != nil {
		return []byte{}, err
	}
	fileTree, err := f.GetFileTree(expID)
	if err != nil {
		return []byte{}, err
	}
	for _, file := range fileTree {
		if file.Path == path {
			folder.lock.RLock()
			defer folder.lock.RUnlock()
			file, err := os.ReadFile(f.genPath(expID, path))
			if err != nil {
				_, ok := err.(*fs.PathError)
				if ok {
					// This means memory and file system are out of sync.
					err = os.RemoveAll(f.rootDir)
					if err != nil {
						return []byte{}, err
					}
					f.caches = make(map[int]*modelDefFolder)
					return f.GetFileContent(expID, path)
				}
				return []byte{}, err
			}
			return file, err
		}
	}
	return nil, fs.ErrNotExist
}
