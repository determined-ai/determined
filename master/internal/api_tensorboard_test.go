package internal

import (
	"testing"

	"gotest.tools/assert"
)

func TestRefineArgs(t *testing.T) {
	args := []string{
		"first",
		"-second",
		"--third",
		"-fourth-one",
		"-----fifth",
		"h",
		"-h",
		"--h",
		"------h",
	}

	expected := []string{
		"--first",
		"--second",
		"--third",
		"--fourth-one",
		"--fifth",
		"-h",
		"-h",
		"-h",
		"-h",
	}
	refineArgs(args)

	assert.DeepEqual(t, args, expected)
}
