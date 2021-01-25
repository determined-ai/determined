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

	Merge(&obj, src)

	assert.Assert(t, *obj.A == "obj:x.a")
	assert.Assert(t, *obj.B == "src:x.b")
	assert.Assert(t, *obj.C == "obj:x.c")
}

type Y struct {
	A *UA `union:"type,ux" json:"-"`
	B *UB `union:"type,uy" json:"-"`
	C *string
}

type UA struct {
	A *string
}

type UB struct {
	B *string
}

func TestUnionMerge(t *testing.T) {
	// 1. src has a union member, obj does not -> recurse into that field.
	obj := Y{
		A: nil,
		B: nil,
		C: ptrs.StringPtr("obj:c"),
	}

	src := Y{
		A: nil,
		B: &UB{
			B: ptrs.StringPtr("src:b:b"),
		},
		C: ptrs.StringPtr("src:c"),
	}

	Merge(&obj, src)

	assert.Assert(t, obj.A == nil)
	assert.Assert(t, *obj.B.B == "src:b:b")
	assert.Assert(t, *obj.C == "obj:c")

	// 2. src has a union member, obj has the same one -> recurse into that field.
	obj = Y{
		A: &UA{},
		B: nil,
		C: nil,
	}

	src = Y{
		A: &UA{A: ptrs.StringPtr("src:a:a")},
		B: nil,
		C: ptrs.StringPtr("src:y.c"),
	}

	Merge(&obj, src)
	assert.Assert(t, *obj.A.A == "src:a:a")
	assert.Assert(t, obj.B == nil)
	assert.Assert(t, *obj.C == "src:y.c")

	// 3. src has a union member, obj has the different one -> do not recurse.
	obj = Y{
		A: &UA{},
		B: nil,
		C: nil,
	}

	src = Y{
		A: nil,
		B: &UB{
			B: ptrs.StringPtr("src:b:b"),
		},
		C: nil,
	}

	Merge(&obj, src)
	assert.Assert(t, obj.A.A == nil)
	assert.Assert(t, obj.B == nil)
	assert.Assert(t, obj.C == nil)
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
	Merge(&obj, src)
	assertCorrectMerge(obj)

	// Make sure the number of pointers on the input does not affect merging.
	objRef := &map[string]string{"1": "obj:one", "2": "obj:two"}
	Merge(&objRef, src)
	assertCorrectMerge(*objRef)

	objRef = &map[string]string{"1": "obj:one", "2": "obj:two"}
	objRefRef := &objRef
	Merge(&objRefRef, src)
	assertCorrectMerge(**objRefRef)

	// Make sure the nubmer of input pointers is irrelevant as well.
	obj = map[string]string{"1": "obj:one", "2": "obj:two"}
	srcRef := &src
	Merge(&obj, &srcRef)
	assertCorrectMerge(obj)
}

func TestSliceMerge(t *testing.T) {
	obj := &[]int{0, 1}
	src := &[]int{2, 3}
	Merge(&obj, src)
	assert.Assert(t, len(*obj) == 2)
	assert.Assert(t, (*obj)[0] == 0)
	assert.Assert(t, (*obj)[1] == 1)

	obj = nil
	src = &[]int{2, 3}
	Merge(&obj, src)
	assert.Assert(t, len(*obj) == 2)
	assert.Assert(t, (*obj)[0] == 2)
	assert.Assert(t, (*obj)[1] == 3)
}

type Z []int

func (z *Z) Merge(src interface{}) {
	*z = append(*z, src.(Z)...)
}

func TestMergable(t *testing.T) {
	obj := &Z{0, 1}
	src := &Z{2, 3}
	Merge(obj, src)
	assert.Assert(t, len(*obj) == 4)
	assert.Assert(t, (*obj)[0] == 0)
	assert.Assert(t, (*obj)[1] == 1)
	assert.Assert(t, (*obj)[2] == 2)
	assert.Assert(t, (*obj)[3] == 3)
}
