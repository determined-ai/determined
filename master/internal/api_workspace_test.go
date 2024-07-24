package internal

import (
	"regexp"
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
		expectedName:  "null-foo-bar-742915",
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

	k8sNamespaceConstraint := regexp.MustCompile("^[a-z0-9]([-a-z0-9]*[a-z0-9])?$")
	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			a := assert.New(t)

			actual := generateNamespaceName(tc.workspaceName, tc.instNamespace, tc.clusterID, tc.workspaceID)
			a.Equal(tc.expectedName, actual)

			// Kubernetes namespace name length cannot exceed 63 characters, so do an extra check for that.
			a.LessOrEqual(len(actual), 63, "generated name was too long to be a valid k8s namespace name")
			// Kubernetes namespace name must match the above regex, so also check that.
			a.Regexp(k8sNamespaceConstraint, actual, "generated name was not a valid k8s namespace name")
		})
	}
}
