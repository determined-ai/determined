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
	Type string `json:"type"`
	ValA int    `json:"val_a"`

	CommonVal *string `json:"common_val"`
}

//go:generate ../gen.sh
// TestUnionBV0 is exported.
type TestUnionBV0 struct {
	Type string `json:"type"`
	ValB int    `json:"val_b"`

	CommonVal *string `json:"common_val"`
}

//go:generate ../gen.sh
// TestUnionV0 is exported.
type TestUnionV0 struct {
	A *TestUnionAV0 `union:"type,a" json:"-"`
	B *TestUnionBV0 `union:"type,b" json:"-"`
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
	ValY *string `json:"val_y"`
}

//go:generate ../gen.sh
// TestRootV0 is exported.
type TestRootV0 struct {
	// required; non-pointer.
	ValX int `json:"val_x"`

	// defaultable; pointer.
	SubObj             *TestSubV0   `json:"sub_obj"`
	SubUnion           *TestUnionV0 `json:"sub_union"`
	RuntimeDefaultable *int         `json:"runtime_defaultable"`
	DefaultedArray     []string     `json:"defaulted_array"`
	NodefaultArray     []string     `json:"nodefault_array"`
}

// RuntimeDefaults implements the RuntimeDefaultable interface.
func (t TestRootV0) RuntimeDefaults() interface{} {
	if t.RuntimeDefaultable == nil {
		t.RuntimeDefaultable = ptrs.IntPtr(10)
	}
	return t
}
