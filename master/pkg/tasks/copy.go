package tasks

import (
	"archive/tar"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/pkg/errors"

	"github.com/determined-ai/determined/master/pkg/archive"
	"github.com/determined-ai/determined/master/pkg/cproto"
	"github.com/determined-ai/determined/master/pkg/model"
	"github.com/determined-ai/determined/master/version"
)

const (
	// Regardless of where we're looking locally, the harness files we send around should go to this
	// path in task containers. The value must match where the entrypoint scripts look for wheels when
	// they run `pip install`.
	harnessTargetPath = "/opt/determined/wheels"
)

// normalizePythonVersion converts the given SemVer version into the equivalent normalized version
// as described by PEP 440 (which is what the filenames of built wheels will contain). In SemVer,
// the separators for the likes of `devN` and `rcN` is a hyphen, but PEP 440 specifies a period for
// `devN` and nothing for `rcN`.
func normalizePythonVersion(version string) string {
	version = strings.ReplaceAll(version, "-dev", ".dev")
	version = strings.ReplaceAll(version, "-rc", "rc")
	return version
}

func harnessArchive(harnessPath string, aug *model.AgentUserGroup) cproto.RunArchive {
	var harnessFiles archive.Archive
	validWhlNames := fmt.Sprintf("*%s*.whl", normalizePythonVersion(version.Version))
	wheelPaths, err := filepath.Glob(filepath.Join(harnessPath, validWhlNames))
	if err != nil {
		panic(errors.Wrapf(err, "error finding Python wheel files for version %s in path: %s",
			version.Version, harnessPath))
	}
	for _, path := range wheelPaths {
		info, err := os.Stat(path)
		if err != nil {
			panic(errors.Wrapf(err, "error retrieving stats for harness file: %s", path))
		}
		var content []byte
		content, err = os.ReadFile(path) // #nosec: G304
		if err != nil {
			panic(errors.Wrapf(err, "error reading harness file: %s", path))
		}
		rel, err := filepath.Rel(harnessPath, path)
		if err != nil {
			panic(errors.Wrapf(err, "error constructing relative path: %s", path))
		}

		var uid int
		if aug != nil {
			uid = aug.UID
		}
		var gid int
		if aug != nil {
			gid = aug.GID
		}

		harnessFiles = append(harnessFiles, archive.Item{
			Path:         filepath.Join(harnessTargetPath, rel),
			Type:         byte(tar.TypeReg),
			Content:      content,
			FileMode:     info.Mode(),
			ModifiedTime: archive.UnixTime{Time: info.ModTime()},
			UserID:       uid,
			GroupID:      gid,
		})
	}
	return wrapArchive(aug.OwnArchive(harnessFiles), "/")
}

func masterCertArchive(certBytes []byte) cproto.RunArchive {
	var arch archive.Archive
	if len(certBytes) != 0 {
		arch = append(arch, archive.RootItem(certPath, certBytes, 0o644, tar.TypeReg))
	}
	return wrapArchive(arch, "/")
}

func wrapArchive(archive archive.Archive, path string) cproto.RunArchive {
	return cproto.RunArchive{Path: path, Archive: archive}
}
