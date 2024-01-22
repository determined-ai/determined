package archive

import (
	"archive/tar"
	"bytes"
	"io"
	"strings"

	"github.com/pkg/errors"
)

type tarArchiveWriter struct {
	archiveClosers
	tw     *tar.Writer
	dry    *tar.Writer
	dryBuf *bytes.Buffer
}

func newTarArchiveWriter(w io.Writer, closers []io.Closer) *tarArchiveWriter {
	tw := tar.NewWriter(w)
	closers = append(closers, tw)
	return &tarArchiveWriter{archiveClosers{closers}, tw, nil, nil}
}

func tarHeader(path string, size int64, typeflag byte) *tar.Header {
	hdr := &tar.Header{
		Name: path,
		Mode: 0o666,
		Size: size,
	}
	if strings.HasSuffix(path, "/") {
		// This a directory
		hdr.Mode = 0o777
		hdr.Typeflag = tar.TypeDir
	} else {
		hdr.Typeflag = typeflag
	}
	return hdr
}

// tarPadding returns the number of bytes needed to pad for a block.
func tarPadding(size int64) (n int64) {
	// binary math trick used in tar implementation.
	return -size & 511
}

func (aw *tarArchiveWriter) enableDryRun() *tarArchiveWriter {
	if aw.dry != nil && aw.dryBuf != nil {
		return aw
	}
	aw.dryBuf = &bytes.Buffer{}
	aw.dry = tar.NewWriter(aw.dryBuf)
	aw.closers = append(aw.closers, aw.dry)
	return aw
}

func (aw *tarArchiveWriter) WriteHeader(path string, size int64) error {
	return aw.tw.WriteHeader(tarHeader(path, size, tar.TypeReg))
}

func (aw *tarArchiveWriter) Write(p []byte) (int, error) {
	return aw.tw.Write(p)
}

func (aw *tarArchiveWriter) DryRunEnabled() bool {
	return aw.dry != nil && aw.dryBuf != nil
}

func (aw *tarArchiveWriter) DryRunLength(path string, size int64) (int64, error) {
	if aw.dry == nil || aw.dryBuf == nil {
		return 0, errors.New("dry run not enabled")
	}
	// Write the header as tar.TypeLink to avoid writing the file contents.
	if err := aw.dry.WriteHeader(tarHeader(path, size, tar.TypeLink)); err != nil {
		return 0, err
	}
	// Write the header contents to the buffer.
	if err := aw.dry.Flush(); err != nil {
		return 0, err
	}
	// Size for this file is the size of the header plus the size of the padded contents.
	totalSize := int64(aw.dryBuf.Len()) + size + tarPadding(size)
	// Clear the buffer for the next file.
	aw.dryBuf.Reset()
	return totalSize, nil
}

func (aw *tarArchiveWriter) DryRunClose() (int64, error) {
	if aw.dry == nil || aw.dryBuf == nil {
		return 0, errors.New("dry run not enabled")
	}
	if err := aw.dry.Close(); err != nil {
		return 0, err
	}
	size := int64(aw.dryBuf.Len())
	aw.dryBuf.Reset()
	return size, nil
}
