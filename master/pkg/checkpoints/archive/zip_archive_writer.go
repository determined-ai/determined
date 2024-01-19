package archive

import (
	"archive/zip"
	"io"

	"github.com/pkg/errors"
)

type zipArchiveWriter struct {
	archiveClosers
	zw        *zip.Writer
	zwContent io.Writer
}

func newZipArchiveWriter(w io.Writer, closers []io.Closer) *zipArchiveWriter {
	zw := zip.NewWriter(w)
	closers = append(closers, zw)
	return &zipArchiveWriter{archiveClosers{closers}, zw, nil}
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

func (aw *zipArchiveWriter) DryRunEnabled() bool {
	return false
}

func (aw *zipArchiveWriter) DryRunLength(path string, size int64) (int64, error) {
	return 0, errors.New("dry run not supported for zip")
}

func (aw *zipArchiveWriter) DryRunClose() (int64, error) {
	return 0, errors.New("dry run not supported for zip")
}
