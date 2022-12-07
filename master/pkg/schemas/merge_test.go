package schemas

import (
	"reflect"
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
		A: ptrs.Ptr("obj:x.a"),
		B: nil,
		C: ptrs.Ptr("obj:x.c"),
	}

	src := X{
		A: ptrs.Ptr("src:x.a"),
		B: ptrs.Ptr("src:x.b"),
		C: nil,
	}

	out := Merge(obj, src)

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
	out := Merge(obj, src)
	assertCorrectMerge(out)
}

func TestSliceMerge(t *testing.T) {
	obj := &[]int{0, 1}
	src := &[]int{2, 3}
	out := Merge(obj, src)
	assert.Assert(t, len(*out) == 2)
	assert.Assert(t, (*out)[0] == 0)
	assert.Assert(t, (*out)[1] == 1)

	obj = nil
	src = &[]int{2, 3}
	out = Merge(obj, src)
	assert.Assert(t, len(*out) == 2)
	assert.Assert(t, (*out)[0] == 2)
	assert.Assert(t, (*out)[1] == 3)
}

// Z is Mergable.
type Z []int

func (z Z) Merge(src Z) Z {
	return append(z, src...)
}

// Wrong number of inputs is not Mergable.
type F int

func (f F) Merge() F {
	return F(0)
}

// Wrong type of inputs is not Mergable.
type G int

func (g *G) Merge(g2 G) G {
	return G(0)
}

// Wrong number of outputs is not Mergable.
type H int

func (h H) Merge(h2 H) (H, H) {
	return H(0), H(1)
}

// Wrong type of output is not Mergable.
type I int

func (i I) Merge(i2 I) *I {
	return &i
}

func TestMergable(t *testing.T) {
	obj := &Z{0, 1}
	src := &Z{2, 3}
	out := Merge(obj, src)
	assert.Assert(t, len(*out) == 4)
	assert.Assert(t, (*out)[0] == 0)
	assert.Assert(t, (*out)[1] == 1)
	assert.Assert(t, (*out)[2] == 2)
	assert.Assert(t, (*out)[3] == 3)

	// Test the reflect layer directly.

	// Z is Mergable.
	vobj := reflect.ValueOf(Z{0, 1})
	vsrc := reflect.ValueOf(Z{2, 3})
	_, ok := mergeIfMergable(vobj, vsrc)
	assert.Assert(t, ok)

	// Pointer to a Mergable is not Mergable
	vobj = reflect.ValueOf(&Z{0, 1})
	vsrc = reflect.ValueOf(&Z{2, 3})
	_, ok = mergeIfMergable(vobj, vsrc)
	assert.Assert(t, !ok)

	// Wrong num inputs is not Mergable.
	vobj = reflect.ValueOf(F(1))
	vsrc = reflect.ValueOf(F(1))
	_, ok = mergeIfMergable(vobj, vsrc)
	assert.Assert(t, !ok)

	// Wrong type of inputs is not Mergable.
	vobj = reflect.ValueOf(G(1))
	vsrc = reflect.ValueOf(G(1))
	_, ok = mergeIfMergable(vobj, vsrc)
	assert.Assert(t, !ok)

	// Wrong type of output is not Mergable.
	vobj = reflect.ValueOf(H(1))
	vsrc = reflect.ValueOf(H(1))
	_, ok = mergeIfMergable(vobj, vsrc)
	assert.Assert(t, !ok)
}
