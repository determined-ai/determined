package schemas

import (
	"reflect"
	"testing"

	"github.com/santhosh-tekuri/jsonschema/v2"
	"gotest.tools/assert"

	"github.com/determined-ai/determined/master/pkg/ptrs"
)

const (
	rprivate = "rprivate"
)

// Re-implement a simple expconf object to avoid circular references.
type BindMountV0 struct {
	HostPath      string  `json:"host_path"`
	ContainerPath string  `json:"container_path"`
	ReadOnly      *bool   `json:"read_only"`
	Propagation   *string `json:"propagation"`
}

func (b BindMountV0) ParsedSchema() interface{} {
	return ParsedBindMountV0()
}

func (b BindMountV0) SanityValidator() *jsonschema.Schema {
	return GetSanityValidator(
		"http://determined.ai/schemas/expconf/v0/bind-mount.json",
	)
}

func (b BindMountV0) CompletenessValidator() *jsonschema.Schema {
	return GetCompletenessValidator(
		"http://determined.ai/schemas/expconf/v0/bind-mount.json",
	)
}

func TestFillEmptyDefaults(t *testing.T) {
	assertDefaults := func(b BindMountV0) {
		assert.Assert(t, b.ReadOnly != nil)
		assert.Assert(t, *b.ReadOnly == false)
		assert.Assert(t, b.Propagation != nil)
		assert.Assert(t, *b.Propagation == rprivate)
	}

	obj := BindMountV0{}
	out := WithDefaults(obj)
	assertDefaults(out)

	// Make sure pointers on the input are ok.
	objRef := &BindMountV0{}
	objRef = WithDefaults(objRef)
	assertDefaults(*objRef)

	// Make sure input interfaces are ok.
	var iObj interface{} = &BindMountV0{}
	iObj = WithDefaults(iObj)
	assertDefaults(*(iObj.(*BindMountV0)))
}

func TestNonEmptyDefaults(t *testing.T) {
	obj := BindMountV0{ReadOnly: ptrs.Ptr(true), Propagation: ptrs.Ptr("asdf")}
	out := WithDefaults(obj)
	assert.Assert(t, *out.ReadOnly == true)
	assert.Assert(t, *out.Propagation == "asdf")
}

func TestArrayOfDefautables(t *testing.T) {
	var obj []BindMountV0
	obj = append(obj, BindMountV0{})
	obj = append(obj, BindMountV0{})
	obj = append(obj, BindMountV0{})

	out := WithDefaults(obj)

	for _, b := range out {
		assert.Assert(t, b.ReadOnly != nil)
		assert.Assert(t, *b.ReadOnly == false)
		assert.Assert(t, b.Propagation != nil)
		assert.Assert(t, *b.Propagation == rprivate)
	}
}

// Y is Defaultable.
type Y int

func (y Y) WithDefaults() Y {
	return Y(0)
}

// Wrong number of inputs is not Defaultable.
// type F int // defined in merge_test.go

func (f F) WithDefaults(f2 F) F {
	return F(0)
}

// Wrong type of inputs is not Defaultable.
// type G int // defined in merge_test.go

func (g *G) WithDefaults() G {
	return G(0)
}

// Wrong number of outputs is not Defaultable.
// type H int // defined in merge_test.go

func (h H) WithDefaults() (H, H) {
	return H(0), H(1)
}

// Wrong type of output is not Defaultable.
// type I int // defined in merge_test.go

func (i I) WithDefaults() *I {
	return &i
}

func TestDefaultIfDefaultable(t *testing.T) {
	// Y is Defaultable.
	vobj := reflect.ValueOf(Y(0))
	_, ok := defaultIfDefaultable(vobj)
	assert.Assert(t, ok)

	// Pointer to a Defaultable is not Defaultable
	vobj = reflect.ValueOf(&E{})
	_, ok = defaultIfDefaultable(vobj)
	assert.Assert(t, !ok)

	// Wrong num inputs is not Defaultable.
	vobj = reflect.ValueOf(F(1))
	_, ok = defaultIfDefaultable(vobj)
	assert.Assert(t, !ok)

	// Wrong type of inputs is not Defaultable.
	vobj = reflect.ValueOf(G(1))
	_, ok = defaultIfDefaultable(vobj)
	assert.Assert(t, !ok)

	// Wrong type of output is not Defaultable.
	vobj = reflect.ValueOf(H(1))
	_, ok = defaultIfDefaultable(vobj)
	assert.Assert(t, !ok)
}
