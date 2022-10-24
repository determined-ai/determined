package archive

import (
	"archive/tar"
	"archive/zip"
	"compress/gzip"
	"fmt"
	"io"
	"strings"
)

// ArchiveType currently includes tgz and zip.
type ArchiveType string

const (
	// ArchiveTgz is a gzipped tar ball.
	ArchiveTgz = "tgz"
	// ArchiveZip is a zip file.
	ArchiveZip = "zip"
	// ArchiveUnknown represents an unknown archive type.
	ArchiveUnknown = "unknown"
)

// ArchiveWriter defines an interface to create an archive file.
type ArchiveWriter interface {
	WriteHeader(path string, size int64) error
	Write(b []byte) (int, error)
	Close() error
}

// NewArchiveWriter returns a new ArchiveWriter for archiveType that writes to w.
func NewArchiveWriter(w io.Writer, archiveType ArchiveType) (ArchiveWriter, error) {
	closers := []io.Closer{}
	switch archiveType {
	case ArchiveTgz:
		gz := gzip.NewWriter(w)
		closers = append(closers, gz)

		tw := tar.NewWriter(gz)
		closers = append(closers, tw)

		return &tarArchiveWriter{archiveClosers{closers}, tw}, nil

	case ArchiveZip:
		zw := zip.NewWriter(w)
		closers = append(closers, zw)

		return &zipArchiveWriter{archiveClosers{closers}, zw, nil}, nil

	default:
		return nil, fmt.Errorf(
			"archive type must be %s or %s but got %s", ArchiveTgz, ArchiveZip, archiveType)
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

type tarArchiveWriter struct {
	archiveClosers
	tw *tar.Writer
}

func (aw *tarArchiveWriter) WriteHeader(path string, size int64) error {
	hdr := tar.Header{
		Name: path,
		Mode: 0o666,
		Size: size,
	}
	if strings.HasSuffix(path, "/") {
		// This a directory
		hdr.Mode = 0o777
	}
	return aw.tw.WriteHeader(&hdr)
}

func (aw *tarArchiveWriter) Write(p []byte) (int, error) {
	return aw.tw.Write(p)
}

type zipArchiveWriter struct {
	archiveClosers
	zw        *zip.Writer
	zwContent io.Writer
}

func (aw *zipArchiveWriter) WriteHeader(path string, size int64) error {
	// Zip by default sets mode 0666 and 0777 for files and folders respectively.
	zwc, err := aw.zw.Create(path)
	if err != nil {
		return err
	}
	aw.zwContent = zwc
	return nil
}

func (aw *zipArchiveWriter) Write(p []byte) (int, error) {
	// Guard against the mistake where WriteHeader() is not called before
	// calling Write(). The AWS SDK likely will not make this mistake but
	// zipArchiveWriter is not just limited to being used with AWS.
	if aw.zwContent == nil {
		return 0, nil
	}
	return aw.zwContent.Write(p)
}
