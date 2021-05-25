// See determined/common/schemas/extensions.py for the explanation of this and other extensions.
// See ./checks.go for notes on implementing extensions for the santhosh-tekuri/jsonschema package.

package extensions

import (
	"github.com/santhosh-tekuri/jsonschema/v2"
)

func eventuallyCompile(ctx jsonschema.CompilerContext, m JSONObject) (interface{}, error) {
	rawEventually, ok := m["eventually"]
	if !ok {
		return nil, nil
	}

	// eventually is a JSON object that wraps other JSON objects that are validated during
	// the completeness validation step.
	eventually := rawEventually.(JSONObject)

	compiled, err := ctx.Compile(eventually)
	if err != nil {
		return nil, err
	}

	return compiled, nil
}

func eventuallyValidate(
	ctx jsonschema.ValidationContext, rawCompiled interface{}, instance JSON,
) error {
	return ctx.Validate(rawCompiled.(*jsonschema.Schema), instance)
}

// EventuallyExtension instantiates the eventually extension.
func EventuallyExtension() jsonschema.Extension {
	meta, err := jsonschema.CompileString("eventuallyExtension.json", `{
		"properties" : {
			"eventually": {
				"additionalProperties": { "type": "object" }
			}
		}
	}`)
	if err != nil {
		panic(err)
	}
	return jsonschema.Extension{
		Meta:     meta,
		Compile:  eventuallyCompile,
		Validate: eventuallyValidate,
	}
}
