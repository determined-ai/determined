package tasks

import (
	"archive/tar"
	"io/ioutil"
	"os"
	"path/filepath"

	"github.com/pkg/errors"

	"github.com/determined-ai/determined/master/pkg/archive"
	"github.com/determined-ai/determined/master/pkg/container"
)

const (
	// Regardless of where we're looking locally, the harness files we send around should go to this
	// path in task containers. The value must match where the entrypoint scripts look for wheels when
	// they run `pip install`.
	harnessTargetPath = "/opt/determined/wheels"
)

func harnessArchive(harnessPath string) container.RunArchive {
	var harnessFiles archive.Archive
	wheelPaths, err := filepath.Glob(filepath.Join(harnessPath, "*.whl"))
	if err != nil {
		panic(errors.Wrapf(err, "error finding Python wheel files in path: %s", harnessPath))
	}
	for _, path := range wheelPaths {
		info, err := os.Stat(path)
		if err != nil {
			panic(errors.Wrapf(err, "error retrieving stats for harness file: %s", path))
		}
		var content []byte
		content, err = ioutil.ReadFile(path) // #nosec: G304
		if err != nil {
			panic(errors.Wrapf(err, "error reading harness file: %s", path))
		}
		rel, err := filepath.Rel(harnessPath, path)
		if err != nil {
			panic(errors.Wrapf(err, "error constructing relative path: %s", path))
		}

		harnessFiles = append(harnessFiles, archive.Item{
			Path:         filepath.Join(harnessTargetPath, rel),
			Type:         byte(tar.TypeReg),
			Content:      content,
			FileMode:     info.Mode(),
			ModifiedTime: archive.UnixTime{Time: info.ModTime()},
		})
	}
	return wrapArchive(harnessFiles, "/")
}

func wrapArchive(archive archive.Archive, path string) container.RunArchive {
	return container.RunArchive{Path: path, Archive: archive}
}
