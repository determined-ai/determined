package internal

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestGenerateNamespaceName(t *testing.T) {
	testCases := []struct {
		name string

		instNamespace string
		clusterID     string
		workspaceName string
		workspaceID   uint

		expectedName string
	}{{
		name:          "empty cluster id",
		instNamespace: "foo",
		clusterID:     "",
		workspaceName: "bar",
		workspaceID:   42,
		expectedName:  "bar-foo--42",
	}, {
		name:          "empty workspace name",
		instNamespace: "foo",
		clusterID:     "bar",
		workspaceName: "",
		workspaceID:   742_915,
		expectedName:  "-foo-bar-742915",
	}, {
		name:          "empty install namespace",
		instNamespace: "",
		clusterID:     "bar",
		workspaceName: "foo",
		workspaceID:   1,
		expectedName:  "foo--bar-1",
	}, {
		name:          "long values",
		instNamespace: "thisisanextremelylonginstallnamespacevaluetotesthowthisishandled",
		clusterID:     "thisisanextremelylongclusteridvaluetotesthowthisishandled",
		workspaceName: "thisisanextremelylongworkspacenamevaluetotesthowthisishandled",
		workspaceID:   9001,
		expectedName:  "thisisanextremelylongworkspacen-thisisanextremelylo-this-9001",
	}, {
		name:          "large workspace id",
		instNamespace: "foo",
		clusterID:     "bar",
		workspaceName: "baz",
		workspaceID:   8_675_309,
		expectedName:  "baz-foo-bar-867530",
	}, {
		name:          "invalid characters stripped out of workspace name",
		instNamespace: "foo",
		clusterID:     "bar",
		workspaceName: "💪🏼baz",
		workspaceID:   3,
		expectedName:  "baz-foo-bar-3",
	}}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			a := assert.New(t)

			actual := generateNamespaceName(tc.instNamespace, tc.clusterID, tc.workspaceName, tc.workspaceID)
			a.Equal(tc.expectedName, actual)

			// Kubernetes namespace name length cannot exceed 63 characters, so do an extra check for that.
			a.LessOrEqual(len(actual), 63)
		})
	}
}
