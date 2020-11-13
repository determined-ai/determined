package model

import (
	"testing"

	"github.com/determined-ai/determined/master/pkg/check"
)

func TestSharedFSConfigValidate(t *testing.T) {
	type fields struct {
		HostPath    string
		StoragePath *string
	}
	type testCase struct {
		name            string
		fields          fields
		wantErr         bool
		pathInContainer string
	}

	storage1 := "/host_path/storage"
	storage2 := "storage"
	storage3 := "/storage"
	storage4 := "/host_path/../sneaky"
	storage5 := "../sneaky"

	tests := []testCase{
		{
			name: "valid no storage_path",
			fields: fields{
				HostPath: "/host_path",
			},
			pathInContainer: "/determined_shared_fs",
		},
		{
			name: "valid absolute storage_path",
			fields: fields{
				HostPath:    "/host_path",
				StoragePath: &storage1,
			},
			pathInContainer: "/determined_shared_fs/storage",
		},
		{
			name: "valid relative storage_path",
			fields: fields{
				HostPath:    "/host_path",
				StoragePath: &storage2,
			},
			pathInContainer: "/determined_shared_fs/storage",
		},
		{
			name: "invalid relative host_path",
			fields: fields{
				HostPath: "host_path",
			},
			wantErr: true,
		},
		{
			name: "invalid absolute storage path",
			fields: fields{
				HostPath:    "/host_path",
				StoragePath: &storage3,
			},
			wantErr: true,
		},
		{
			name: "sneaky absolute storage path",
			fields: fields{
				HostPath:    "/host_path",
				StoragePath: &storage4,
			},
			wantErr: true,
		},
		{
			name: "sneaky relative storage path",
			fields: fields{
				HostPath:    "/host_path",
				StoragePath: &storage5,
			},
			wantErr: true,
		},
	}
	runTestCase := func(t *testing.T, tc testCase) {
		t.Run(tc.name, func(t *testing.T) {
			c := &SharedFSConfig{
				HostPath:    tc.fields.HostPath,
				StoragePath: tc.fields.StoragePath,
			}
			if err := check.Validate(c); (err != nil) != tc.wantErr {
				t.Errorf("config.Validate() error = %v, wantErr %v", err, tc.wantErr)
			}
			if !tc.wantErr && c.PathInContainer() != tc.pathInContainer {
				t.Errorf(
					"config.PathInContainer() return = %v, expected %v",
					c.PathInContainer(),
					tc.pathInContainer,
				)
			}
		})
	}

	for _, tc := range tests {
		runTestCase(t, tc)
	}
}
