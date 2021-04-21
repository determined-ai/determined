package expconf

// Define types that are only used in testing.

import (
	"encoding/json"

	"github.com/pkg/errors"

	"github.com/determined-ai/determined/master/pkg/ptrs"
	"github.com/determined-ai/determined/master/pkg/union"
)

//go:generate ../gen.sh
// TestUnionAV0 is exported.
type TestUnionAV0 struct {
	RawType string `json:"type"`
	RawValA int    `json:"val_a"`

	RawCommonVal *string `json:"common_val"`
}

//go:generate ../gen.sh
// TestUnionBV0 is exported.
type TestUnionBV0 struct {
	RawType string `json:"type"`
	RawValB int    `json:"val_b"`

	RawCommonVal *string `json:"common_val"`
}

//go:generate ../gen.sh
// TestUnionV0 is exported.
type TestUnionV0 struct {
	RawA *TestUnionAV0 `union:"type,a" json:"-"`
	RawB *TestUnionBV0 `union:"type,b" json:"-"`
}

// UnmarshalJSON is exported.
func (t *TestUnionV0) UnmarshalJSON(data []byte) error {
	if err := union.Unmarshal(data, t); err != nil {
		return err
	}
	type DefaultParser *TestUnionV0
	return errors.Wrap(json.Unmarshal(data, DefaultParser(t)), "failed to parse TestUnion")
}

// MarshalJSON is exported.
func (t TestUnionV0) MarshalJSON() ([]byte, error) {
	return union.Marshal(t)
}

//go:generate ../gen.sh
// TestSubV0 is exported.
type TestSubV0 struct {
	// defaultable; pointer.
	RawValY *string `json:"val_y"`
}

//go:generate ../gen.sh
// TestRootV0 is exported.
type TestRootV0 struct {
	// required; non-pointer.
	RawValX int `json:"val_x"`

	// defaultable; pointer.
	RawSubObj             *TestSubV0   `json:"sub_obj"`
	RawSubUnion           *TestUnionV0 `json:"sub_union"`
	RawRuntimeDefaultable *int         `json:"runtime_defaultable"`
	RawDefaultedArray     []string     `json:"defaulted_array"`
	RawNodefaultArray     []string     `json:"nodefault_array"`
}

// RuntimeDefaults implements the RuntimeDefaultable interface.
func (t TestRootV0) RuntimeDefaults() interface{} {
	if t.RawRuntimeDefaultable == nil {
		t.RawRuntimeDefaultable = ptrs.IntPtr(10)
	}
	return t
}
