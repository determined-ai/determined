package archive

import (
	"compress/gzip"
	"fmt"
	"io"
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
