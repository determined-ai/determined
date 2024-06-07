package check

import (
	"testing"

	"github.com/stretchr/testify/require"
)

type testcase1 struct {
	A bool
}

func (t *testcase1) Validate() []error {
	return []error{
		True(t.A, "field A must be true"),
	}
}

type testcase2 struct {
	A bool
}

func (t testcase2) Validate() []error {
	return []error{
		True(t.A, "field A must be true"),
	}
}

func TestMethodSets(t *testing.T) {
	case1 := testcase1{
		A: false,
	}
	case2 := testcase2{
		A: false,
	}
	err := Validate(case1)
	require.ErrorContains(t, err, "error found at root: field A must be true: expected true, got false")
	err = Validate(&case1)
	require.ErrorContains(t, err, "error found at root: field A must be true: expected true, got false")
	err = Validate(case2)
	require.ErrorContains(t, err, "error found at root: field A must be true: expected true, got false")
	err = Validate(&case2)
	require.ErrorContains(t, err, "error found at root: field A must be true: expected true, got false")
}
