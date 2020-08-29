package expconf

import (
	"encoding/json"
	"fmt"
	"sort"
	"strconv"
	"strings"

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

// jsonFromYaml takes yaml-formatted bytes and converts them to json-format for the purpose of
// applying json-schema validation.
func jsonFromYaml(byts []byte) ([]byte, error) {
	var blob JSON
	err := yaml.Unmarshal(byts, &blob)
	if err != nil {
		return nil, err
	}

	return json.Marshal(blob)
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
	var errors []string

	for _, subError := range valError.Causes {
		errors = append(errors, getChildErrors(subError, instance)...)
	}

	if len(errors) > 0 {
		sort.Strings(errors)
		return errors
	}

	msg := valError.Message
	displayPtr := renderJSONPointer(valError.InstancePtr, instance)
	errors = append(errors, fmt.Sprintf("% *s<config>%v: %v", 0, "", displayPtr, msg))

	return errors
}

// getRenderedErrors takes a jsonschema valiation plus the bytes that caused it and returns
// user-facing errors.
func getRenderedErrors(err error, byts []byte) []string {
	// Make sure the input is even valid json, plus we'll need this to render the json pointer.
	var instance JSON
	if uErr := json.Unmarshal(byts, &instance); uErr != nil {
		return []string{fmt.Sprintf("%v", uErr)}
	}

	tErr, ok := err.(*jsonschema.ValidationError)
	if !ok {
		return []string{fmt.Sprintf("%v", err)}
	}

	return getChildErrors(tErr, instance)
}
