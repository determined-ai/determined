// See determined/common/schemas/extensions.py for the explanation of this and other extensions.
// See ./checks.go for notes on implementing extensions for the santhosh-tekuri/jsonschema package.

package extensions

import (
	"encoding/json"
	"fmt"
	"path/filepath"
	"strings"

	"github.com/santhosh-tekuri/jsonschema/v2"
)

// getPropertyByPath dereferences a series of nested json objects according to a "."-dilineated
// name like "a.b.c" and returns the result.  It panics if the name is not present.
func getPropertyByPath(instance JSON, name string) JSON {
	for _, key := range strings.Split(name, ".") {
		instance = instance.(JSONObject)[key]
	}
	return instance
}

// getLengthUnits takes a raw, unmarshaled Length object and returns "records", "epochs", or
// "batches".  It does not validate the Length; that should be done elsewhere.
func getLengthUnits(instance JSON) string {
	// Just return the name of the first key in the object.
	for key := range instance.(JSONObject) {
		return key
	}
	panic("no unit found")
}

// getLengthValue takes a raw, unmarshaled Length object and returns the numeric length associated
// with it.  It does not validate the Length; that should be done elsewhere.
func getLengthValue(instance JSON) int64 {
	for _, value := range instance.(JSONObject) {
		i, err := value.(json.Number).Int64()
		if err != nil {
			panic("length is not an integer")
		}
		return i
	}
	panic("no value found")
}

// compareProperty is the parsed schema that must be matched against.
type compareProperty struct {
	Type string
	A    string
	B    string
}

func comparePropertiesCompile(ctx jsonschema.CompilerContext, m JSONObject) (interface{}, error) {
	rawCompare, ok := m["compareProperties"]
	if !ok {
		return nil, nil
	}
	compare := rawCompare.(JSONObject)

	cmp := compareProperty{}

	cmp.Type = compare["type"].(string)
	cmp.A = compare["a"].(string)
	cmp.B = compare["b"].(string)

	return cmp, nil
}

func comparePropertiesValidate(
	ctx jsonschema.ValidationContext, rawCmp interface{}, instance JSON,
) error {
	cmp := rawCmp.(compareProperty)

	// Disregard panics due to wrongly-typed structures; this extensions does not need to
	// double-check the well-formedness of the instance, just the values.  In fact, duplicate
	// errors would actually be actively unhelpful.
	defer func() {
		_ = recover()
	}()

	a := getPropertyByPath(instance, cmp.A)
	b := getPropertyByPath(instance, cmp.B)

	switch cmp.Type {
	case "a<b":
		aNum, err := a.(json.Number).Float64()
		if err != nil {
			panic("length is not a number")
		}
		bNum, err := b.(json.Number).Float64()
		if err != nil {
			panic("length is not a number")
		}
		if aNum >= bNum {
			return ctx.Error(
				"compareProperties",
				fmt.Sprintf("%v must be less than %v", cmp.A, cmp.B),
			)
		}

	case "same_units":
		aUnit := getLengthUnits(a)
		bUnit := getLengthUnits(b)
		if aUnit != bUnit {
			return ctx.Error(
				"compareProperties",
				fmt.Sprintf("%v must use the same units as %v", cmp.A, cmp.B),
			)
		}

	case "length_a<length_b":
		aLength := getLengthValue(a)
		bLength := getLengthValue(b)
		if aLength >= bLength {
			return ctx.Error(
				"compareProperties",
				fmt.Sprintf("%v must be less than %v", cmp.A, cmp.B),
			)
		}

	case "a_is_subdir_of_b":
		aPath := filepath.Clean(a.(string))
		bPath := filepath.Clean(b.(string))
		if filepath.IsAbs(aPath) {
			if !strings.HasPrefix(aPath, bPath) {
				return ctx.Error(
					"compareProperties",
					fmt.Sprintf("%v must be a subdirectory of %v", cmp.A, cmp.B),
				)
			}
		} else {
			if strings.HasPrefix(aPath, "..") {
				return ctx.Error(
					"compareProperties",
					fmt.Sprintf("%v must be a subdirectory of %v", cmp.A, cmp.B),
				)
			}
		}
	}
	return nil
}

// ComparePropertiesExtension instantiates the compareProperties extension.
func ComparePropertiesExtension() jsonschema.Extension {
	meta, err := jsonschema.CompileString("compareProperties.json", `{
		"properties" : {
			"compareProperties": {
				"type": "object",
				"required": ["type", "a", "b"],
				"properties": {
					"type": {"type": "string"},
					"a": {"type": "string"},
					"b": {"type": "string"}
				}
			}
		}
	}`)
	if err != nil {
		panic(err)
	}
	return jsonschema.Extension{
		Meta:     meta,
		Compile:  comparePropertiesCompile,
		Validate: comparePropertiesValidate,
	}
}
