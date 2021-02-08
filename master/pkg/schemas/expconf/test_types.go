package expconf

import (
	"encoding/json"
	// "fmt"
	// "time"

	// petname "github.com/dustinkirkland/golang-petname"
	"github.com/pkg/errors"

	"github.com/determined-ai/determined/master/pkg/ptrs"
	"github.com/determined-ai/determined/master/pkg/schemas"
	"github.com/determined-ai/determined/master/pkg/union"
)

// TestUnionAV1 is exported.
type TestUnionAV1 struct {
	Type string `json:"type"`
	ValA int    `json:"val_a"`

	CommonVal *string `json:"common_val"`
}

// TestUnionBV1 is exported.
type TestUnionBV1 struct {
	Type string `json:"type"`
	ValB int    `json:"val_b"`

	CommonVal *string `json:"common_val"`
}

// TestUnionV1 is exported.
type TestUnionV1 struct {
	A *TestUnionAV1 `union:"type,a" json:"-"`
	B *TestUnionBV1 `union:"type,b" json:"-"`

	// I think common memebers should not exist, but for now they do and you can handle them with
	// the DefaultSource interface.
	CommonVal *string `json:"common_val"`
}

// UnmarshalJSON is exported.
func (t *TestUnionV1) UnmarshalJSON(data []byte) error {
	if err := union.Unmarshal(data, t); err != nil {
		return err
	}
	type DefaultParser *TestUnionV1
	return errors.Wrap(json.Unmarshal(data, DefaultParser(t)), "failed to parse TestUnion")
}

// MarshalJSON is exported.
func (t TestUnionV1) MarshalJSON() ([]byte, error) {
	return union.Marshal(t)
}

// DefaultSource implements the Defaultable interface.
func (t *TestUnionV1) DefaultSource() interface{} {
	return schemas.UnionDefaultSchema(t)
}

// TestSubV1 is exported.
type TestSubV1 struct {
	// defaultable; pointer.
	ValY *string `json:"val_y"`
}

// TestRootV1 is exported.
type TestRootV1 struct {
	// required; non-pointer.
	ValX int `json:"val_x"`

	// defaultable; pointer.
	SubObj             *TestSubV1   `json:"sub_obj"`
	SubUnion           *TestUnionV1 `json:"sub_union"`
	RuntimeDefaultable *int         `json:"runtime_defaultable"`
}

// RuntimeDefaults implements the RuntimeDefaultable interface.
func (t *TestRootV1) RuntimeDefaults() {
	if t.RuntimeDefaultable == nil {
		t.RuntimeDefaultable = ptrs.IntPtr(10)
	}
}
