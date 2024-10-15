// disallowProperties is for restricting which properties are allowed in an object with
// per-property error messages, such as when we allow a k8s pod spec with some fields disallowed.
//
// Example: The "pod_spec" property of the environment config:
//
//     "pod_spec": {
//         "type": "object",
//         "disallowProperties": {
//             "name": "pod Name is not a configurable option",
//             "name_space": "pod NameSpace is not a configurable option"
//         }
//     }

// See ./checks.go for notes on implementing extensions for the santhosh-tekuri/jsonschema package.

package extensions

import (
	"github.com/santhosh-tekuri/jsonschema/v2"
)

func disallowPropertiesCompile(ctx jsonschema.CompilerContext, m JSONObject) (interface{}, error) {
	disallowSchema, ok := m["disallowProperties"]
	if !ok {
		return nil, nil
	}

	// disallowProperties is a map of string property names to error messages.
	disallowed := map[string]string{}

	for prop, msg := range disallowSchema.(JSONObject) {
		disallowed[prop] = msg.(string)
	}

	return disallowed, nil
}

func disallowPropertiesValidate(
	ctx jsonschema.ValidationContext, rawDisallowed interface{}, instance JSON,
) error {
	object, ok := instance.(JSONObject)
	if !ok {
		return nil
	}

	disallowed := rawDisallowed.(map[string]string)

	var errors []error

	for prop, msg := range disallowed {
		if _, ok := object[prop]; ok {
			errors = append(errors, ctx.Error("disallowProperties", msg))
		}
	}

	if len(errors) > 0 {
		var x jsonschema.ValidationError
		return x.Group(ctx.Error("disallowProperties", "found disallowed properties"), errors...)
	}

	return nil
}

// DisallowPropertiesExtension instantiates the disallowProperties extension.
func DisallowPropertiesExtension() jsonschema.Extension {
	meta, err := jsonschema.CompileString("disallowProperties.json", `{
		"properties" : {
			"disallowProperties": {
				"additionalProperties": { "type": "string" }
			}
		}
	}`)
	if err != nil {
		panic(err)
	}
	return jsonschema.Extension{
		Meta:     meta,
		Compile:  disallowPropertiesCompile,
		Validate: disallowPropertiesValidate,
	}
}
