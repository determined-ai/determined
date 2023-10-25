//go:build integration
// +build integration

package logpattern

import (
	"testing"

	"github.com/determined-ai/determined/master/pkg/model"
	"github.com/determined-ai/determined/master/pkg/set"
)

// SetDisallowedNodesCacheTest is used only in tests to set expected blocklist for tasks.
func SetDisallowedNodesCacheTest(t *testing.T, c map[model.TaskID]*set.Set[string]) {
	defaultSingleton = &logPatternPolicies{
		blockListCache: c,
	}
}
