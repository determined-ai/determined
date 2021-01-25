package schemas

import (
	"testing"

	"github.com/santhosh-tekuri/jsonschema/v2"
	"gotest.tools/assert"

	"github.com/determined-ai/determined/master/pkg/ptrs"
)

// Re-implement a simple expconf object to avoid circular references.
type BindMountV1 struct {
	HostPath      string  `json:"host_path"`
	ContainerPath string  `json:"container_path"`
	ReadOnly      *bool   `json:"read_only"`
	Propagation   *string `json:"propagation"`
}

func (b *BindMountV1) ParsedSchema() interface{} {
	return ParsedBindMountV1()
}

func (b *BindMountV1) SanityValidator() *jsonschema.Schema {
	return GetSanityValidator(
		"http://determined.ai/schemas/expconf/v1/bind-mount.json",
	)
}

func (b *BindMountV1) CompletenessValidator() *jsonschema.Schema {
	return GetCompletenessValidator(
		"http://determined.ai/schemas/expconf/v1/bind-mount.json",
	)
}

func TestFillEmptyDefaults(t *testing.T) {
	obj := BindMountV1{}
	assertDefaults := func() {
		assert.Assert(t, obj.ReadOnly != nil)
		assert.Assert(t, *obj.ReadOnly == false)
		assert.Assert(t, obj.Propagation != nil)
		assert.Assert(t, *obj.Propagation == "rprivate")
	}

	FillDefaults(&obj)
	assertDefaults()

	// Make sure pointers on the input are ok.
	objRef := &BindMountV1{}
	FillDefaults(&objRef)
	assertDefaults()

	// Make sure input interfaces are ok.
	var iObj interface{} = &BindMountV1{}
	FillDefaults(iObj)
	assertDefaults()
}

func TestNonEmptyDefaults(t *testing.T) {
	obj := BindMountV1{ReadOnly: ptrs.BoolPtr(true), Propagation: ptrs.StringPtr("asdf")}
	FillDefaults(&obj)
	assert.Assert(t, *obj.ReadOnly == true)
	assert.Assert(t, *obj.Propagation == "asdf")
}
