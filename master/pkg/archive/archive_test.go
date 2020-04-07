package archive

import (
	"archive/tar"
	"encoding/json"
	"os"
	"reflect"
	"testing"
	"time"

	"gotest.tools/assert"
)

func TestItem(t *testing.T) {
	type IntermediateExpected struct {
		Path    string `json:"path"`
		Type    byte   `json:"type"`
		Content string `json:"content"`
		Mode    int    `json:"mode"`
		Mtime   int64  `json:"mtime"`
		UID     int64  `json:"uid"`
		GID     int64  `json:"gid"`
	}

	intermediateExpected := IntermediateExpected{
		Path:    "/test",
		Type:    '0',
		Content: "b3JpZ2luYWw=",
		Mode:    0644,
		Mtime:   1501632000,
		UID:     0,
		GID:     0,
	}

	fileIn := RootItem("/test", []byte("original"), intermediateExpected.Mode, tar.TypeReg)
	marshaled, err := json.Marshal(fileIn)
	assert.NilError(t, err)

	// Confirm that we properly marshal into the intermediate types we expect, e.g., Content is
	// base64-encoded, ModifiedTime is in epoch seconds.
	t.Run("content is base64 and time is Unix time", func(t *testing.T) {
		var res IntermediateExpected
		err = json.Unmarshal(marshaled, &res)
		assert.NilError(t, err)
		assert.DeepEqual(t, intermediateExpected, res)
	})

	// Confirm that we can round-trip from Item to json to Item. The marshal/unmarshal of UnixTime
	// is lossy, so this only works with integer-second ModifiedTime values.
	t.Run("ArchiveItems can unmarshal properly", func(t *testing.T) {
		var fileResult Item
		err = json.Unmarshal(marshaled, &fileResult)
		assert.NilError(t, err)
		assert.DeepEqual(t, fileIn, fileResult)
	})
}

func TestByteStringMarshalText(t *testing.T) {
	type testCase struct {
		name     string
		b        byteString
		wantText []byte
		wantErr  bool
	}
	tests := []testCase{
		{
			name:     "test",
			b:        byteString("original"),
			wantText: []byte("b3JpZ2luYWw="),
			wantErr:  false,
		},
	}
	runTestCase := func(t *testing.T, tc testCase) {
		t.Run(tc.name, func(t *testing.T) {
			gotText, err := tc.b.MarshalText()
			if (err != nil) != tc.wantErr {
				t.Errorf("byteString.MarshalText() error = %v, wantErr %v",
					err, tc.wantErr)
				return
			}
			if !reflect.DeepEqual(gotText, tc.wantText) {
				t.Errorf("byteString.MarshalText() = %v, want %v", gotText, tc.wantText)
			}
		})
	}

	for _, tc := range tests {
		runTestCase(t, tc)
	}
}

func TestRoundtrip(t *testing.T) {
	archive := Archive{
		Item{
			Path:     "dir",
			Type:     tar.TypeDir,
			FileMode: os.FileMode(0644),
			ModifiedTime: UnixTime{
				Time: time.Unix(0, 0),
			},
			UserID:  0,
			GroupID: 0,
		},
		Item{
			Path:     "dir/b.txt",
			Type:     tar.TypeReg,
			FileMode: os.FileMode(0644),
			Content:  []byte("this is b"),
			ModifiedTime: UnixTime{
				Time: time.Unix(0, 0),
			},
			UserID:  0,
			GroupID: 0,
		},
		Item{
			Path:     "a.txt",
			Type:     tar.TypeReg,
			FileMode: os.FileMode(0644),
			Content:  []byte("this is a"),
			ModifiedTime: UnixTime{
				Time: time.Unix(0, 0),
			},
			UserID:  0,
			GroupID: 0,
		},
	}

	bytes, err := ToTarGz(archive)
	assert.NilError(t, err)

	roundTripArchive, err := FromTarGz(bytes)
	assert.NilError(t, err)

	assert.DeepEqual(t, archive, roundTripArchive)
}
