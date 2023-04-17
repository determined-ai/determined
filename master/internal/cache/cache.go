package cache

import (
	"context"
	"fmt"
	"io/fs"
	"net/http"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/pkg/errors"

	log "github.com/sirupsen/logrus"

	"google.golang.org/protobuf/types/known/timestamppb"

	"github.com/determined-ai/determined/master/internal/db"
	"github.com/determined-ai/determined/master/pkg/archive"
	"github.com/determined-ai/determined/proto/pkg/experimentv1"
)

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

// NewFileCache initialize FileCache obj.
func NewFileCache(rootDir string, maxAge time.Duration) *FileCache {
	err := os.RemoveAll(rootDir)
	if err != nil {
		log.WithError(err).Errorf("failed to clear the content of cache folder at %s", rootDir)
	}
	return &FileCache{
		rootDir: rootDir,
		maxAge:  maxAge,
		caches:  make(map[int]*modelDefFolder),
	}
}

func (f *FileCache) getOrCreateFolder(expID int) (*modelDefFolder, error) {
	f.lock.Lock()
	defer f.lock.Unlock()
	value, ok := f.caches[expID]
	if ok {
		return value, nil
	}

	exp := struct {
		ModelDefinition []byte
	}{}
	err := db.Bun().NewSelect().TableExpr(
		"experiments").Column("model_definition").Where("id = ?", expID).Scan(context.Background(), &exp)
	if err != nil {
		return nil, fmt.Errorf("error getting model definition for experiment id %d: %w", expID, err)
	}
	var fileTree []*experimentv1.FileNode
	arc, err := archive.FromTarGz(exp.ModelDefinition)
	if err != nil {
		return nil, fmt.Errorf("error creating tar archive for experiment id %d: %w", expID, err)
	}
	err = os.MkdirAll(f.genPath(expID, ""), 0o700)
	if err != nil {
		return nil, fmt.Errorf("error creating cache directory for experiment id %d: %w", expID, err)
	}
	for _, ar := range arc {
		path, err := f.genPathWithValidation(expID, ar.Path)
		if err != nil {
			return nil, err
		}
		if ar.IsDir() {
			err = os.MkdirAll(path, 0o700)
		} else {
			err = os.WriteFile(path, ar.Content, 0o600)
		}
		if err != nil {
			return nil, fmt.Errorf(
				"error adding archive file or directory for experiment id %d: %w", expID, err)
		}
		fileTree = append(fileTree, &experimentv1.FileNode{
			Path:          ar.Path,
			ModifiedTime:  timestamppb.New(ar.ModifiedTime.Time),
			ContentLength: int32(len(ar.Content)),
			IsDir:         ar.IsDir(),
			ContentType:   http.DetectContentType(ar.Content),
			Name:          filepath.Base(ar.Path),
		})
	}
	value = &modelDefFolder{
		path:       f.genPath(expID, ""),
		fileTree:   fileTree,
		cachedTime: time.Now(),
	}
	f.caches[expID] = value
	f.prune()

	return value, nil
}

// prune is not locked because it's only meant to be triggered inside getOrCreateFolder.
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

// genPathWithValidation checks if given path is under cache directory
// by checking if the relative path of given path to cache directory
// refer to parent directory. This is to aviod paths in tarball
// are tempting to affect file system outside of cache directory.
func (f *FileCache) genPathWithValidation(expID int, path string) (string, error) {
	p := f.genPath(expID, path)
	rp, err := filepath.Rel(f.genPath(expID, ""), p)
	if err != nil {
		return "", fmt.Errorf("error getting relative path for experiment id %d cache: %w", expID, err)
	}
	if strings.HasPrefix(rp, "..") {
		return "", errors.Errorf("%s is not a valid path.", path)
	}
	return p, nil
}

func (f *FileCache) genPath(expID int, path string) string {
	return filepath.Join(f.rootDir, strconv.Itoa(expID), path)
}

func (f *FileCache) getFileTree(expID int) ([]*experimentv1.FileNode, *modelDefFolder, error) {
	folder, err := f.getOrCreateFolder(expID)
	if err != nil {
		return nil, nil, err
	}
	folder.lock.RLock()
	defer folder.lock.RUnlock()
	return folder.fileTree, folder, nil
}

// FileTreeNested returns folder tree structure with given experiment id.
func (f *FileCache) FileTreeNested(expID int) ([]*experimentv1.FileNode, error) {
	fileTree, _, err := f.getFileTree(expID)
	if err != nil {
		return nil, err
	}
	return genNestedTree(fileTree), nil
}

// FileContent returns file with given experiment id and path.
func (f *FileCache) FileContent(expID int, path string) ([]byte, error) {
	fileTree, folder, err := f.getFileTree(expID)
	if err != nil {
		return []byte{}, err
	}
	for _, file := range fileTree {
		if file.Path == path && !file.IsDir {
			folder.lock.Lock()
			defer folder.lock.Unlock()
			file, err := os.ReadFile(f.genPath(expID, path))
			if err != nil {
				_, ok := err.(*fs.PathError)
				if ok {
					log.Errorf(`File system cache (%s) is likely out of sync. 
File system cache is about to re-initialize.`,
						f.rootDir)
					return f.fileContentAfterReset(expID, path)
				}
				return []byte{}, fmt.Errorf(
					"error reading file cache content for experiment id %d: %w", expID, err)
			}
			return file, nil
		}
	}
	return nil, fs.ErrNotExist
}

func (f *FileCache) fileContentAfterReset(expID int, path string) ([]byte, error) {
	err := f.resetCache(expID)
	if err != nil {
		return []byte{}, err
	}
	_, folder, err := f.getFileTree(expID)
	if err != nil {
		return []byte{}, err
	}
	folder.lock.Lock()
	defer folder.lock.Unlock()

	b, err := os.ReadFile(f.genPath(expID, path))
	if err != nil {
		return []byte{}, fmt.Errorf(
			"error reading cache file content after reset for experiment id %d: %w", expID, err)
	}
	return b, nil
}

func (f *FileCache) resetCache(expID int) error {
	f.lock.Lock()
	defer f.lock.Unlock()
	delete(f.caches, expID)

	if err := os.RemoveAll(f.genPath(expID, "")); err != nil {
		return fmt.Errorf("error reseting cache for experiment id %d: %w", expID, err)
	}
	return nil
}

// This function assumes fileTree is a valid input generated from file system.
// Which means all nodes are presented, and parent folder comes before child.
func genNestedTree(fileTree []*experimentv1.FileNode) []*experimentv1.FileNode {
	var fileTreeNested []*experimentv1.FileNode
	for _, file := range fileTree {
		fileTreeNested = insertToTree(
			fileTreeNested, strings.Split(file.Path, string(os.PathSeparator)), file)
	}
	return fileTreeNested
}

func insertToTree(
	root []*experimentv1.FileNode, paths []string, node *experimentv1.FileNode,
) []*experimentv1.FileNode {
	if len(paths) > 0 {
		var i int
		for i = 0; i < len(root); i++ {
			if root[i].Name == paths[0] {
				break
			}
		}
		if i == len(root) {
			root = append(root, node)
		}
		root[i].Files = insertToTree(root[i].Files, paths[1:], node)
	}
	return root
}
