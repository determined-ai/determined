// See determined_common/schemas/extensions.py for the explanation of this and other extensions.
// See ./checks.go for notes on implementing extensions for the santhosh-tekuri/jsonschema package.

package extensions

import (
	"github.com/santhosh-tekuri/jsonschema/v2"
)

func optionalRefCompile(ctx jsonschema.CompilerContext, m JSONObject) (interface{}, error) {
	rawOptionalRef, ok := m["optionalRef"]
	if !ok {
		return nil, nil
	}

	// optionalRef behaves identical to a normal $ref, except it allows the $ref to be null, so
	// create a {"$ref": "..."} schema and compile that.
	ref := map[string]JSON{}
	ref["$ref"] = rawOptionalRef
	return ctx.Compile(ref)
}

func optionalRefValidate(
	ctx jsonschema.ValidationContext, rawRefSchema interface{}, instance JSON,
) error {
	// Allow nulls; this is the optional part of optionalRef.
	if instance == nil {
		return nil
	}

	// Otherwise enforce the normal $ref.
	refSchema := rawRefSchema.(*jsonschema.Schema)
	return ctx.Validate(refSchema, instance)
}

// OptionalRefExtension creates the metaschema and returns the full jsonschema.Extension object,
// gluing together the metaschema, the compile function, and the validate function.
func OptionalRefExtension() jsonschema.Extension {
	meta, err := jsonschema.CompileString("optionalRefExtension.json", `{
		"properties" : {
			"optionalRef": {
				"type": "string"
			}
		}
	}`)
	if err != nil {
		panic(err)
	}
	return jsonschema.Extension{
		Meta:     meta,
		Compile:  optionalRefCompile,
		Validate: optionalRefValidate,
	}
}
