package schemas

import (
	"testing"

	"github.com/santhosh-tekuri/jsonschema/v2"
	"gotest.tools/assert"

	"github.com/determined-ai/determined/master/pkg/ptrs"
)

// Re-implement a simple expconf object to avoid circular references.
type BindMountV0 struct {
	HostPath      string  `json:"host_path"`
	ContainerPath string  `json:"container_path"`
	ReadOnly      *bool   `json:"read_only"`
	Propagation   *string `json:"propagation"`
}

func (b *BindMountV0) ParsedSchema() interface{} {
	return ParsedBindMountV0()
}

func (b *BindMountV0) SanityValidator() *jsonschema.Schema {
	return GetSanityValidator(
		"http://determined.ai/schemas/expconf/v0/bind-mount.json",
	)
}

func (b *BindMountV0) CompletenessValidator() *jsonschema.Schema {
	return GetCompletenessValidator(
		"http://determined.ai/schemas/expconf/v0/bind-mount.json",
	)
}

type BindMountsConfigV0 []BindMountV0

func TestFillEmptyDefaults(t *testing.T) {
	obj := BindMountV0{}

	FillDefaults(&obj)
	assert.Assert(t, obj.ReadOnly != nil)
	assert.Assert(t, *obj.ReadOnly == false)
	assert.Assert(t, obj.Propagation != nil)
	assert.Assert(t, *obj.Propagation == "rprivate")

	// Make sure pointers on the input are ok.
	objRef := &BindMountV0{}
	FillDefaults(&objRef)
	assert.Assert(t, objRef.ReadOnly != nil)
	assert.Assert(t, *objRef.ReadOnly == false)
	assert.Assert(t, objRef.Propagation != nil)
	assert.Assert(t, *objRef.Propagation == "rprivate")

	// Make sure input interfaces are ok.
	var iObj interface{}
	iObj = &BindMountV0{}
	FillDefaults(iObj)
	assert.Assert(t, iObj.(*BindMountV0).ReadOnly != nil)
	assert.Assert(t, *iObj.(*BindMountV0).ReadOnly == false)
	assert.Assert(t, iObj.(*BindMountV0).Propagation != nil)
	assert.Assert(t, *iObj.(*BindMountV0).Propagation == "rprivate")
}

func TestNonEmptyDefaults(t *testing.T) {
	obj := BindMountV0{ReadOnly: ptrs.BoolPtr(true), Propagation: ptrs.StringPtr("asdf")}
	FillDefaults(&obj)
	assert.Assert(t, *obj.ReadOnly == true)
	assert.Assert(t, *obj.Propagation == "asdf")
}
