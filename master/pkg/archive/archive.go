package archive

import (
	"archive/tar"
	"bytes"
	"compress/gzip"
	"encoding/base64"
	"encoding/json"
	"io"
	"io/ioutil"
	"os"
	"path"
	"time"

	"github.com/determined-ai/determined/master/pkg"
)

type (
	// Archive contains an ordered list of Item objects, much like a tarball.
	Archive []Item
)

var defaultModifiedTime UnixTime = func() UnixTime {
	t, err := time.Parse(time.RFC3339, pkg.DeterminedBirthday)
	if err != nil {
		panic(err)
	}
	return UnixTime{Time: t}
}()

// Item is an in-memory representation of a file.  It contains the content and additional metadata
// of a file.
type Item struct {
	// Path should include the filename.  For directories, it should not end in a '/'.
	Path string `json:"path"`
	// Type should match the tar.Header.Typeflag values.
	Type         byte        `json:"type"`
	Content      byteString  `json:"content"`
	FileMode     os.FileMode `json:"mode"`
	ModifiedTime UnixTime    `json:"mtime"`
	UserID       int         `json:"uid"`
	GroupID      int         `json:"gid"`
}

// BaseName returns the base name of the file.
func (i *Item) BaseName() string {
	return path.Base(i.Path)
}

// DirName returns the directory name of the file.
func (i *Item) DirName() string {
	return path.Dir(i.Path)
}

// IsDir returns if the file is a directory.
func (i *Item) IsDir() bool {
	return i.Type == tar.TypeDir
}

// ContainsPath returns if Item with the exact path given is present in an Archive.
func (ar Archive) ContainsPath(path string) bool {
	for _, file := range ar {
		if file.Path == path {
			return true
		}
	}
	return false
}

// RootItem returns a new Item which will be owned by root when embedded in a container.
func RootItem(path string, content []byte, mode int, fileType byte) Item {
	return Item{
		Path:         path,
		Content:      content,
		FileMode:     os.FileMode(mode),
		Type:         fileType,
		ModifiedTime: defaultModifiedTime,
	}
}

// UserItem returns a new Item which will be owned by the user under which the container runs.
func UserItem(path string, content []byte, mode int, fileType byte, userID int, groupID int) Item {
	return Item{
		Path:         path,
		Content:      content,
		FileMode:     os.FileMode(mode),
		Type:         fileType,
		ModifiedTime: defaultModifiedTime,
		UserID:       userID,
		GroupID:      groupID,
	}
}

// byteString marshals to a base64 encoded string for the content of the file.
type byteString []byte

// UnmarshalText implements the encoding.TextUnmarshaler interface.
func (b *byteString) UnmarshalText(text []byte) (err error) {
	*b, err = base64.StdEncoding.DecodeString(string(text))
	return err
}

// MarshalText implements the encoding.TextMarshaler interface.
func (b byteString) MarshalText() (text []byte, err error) {
	return []byte(base64.StdEncoding.EncodeToString(b)), nil
}

// UnixTime is a wrapper around time.Time that serializes to a Unix timestamp.
// This is a lossy conversion because time.Time also contains time zone and
// nanosecond information, so this type is only appropriate for dealing with
// legacy systems.
type UnixTime struct {
	time.Time
}

// MarshalJSON marshals a UnixTime as seconds since the epoch.
func (t UnixTime) MarshalJSON() ([]byte, error) {
	ts := t.Unix()
	return json.Marshal(ts)
}

// UnmarshalJSON unmarshals seconds since the epoch into a UnixTime.
func (t *UnixTime) UnmarshalJSON(data []byte) error {
	var ts int64
	if err := json.Unmarshal(data, &ts); err != nil {
		return err
	}
	t.Time = time.Unix(ts, 0)
	return nil
}

// Writes the archive as a tarfile to the given Writer.
func tarArchive(ar Archive, writer io.Writer) error {
	w := tar.NewWriter(writer)

	for _, item := range ar {
		if err := w.WriteHeader(&tar.Header{
			Typeflag: item.Type,
			Name:     item.Path,
			Mode:     int64(item.FileMode),
			Size:     int64(len(item.Content)),
			Uid:      item.UserID,
			Gid:      item.GroupID,
			ModTime:  item.ModifiedTime.Time,
		}); err != nil {
			return err
		}
		if _, err := io.Copy(w, bytes.NewBuffer(item.Content)); err != nil {
			return err
		}
	}

	return w.Close()
}

// ToIOReader converts the files in an Archive to an io.Reader bytes buffer.
func ToIOReader(ar Archive) (io.Reader, error) {
	var buf bytes.Buffer

	if err := tarArchive(ar, &buf); err != nil {
		return nil, err
	}

	return &buf, nil
}

// ToTarGz converts the files in an Archive into a gzipped tarfile.
func ToTarGz(ar Archive) ([]byte, error) {
	var buf bytes.Buffer

	gzipWriter := gzip.NewWriter(&buf)

	if err := tarArchive(ar, gzipWriter); err != nil {
		return nil, err
	}

	if err := gzipWriter.Close(); err != nil {
		return nil, err
	}

	return buf.Bytes(), nil
}

// FromTarGz converts a gzipped tarfile (in bytes) to an Archive.
func FromTarGz(zippedTarfile []byte) (Archive, error) {
	byteReader := bytes.NewReader(zippedTarfile)
	gzipReader, err := gzip.NewReader(byteReader)
	if err != nil {
		return nil, err
	}

	tarReader := tar.NewReader(gzipReader)

	var ar Archive
	for {
		header, err := tarReader.Next()

		if err == io.EOF {
			break
		}

		if err != nil {
			return nil, err
		}

		item := Item{
			Path:     header.Name,
			Type:     header.Typeflag,
			FileMode: os.FileMode(header.Mode),
			ModifiedTime: UnixTime{
				Time: header.ModTime,
			},
			UserID:  header.Uid,
			GroupID: header.Gid,
		}

		if header.Typeflag == tar.TypeReg {
			var err error
			item.Content, err = ioutil.ReadAll(tarReader)
			if err != nil {
				return nil, err
			}
		}

		ar = append(ar, item)
	}

	return ar, nil
}
