package schemas

// lint.go is full of functions for writing unit tests that ensure certain assumptions about the
// nature of our json-schema values and related go types hold constant.
//
// lint.go is not useful outside of writing tests.

import (
	"encoding/json"
	"reflect"

	"github.com/pkg/errors"
)

// LintStructDefaults asserts that all fields with json-schema defaults correspond to pointers types
// so that fill-defaults will work.  It also asserts that all default bytes are Unmarshalable.
//
// LintStructDefaults does not recurse; you should call for each generated struct.
//
// LintStructDefaults can accept a typed nil-pointer without issue.
func LintStructDefaults(x interface{}) []error {
	t := reflect.TypeOf(x)
	for t.Kind() == reflect.Ptr || t.Kind() == reflect.Interface {
		t = t.Elem()
	}
	if t.Kind() != reflect.Struct {
		return []error{errors.Errorf(
			"LintStructDefaults can only be called on a struct-like input, not %v", t.Name(),
		)}
	}
	// Allocate some memory to call getDefaultSource on.
	defaultSource := getDefaultSource(reflect.New(t).Elem())
	if defaultSource == nil {
		return []error{errors.Errorf(
			"LintStructDefaults called on %v which has no default source", t.Name(),
		)}
	}
	var out []error
	for i := 0; i < t.NumField(); i++ {
		field := t.Field(i)
		// Is there a default for this field's tag?
		fieldDefaultBytes := findDefaultInSchema(defaultSource, field)
		if fieldDefaultBytes == nil {
			continue
		}
		// Is this field a pointer type?
		if field.Type.Kind() != reflect.Ptr {
			out = append(out, errors.Errorf(
				"%v.%v has default bytes '%v' but it has non-pointer type '%v'",
				t.Name(),
				field.Name,
				string(fieldDefaultBytes),
				field.Type,
			))
		}
		// Can we unmarshal defaultBtyes into a pointer of the field type?
		fieldObj := reflect.New(field.Type).Interface()
		err := json.Unmarshal(fieldDefaultBytes, fieldObj)
		if err != nil {
			out = append(out,
				errors.Wrapf(
					err,
					"failed to unmarshal defaultBytes of '%v' for %v.%v of type '%v'",
					string(fieldDefaultBytes),
					t.Name(),
					field.Name,
					field.Type,
				),
			)
		}
	}
	return out
}
