package schemas

import (
	"testing"

	"gotest.tools/assert"

	"github.com/determined-ai/determined/master/pkg/ptrs"
)

type X struct {
	A *string
	B *string
	C *string
}

func TestMerge(t *testing.T) {
	obj := X{
		A: ptrs.StringPtr("obj:x.a"),
		B: nil,
		C: ptrs.StringPtr("obj:x.c"),
	}

	src := X{
		A: ptrs.StringPtr("src:x.a"),
		B: ptrs.StringPtr("src:x.b"),
		C: nil,
	}

	out := Merge(obj, src).(X)

	assert.Assert(t, *out.A == "obj:x.a")
	assert.Assert(t, *out.B == "src:x.b")
	assert.Assert(t, *out.C == "obj:x.c")
}

func TestMapMerge(t *testing.T) {
	assertCorrectMerge := func(result map[string]string) {
		assert.Assert(t, len(result) == 3)
		assert.Assert(t, result["1"] == "obj:one")
		assert.Assert(t, result["2"] == "obj:two")
		assert.Assert(t, result["3"] == "src:three")
	}

	obj := map[string]string{"1": "obj:one", "2": "obj:two"}
	src := map[string]string{"2": "src:two", "3": "src:three"}
	out := Merge(obj, src).(map[string]string)
	assertCorrectMerge(out)
}

func TestSliceMerge(t *testing.T) {
	obj := &[]int{0, 1}
	src := &[]int{2, 3}
	out := Merge(obj, src).(*[]int)
	assert.Assert(t, len(*out) == 2)
	assert.Assert(t, (*out)[0] == 0)
	assert.Assert(t, (*out)[1] == 1)

	obj = nil
	src = &[]int{2, 3}
	out = Merge(obj, src).(*[]int)
	assert.Assert(t, len(*out) == 2)
	assert.Assert(t, (*out)[0] == 2)
	assert.Assert(t, (*out)[1] == 3)
}

type Z []int

func (z Z) Merge(src interface{}) interface{} {
	return append(z, src.(Z)...)
}

func TestMergable(t *testing.T) {
	obj := &Z{0, 1}
	src := &Z{2, 3}
	out := Merge(obj, src).(*Z)
	assert.Assert(t, len(*out) == 4)
	assert.Assert(t, (*out)[0] == 0)
	assert.Assert(t, (*out)[1] == 1)
	assert.Assert(t, (*out)[2] == 2)
	assert.Assert(t, (*out)[3] == 3)
}
