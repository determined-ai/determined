package archive

import (
	"compress/gzip"
	"fmt"
	"io"

	"github.com/pkg/errors"
)

// ArchiveType currently includes tgz and zip.
type ArchiveType string

const (
	// ArchiveTar is a tar ball.
	ArchiveTar = "tar"
	// ArchiveTgz is a gzipped tar ball.
	ArchiveTgz = "tgz"
	// ArchiveZip is a zip file.
	ArchiveZip = "zip"
	// ArchiveUnknown represents an unknown archive type.
	ArchiveUnknown = "unknown"
)

// FileEntry represents a file in an archive.
type FileEntry struct {
	// Path is the path of the file in the archive.
	Path string
	// Size is the size of the file in bytes.
	Size int64
}

// ArchiveWriter defines an interface to create an archive file.
type ArchiveWriter interface {
	WriteHeader(path string, size int64) error
	Write(b []byte) (int, error)
	Close() error
	DryRunEnabled() bool
	DryRunLength(path string, size int64) (int64, error)
	DryRunClose() (int64, error)
}

// NewArchiveWriter returns a new ArchiveWriter for archiveType that writes to w.
func NewArchiveWriter(w io.Writer, archiveType ArchiveType) (ArchiveWriter, error) {
	closers := []io.Closer{}
	switch archiveType {
	case ArchiveTar:
		return newTarArchiveWriter(w, closers).enableDryRun(), nil

	case ArchiveTgz:
		gz := gzip.NewWriter(w)
		closers = append(closers, gz)
		return newTarArchiveWriter(gz, closers), nil

	case ArchiveZip:
		return newZipArchiveWriter(w, closers), nil

	default:
		return nil, fmt.Errorf(
			"archive type must be %s, %s, or %s. received %s", ArchiveTar, ArchiveTgz, ArchiveZip, archiveType)
	}
}

type archiveClosers struct {
	closers []io.Closer
}

// Close() closes all items in closers in reverse order.
func (ac *archiveClosers) Close() error {
	for i := len(ac.closers) - 1; i >= 0; i-- {
		err := ac.closers[i].Close()
		if err != nil {
			return err
		}
	}
	return nil
}

// DryRunLength returns the length of the archive file that would be created if the files were
// written to the archive.
func DryRunLength(
	aw ArchiveWriter,
	files []FileEntry,
) (int64, error) {
	if !aw.DryRunEnabled() {
		return 0, errors.New("dry run not enabled")
	}
	contentLength := int64(0)
	for _, file := range files {
		size, err := aw.DryRunLength(file.Path, file.Size)
		if err != nil {
			return 0, err
		}
		contentLength += size
	}
	closeSize, err := aw.DryRunClose()
	if err != nil {
		return 0, err
	}
	return contentLength + closeSize, nil
}
