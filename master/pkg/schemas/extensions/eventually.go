// eventually allows for two-step validation, by only enforcing the specified subschemas
// during the completeness validation phase. This is a requirement specific to Determined.
//
// One use case is when it is necessary to enforce a `oneOf` on two fields that are
// `eventuallyRequired`. If the `oneOf` is evaluated during the sanity validation phase, it will
// always fail, if for example, the user is using cluster default values, but if validation
// for this subschema is held off until completeness validation, it will validate correctly.
//
// Example: eventually require one of connection string and account url to be specified:
//
// "eventually": {
//     "checks": {
//         "Exactly one of connection_string or account_url must be set": {
//             "oneOf": [
//                 {
//                     "eventuallyRequired": [
//                         "connection_string"
//                     ]
//                 },
//                 {
//                     "eventuallyRequired": [
//                         "account_url"
//                     ]
//                 }
//             ]
//         }
//     }
// }

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
