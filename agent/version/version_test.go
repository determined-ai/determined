package version

import (
	"testing"

	"gotest.tools/assert"
)

func TestVersion(t *testing.T) {
	assert.Assert(t, Version == "unknown")
}
