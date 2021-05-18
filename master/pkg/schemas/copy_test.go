package schemas

import (
	"reflect"
	"testing"

	"gotest.tools/assert"
)

// Copy is the non-reflect version of copy, but mostly the reflect version is called from other
// reflect code, so it's defined here in test code.
func Copy(src interface{}) interface{} {
	return cpy(reflect.ValueOf(src)).Interface()
}

func TestCopyAllocatedSlice(t *testing.T) {
	src := []string{}
	obj := Copy(src).([]string)
	assert.DeepEqual(t, obj, src)
}

func TestCopyUnallocatedSlice(t *testing.T) {
	// Copying an unallocated slice encodes to null.
	var src []string
	obj := Copy(src).([]string)
	assert.DeepEqual(t, obj, src)
}

func TestCopyAllocatedMap(t *testing.T) {
	// Copying an allocated map encodes to [].
	src := map[string]string{}

	obj := Copy(src).(map[string]string)
	assert.DeepEqual(t, obj, src)
}

func TestCopyUnallocatedMap(t *testing.T) {
	// Copying an unallocated map encodes to null.
	var src map[string]string

	obj := Copy(src).(map[string]string)
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
	obj := Copy(src).(A)
	assert.DeepEqual(t, obj, src)
}

type E struct {
	// C is a reflect-friendly public member.
	C C
	// d is a reflect-unfriendly private member.
	d D
}

// Copy implements the Copyable interface.
func (e E) Copy() interface{} {
	return E{
		C: Copy(e.C).(C),
		d: Copy(e.d).(D),
	}
}

// assertDeepEqual is needed since DeepEqual fails on E for the same reason as Copy.
func (e E) assertDeepEqual(t *testing.T, other E) {
	assert.DeepEqual(t, e.C, other.C)
	assert.DeepEqual(t, e.d, other.d)
}

func TestCopyable(t *testing.T) {
	src := E{
		C: C{I: 6, D: map[string]D{"help": {I: 1, S: "im"}, "trapped": {I: 2, S: "in a"}}},
		d: D{I: 1, S: "unittest factory"},
	}
	obj := Copy(src).(E)
	obj.assertDeepEqual(t, src)
}
