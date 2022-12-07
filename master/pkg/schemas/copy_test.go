package schemas

import (
	"reflect"
	"testing"

	"gotest.tools/assert"
)

func TestCopyAllocatedSlice(t *testing.T) {
	src := []string{}
	obj := Copy(src)
	assert.DeepEqual(t, obj, src)
}

func TestCopyUnallocatedSlice(t *testing.T) {
	// Copying an unallocated slice encodes to null.
	var src []string
	obj := Copy(src)
	assert.DeepEqual(t, obj, src)
}

func TestCopyAllocatedMap(t *testing.T) {
	// Copying an allocated map encodes to [].
	src := map[string]string{}

	obj := Copy(src)
	assert.DeepEqual(t, obj, src)
}

func TestCopyUnallocatedMap(t *testing.T) {
	// Copying an unallocated map encodes to null.
	var src map[string]string

	obj := Copy(src)
	assert.DeepEqual(t, obj, src)
}

type A struct {
	M map[string]string
	S []int
	B B
}

type B struct {
	I int
	S string
	C []C
}

type C struct {
	I int
	D map[string]D
}

type D struct {
	I int
	S string
}

func TestCopyNested(t *testing.T) {
	src := A{
		M: map[string]string{"eeny": "meeny", "miney": "moe"},
		S: []int{1, 2, 3, 4},
		B: B{
			I: 5,
			S: "five",
			C: []C{
				{I: 6, D: map[string]D{"one": {I: 1, S: "fish"}, "two": {I: 2, S: "fish"}}},
				{I: 6, D: map[string]D{"red": {I: 3, S: "fish"}, "blue": {I: 4, S: "fish"}}},
			},
		},
	}
	obj := Copy(src)
	assert.DeepEqual(t, obj, src)
}

type E struct {
	// C is a reflect-friendly public member.
	C C
	// d is a reflect-unfriendly private member.
	d D
}

// Copy implements the Copyable psuedointerface.
func (e E) Copy() E {
	return E{
		C: Copy(e.C),
		d: Copy(e.d),
	}
}

// assertDeepEqual is needed since DeepEqual fails on E for the same reason as Copy.
func (e E) assertDeepEqual(t *testing.T, other E) {
	assert.DeepEqual(t, e.C, other.C)
	assert.DeepEqual(t, e.d, other.d)
}

// Wrong number of inputs is not Copyable.
// type F int // defined in merge_test.go

func (f F) Copy(f2 F) F {
	return F(0)
}

// Wrong type of inputs is not Copyable.
// type G int // defined in merge_test.go

func (g *G) Copy() G {
	return G(0)
}

// Wrong number of outputs is not Copyable.
// type H int // defined in merge_test.go

func (h H) Copy() (H, H) {
	return H(0), H(1)
}

// Wrong type of output is not Copyable.
// type I int // defined in merge_test.go

func (i I) Copy() *I {
	return &i
}

func TestCopyable(t *testing.T) {
	src := E{
		C: C{I: 6, D: map[string]D{"help": {I: 1, S: "im"}, "trapped": {I: 2, S: "in a"}}},
		d: D{I: 1, S: "unittest factory"},
	}
	obj := Copy(src)
	obj.assertDeepEqual(t, src)

	// Test the reflect layer directly.

	// E is Copyable.
	vobj := reflect.ValueOf(E{})
	_, ok := copyIfCopyable(vobj)
	assert.Assert(t, ok)

	// Pointer to a Copyable is not Copyable
	vobj = reflect.ValueOf(&E{})
	_, ok = copyIfCopyable(vobj)
	assert.Assert(t, !ok)

	// Wrong num inputs is not Copyable.
	vobj = reflect.ValueOf(F(1))
	_, ok = copyIfCopyable(vobj)
	assert.Assert(t, !ok)

	// Wrong type of inputs is not Copyable.
	vobj = reflect.ValueOf(G(1))
	_, ok = copyIfCopyable(vobj)
	assert.Assert(t, !ok)

	// Wrong type of output is not Copyable.
	vobj = reflect.ValueOf(H(1))
	_, ok = copyIfCopyable(vobj)
	assert.Assert(t, !ok)
}
