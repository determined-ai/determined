package schemas

import (
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
	out := WithDefaults(obj).(BindMountV0)
	assertDefaults(out)

	// Make sure pointers on the input are ok.
	objRef := &BindMountV0{}
	objRef = WithDefaults(objRef).(*BindMountV0)
	assertDefaults(*objRef)

	// Make sure input interfaces are ok.
	var iObj interface{} = &BindMountV0{}
	iObj = WithDefaults(iObj)
	assertDefaults(*(iObj.(*BindMountV0)))
}

func TestNonEmptyDefaults(t *testing.T) {
	obj := BindMountV0{ReadOnly: ptrs.BoolPtr(true), Propagation: ptrs.StringPtr("asdf")}
	out := WithDefaults(obj).(BindMountV0)
	assert.Assert(t, *out.ReadOnly == true)
	assert.Assert(t, *out.Propagation == "asdf")
}

func TestArrayOfDefautables(t *testing.T) {
	var obj []BindMountV0
	obj = append(obj, BindMountV0{})
	obj = append(obj, BindMountV0{})
	obj = append(obj, BindMountV0{})

	out := WithDefaults(obj).([]BindMountV0)

	for _, b := range out {
		assert.Assert(t, b.ReadOnly != nil)
		assert.Assert(t, *b.ReadOnly == false)
		assert.Assert(t, b.Propagation != nil)
		assert.Assert(t, *b.Propagation == rprivate)
	}
}
