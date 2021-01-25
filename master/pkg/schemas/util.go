package schemas

import (
	"encoding/json"
	"fmt"
	"sort"
	"strconv"
	"strings"
	"reflect"

	"github.com/pkg/errors"
	"github.com/ghodss/yaml"
	"github.com/santhosh-tekuri/jsonschema/v2"
)

type (
	// JSON is the type for arbitrary JSON.
	JSON = interface{}
	// JSONObject is the type for JSON objects.
	JSONObject = map[string]interface{}
	// JSONArray is the type for JSON arrays.
	JSONArray = []interface{}
)

// JsonFromYaml takes yaml-formatted bytes and converts them to json-format for the purpose of
// applying json-schema validation.
func JsonFromYaml(byts []byte) ([]byte, error) {
	var blob JSON
	err := yaml.Unmarshal(byts, &blob)
	if err != nil {
		return nil, errors.Wrap(err, "not valid yaml")
	}

	byts, err = json.Marshal(blob)
	if err != nil {
		return nil, errors.Wrap(err, "yaml is not convertible to json")
	}

	return byts, nil
}

// JoinErrors is like strings.Join but for []error types.
func JoinErrors(errs []error, joiner string) string {
	var strs []string
	for _, err := range errs {
		strs = append(strs, err.Error())
	}
	return strings.Join(strs, "\n")
}

// GetRenderedErrors takes a jsonschema valiation plus the bytes that caused it and returns
// user-facing errors.
func GetRenderedErrors(err error, byts []byte) []error {
	// Make sure the input is even valid json, plus we'll need this to render the json pointer.
	var instance JSON
	if uErr := json.Unmarshal(byts, &instance); uErr != nil {
		return []error{uErr}
	}

	tErr, ok := err.(*jsonschema.ValidationError)
	if !ok {
		return []error{err}
	}

	msgs := getChildErrors(tErr, instance)
	sort.Strings(msgs)

	var errs []error
	for _, msg := range msgs {
		errs = append(errs, errors.New(msg))
	}
	return errs
}

// renderJSONPointer renders "#/key/0/key" as ".key[0].key".  Return the raw ptr in case of errors.
func renderJSONPointer(ptr string, instance JSON) string {
	out := ""
	split := strings.Split(ptr, "/")[1:]

	for _, s := range split {
		switch tInstance := instance.(type) {
		case JSONArray:
			i, err := strconv.Atoi(s)
			if err != nil || i >= len(tInstance) {
				return ptr
			}
			instance = tInstance[i]
			out += fmt.Sprintf("[%d]", i)

		case JSONObject:
			var ok bool
			instance, ok = tInstance[s]
			if !ok {
				return ptr
			}
			out += fmt.Sprintf(".%s", s)

		default:
			return ptr
		}
	}
	return out
}

// getChildErrors takes a nested-tree-style jsonschema error and returns a flat list of leaf errors.
func getChildErrors(valError *jsonschema.ValidationError, instance JSON) []string {
	var errs []string

	for _, subError := range valError.Causes {
		errs = append(errs, getChildErrors(subError, instance)...)
	}

	if len(errs) > 0 {
		return errs
	}

	msg := valError.Message
	displayPtr := renderJSONPointer(valError.InstancePtr, instance)
	errs = append(errs, fmt.Sprintf("% *s<config>%v: %v", 0, "", displayPtr, msg))

	return errs
}

func derefType(typ reflect.Type) reflect.Type {
	for typ.Kind() == reflect.Ptr {
		typ = typ.Elem()
	}
	return typ
}

func derefInput(val reflect.Value) (reflect.Value, bool) {
	for val.Kind() == reflect.Ptr {
		if val.IsZero() {
			return val, false
		}
		val = val.Elem()
	}
	return val, true
}

func derefOutput(val reflect.Value) (reflect.Value, bool) {
	allocated := false
	for val.Kind() == reflect.Ptr {
		if val.IsZero() {
			val.Set(reflect.New(val.Type().Elem()))
			allocated = true
		}
		val = val.Elem()
	}
	return val, allocated
}
