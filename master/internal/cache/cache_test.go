package cache

import (
	"reflect"
	"testing"
	"time"

	"github.com/stretchr/testify/require"

	"github.com/determined-ai/determined/proto/pkg/experimentv1"
)

func TestNestedFile(t *testing.T) {
	type testStruct struct {
		description    string
		fileTree       []*experimentv1.FileNode
		nestedFileTree []*experimentv1.FileNode
	}
	tests := []testStruct{
		{
			description: "#1 flat",
			fileTree: []*experimentv1.FileNode{
				{
					Path: "a",
					Name: "a",
				}, {
					Path: "b",
					Name: "b",
				}, {
					Path: "c",
					Name: "c",
				},
			},
			nestedFileTree: []*experimentv1.FileNode{
				{
					Path: "a",
					Name: "a",
				}, {
					Path: "b",
					Name: "b",
				}, {
					Path: "c",
					Name: "c",
				},
			},
		}, {
			description: "#2 nested",
			fileTree: []*experimentv1.FileNode{
				{
					Path: "a",
					Name: "a",
				}, {
					Path: "b",
					Name: "b",
				}, {
					Path: "b/c",
					Name: "c",
				},
			},
			nestedFileTree: []*experimentv1.FileNode{
				{
					Path: "a",
					Name: "a",
				}, {
					Path: "b",
					Name: "b",
					Files: []*experimentv1.FileNode{
						{
							Path: "b/c",
							Name: "c",
						},
					},
				},
			},
		}, {
			description: "#3 deeper nested",
			fileTree: []*experimentv1.FileNode{
				{
					Path: "a",
					Name: "a",
				}, {
					Path: "a/b",
					Name: "b",
				}, {
					Path: "a/b/c",
					Name: "c",
				},
			},
			nestedFileTree: []*experimentv1.FileNode{
				{
					Path: "a",
					Name: "a",
					Files: []*experimentv1.FileNode{{
						Path: "a/b",
						Name: "b",
						Files: []*experimentv1.FileNode{
							{
								Path: "a/b/c",
								Name: "c",
							},
						},
					}},
				},
			},
		}, {
			description: "#4 multiple nested with different depth",
			fileTree: []*experimentv1.FileNode{
				{
					Path: "a",
					Name: "a",
				}, {
					Path: "a/b",
					Name: "b",
				}, {
					Path: "c",
					Name: "c",
				}, {
					Path: "c/d",
					Name: "d",
				}, {
					Path: "c/d/e",
					Name: "e",
				}, {
					Path: "c/d/f",
					Name: "f",
				},
			},
			nestedFileTree: []*experimentv1.FileNode{
				{
					Path: "a",
					Name: "a",
					Files: []*experimentv1.FileNode{{
						Path: "a/b",
						Name: "b",
					}},
				}, {
					Path: "c",
					Name: "c",
					Files: []*experimentv1.FileNode{{
						Path: "c/d",
						Name: "d",
						Files: []*experimentv1.FileNode{{
							Path: "c/d/e",
							Name: "e",
						}, {
							Path: "c/d/f",
							Name: "f",
						}},
					}},
				},
			},
		},
	}

	for _, test := range tests {
		nestedFileTree := genNestedTree(test.fileTree)
		require.Truef(t, reflect.DeepEqual(nestedFileTree, test.nestedFileTree),
			"Failed test %s \nGot: %+v\nExpected: %+v",
			test.description, nestedFileTree, test.nestedFileTree)
	}
}

func TestGenPathWithValidation(t *testing.T) {
	testExpID := 1
	type testStruct struct {
		description string
		path        string
		expected    string
		hasErr      bool
	}
	tests := []testStruct{
		{
			description: "#1 normal case",
			path:        "a/b",
			expected:    "/tmp/determined-cache/1/a/b",
			hasErr:      false,
		}, {
			description: "#2 invalid case",
			path:        "../a",
			expected:    "",
			hasErr:      true,
		}, {
			description: "#3 invalid case",
			path:        "../../a",
			expected:    "",
			hasErr:      true,
		},
	}
	f := NewFileCache("/tmp/determined-cache", 2*time.Hour)
	for _, test := range tests {
		p, err := f.genPathWithValidation(testExpID, test.path)
		if test.hasErr {
			require.Errorf(t, err, test.description)
		} else {
			require.Equalf(t, p, test.expected, test.description)
		}
	}
}
