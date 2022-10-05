package expconf

// Define types that are only used in testing.

import (
	"encoding/json"

	"github.com/pkg/errors"

	"github.com/determined-ai/determined/master/pkg/union"
)

// TestUnionAV0 is exported.
//
//go:generate ../gen.sh
type TestUnionAV0 struct {
	RawType string `json:"type"`
	RawValA int    `json:"val_a"`

	RawCommonVal *string `json:"common_val"`
}

// TestUnionBV0 is exported.
//
//go:generate ../gen.sh
type TestUnionBV0 struct {
	RawType string `json:"type"`
	RawValB int    `json:"val_b"`

	RawCommonVal *string `json:"common_val"`
}

// TestUnionV0 is exported.
//
//go:generate ../gen.sh
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

// TestSubV0 is exported.
//
//go:generate ../gen.sh
type TestSubV0 struct {
	// defaultable; pointer.
	RawValY *string `json:"val_y"`
}

// TestRootV0 is exported.
//
//go:generate ../gen.sh
type TestRootV0 struct {
	// required; non-pointer.
	RawValX int `json:"val_x"`

	// defaultable; pointer.
	RawSubObj         *TestSubV0   `json:"sub_obj"`
	RawSubUnion       *TestUnionV0 `json:"sub_union"`
	RawDefaultedArray []string     `json:"defaulted_array"`
	RawNodefaultArray []string     `json:"nodefault_array"`

	// runtime-defaultable container; non-pointer struct containing a pointer.
	RawRuntimeDefaultable TestRuntimeDefaultable `json:"runtime_defaultable"`
}

// TestRuntimeDefaultable is container for implementing runtime defaults.
type TestRuntimeDefaultable struct {
	RawInt *int
}

// WithDefaults implements the Defaultable interface.
func (t TestRuntimeDefaultable) WithDefaults() interface{} {
	var i int
	if t.RawInt != nil {
		i = *t.RawInt
	} else {
		i = 10
	}
	return TestRuntimeDefaultable{&i}
}

// MarshalJSON makes the container transparent for marshaling.
func (t *TestRuntimeDefaultable) MarshalJSON() ([]byte, error) {
	return json.Marshal(t.RawInt)
}

// UnmarshalJSON makes the container transparent for unmarshaling.
func (t *TestRuntimeDefaultable) UnmarshalJSON(bytes []byte) error {
	return json.Unmarshal(bytes, &t.RawInt)
}
