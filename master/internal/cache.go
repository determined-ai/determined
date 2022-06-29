package internal

import (
	"context"
	"errors"
	"io/fs"
	"net/http"
	"os"
	"path/filepath"
	"strconv"
	"sync"
	"time"

	log "github.com/sirupsen/logrus"

	"github.com/determined-ai/determined/master/internal/db"
	"github.com/determined-ai/determined/master/pkg/archive"
	"github.com/determined-ai/determined/proto/pkg/experimentv1"
	"google.golang.org/protobuf/types/known/timestamppb"
)

var modelDefCache *fileCache

const cacheDir = "/tmp/determined/cache/exp_model_def"

func GetModelDefCache() *fileCache {
	if modelDefCache == nil {
		os.RemoveAll(cacheDir)
		modelDefCache = &fileCache{
			rootDir: cacheDir,
			maxAge:  24 * time.Hour,
			caches:  make(map[int]*modelDefFolder),
		}
	}
	return modelDefCache
}

type modelDefFolder struct {
	fileTree   []*experimentv1.FileNode
	cachedTime time.Time
	lock       sync.RWMutex
	path       string
}

type fileCache struct {
	rootDir string
	maxAge  time.Duration
	caches  map[int]*modelDefFolder
	lock    sync.Mutex
}

func (f *fileCache) getOrCreateFolder(exp_id int) (*modelDefFolder, error) {
	f.lock.Lock()
	defer f.lock.Unlock()
	value, ok := f.caches[exp_id]
	if !ok {
		exp := struct {
			ModelDefinition []byte
		}{}
		err := db.Bun().NewSelect().TableExpr(
			"experiments").Column("model_definition").Where("id = ?", exp_id).Scan(context.TODO(), &exp)
		if err != nil {
			return nil, err
		}
		var fileTree []*experimentv1.FileNode
		arc, err := archive.FromTarGz(exp.ModelDefinition)
		if err != nil {
			return nil, err
		}
		for _, ar := range arc {
			if ar.IsDir() {
				err = os.MkdirAll(f.genPath(exp_id, ar.Path), fs.ModePerm)
			} else {
				err = os.WriteFile(f.genPath(exp_id, ar.Path), ar.Content, fs.ModePerm)
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
			path:       f.genPath(exp_id, ""),
			fileTree:   fileTree,
			cachedTime: time.Now(),
		}
		f.caches[exp_id] = value
	}
	f.prune()
	return value, nil
}

func (f *fileCache) prune() {
	for exp_id, folder := range f.caches {
		if folder.cachedTime.Add(f.maxAge).Before(time.Now()) {
			err := os.RemoveAll(folder.path)
			if err != nil {
				log.WithError(err).Errorf("failed to prune model definition cache under %s", folder.path)
			}
			delete(f.caches, exp_id)
		}
	}
}

func (f *fileCache) genPath(exp_id int, path string) string {
	return filepath.Join(f.rootDir, strconv.Itoa(exp_id), path)
}

func (f *fileCache) GetFileTree(exp_id int) ([]*experimentv1.FileNode, error) {
	folder, err := f.getOrCreateFolder(exp_id)
	if err != nil {
		return nil, err
	}
	folder.lock.RLock()
	defer folder.lock.RUnlock()
	return folder.fileTree, nil
}

func (f *fileCache) GetFileContent(exp_id int, path string) ([]byte, error) {
	folder, err := f.getOrCreateFolder(exp_id)
	if err != nil {
		return []byte{}, err
	}
	folder.lock.RLock()
	defer folder.lock.RUnlock()
	file, err := os.ReadFile(f.genPath(exp_id, path))
	if err != nil {
		// This means memory and file system are out of sync.
		if errors.Is(err, fs.ErrNotExist) {
			f.caches = make(map[int]*modelDefFolder)
			os.RemoveAll(f.rootDir)
			return f.GetFileContent(exp_id, path)
		} else {
			return []byte{}, err
		}
	}
	return file, err
}
