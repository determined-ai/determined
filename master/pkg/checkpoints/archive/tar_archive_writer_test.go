package archive

import (
	"archive/tar"
	"bytes"
	"io"
	"math"
	"strings"
	"testing"

	"github.com/docker/go-units"
	"github.com/stretchr/testify/require"
)

type testIoReaderWriter struct {
	bytes.Buffer
	readCount  int64
	writeCount int64
}

func (t *testIoReaderWriter) Read(b []byte) (n int, err error) {
	count, err := t.Buffer.Read(b)
	t.readCount += int64(count)
	return count, err
}

func (t *testIoReaderWriter) Write(p []byte) (n int, err error) {
	count, err := t.Buffer.Write(p)
	t.writeCount += int64(count)
	return count, err
}

func TestSimpleTar(t *testing.T) {
	var buf testIoReaderWriter
	aw, err := NewArchiveWriter(&buf, ArchiveTar)
	require.NoError(t, err)
	require.NoError(t, aw.WriteHeader("foo", 3))
	size, err := aw.Write([]byte("bar"))
	require.NoError(t, err)
	require.Equal(t, 3, size)
	require.NoError(t, aw.Close())

	// Check the tar file.
	tr := tar.NewReader(&buf)
	hdr, err := tr.Next()
	require.NoError(t, err)
	require.Equal(t, "foo", hdr.Name)
	require.Equal(t, int64(3), hdr.Size)
	require.Equal(t, byte(tar.TypeReg), hdr.Typeflag)
	result := make([]byte, 3)
	size, err = tr.Read(result)
	require.Equal(t, io.EOF, err)
	require.Equal(t, 3, size)
	require.Equal(t, "bar", string(result))
	size, err = tr.Read(result)
	require.Equal(t, io.EOF, err)
	require.Equal(t, 0, size)
	_, err = tr.Next()
	require.Equal(t, io.EOF, err)
	require.Equal(t, buf.writeCount, buf.readCount)
}

func TestTarDryRun(t *testing.T) {
	var buf testIoReaderWriter
	tr := tar.NewReader(&buf)
	aw, err := NewArchiveWriter(&buf, ArchiveTar)
	require.NoError(t, err)

	entries := []FileEntry{
		{
			Path: strings.Repeat("a", 10),
			Size: int64(math.Pow(2, 4)) + 1,
		},
		{
			Path: strings.Repeat("a", 100),
			Size: int64(math.Pow(2, 32)) + 2,
		},
		// this entry should trigger an extended header
		{
			Path: strings.Repeat("a", 1000),
			Size: int64(math.Pow(2, 34)) + 3,
		},
	}

	sampleChunk := bytes.Repeat([]byte("12345678"), units.MiB)
	maxChunkSize := int64(len(sampleChunk))

	contentLength, err := DryRunLength(aw, entries)
	require.NoError(t, err)

	for _, entry := range entries {
		err := aw.WriteHeader(entry.Path, entry.Size)
		require.NoError(t, err)

		hdr, err := tr.Next()
		require.NoError(t, err)
		require.Equal(t, entry.Path, hdr.Name)
		require.Equal(t, entry.Size, hdr.Size)

		for i := int64(0); i < entry.Size; i += maxChunkSize {
			chunkSize := entry.Size - i
			if chunkSize > maxChunkSize {
				chunkSize = maxChunkSize
			}
			_, err := aw.Write(sampleChunk[:chunkSize])
			require.NoError(t, err)
			require.Equal(t, chunkSize, int64(buf.Len()))

			readChunk := make([]byte, chunkSize)
			readSize, err := tr.Read(readChunk)
			if i+chunkSize < entry.Size {
				require.NoError(t, err)
			} else {
				require.Equal(t, io.EOF, err)
			}
			require.Equal(t, chunkSize, int64(readSize))
			require.Equal(t, sampleChunk[:chunkSize], readChunk[:chunkSize])
		}
	}

	require.NoError(t, aw.Close())

	_, err = tr.Next()
	require.Equal(t, io.EOF, err)
	require.Equal(t, buf.Len(), 0)
	require.Equal(t, buf.writeCount, buf.readCount)
	require.Equal(t, buf.writeCount, contentLength)
}
