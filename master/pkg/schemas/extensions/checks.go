// See determined_common/schemas/extensions.py for the explanation of this and other extensions.

// This file is a tutorial for implementing extensions for the santhosh-tekuri/jsonschema package.
//
// A jsonschema extension consists of three parts:
//
//  - A compile function.  Compiling happens once before any validations and takes the schema as
//    input.  The extension's compile function will be called each time the extension appears in
//    the schema.  The return value of the compile function will be fed to the validation function.
//
//  - A validate function.  Validation happens once for each instance of data that is checked.  The
//    extension's validate function is called once for each time the compile function was caused,
//    for each overall validation.  Each call to the validate function receives the output from one
//    call to the compile function and corresponds to a particular appearance of the extension in
//    the schema.
//
//  - A metaschema which describes how the *schema* that uses the extension is allowed to look.

package extensions

import (
	"github.com/santhosh-tekuri/jsonschema/v2"
)

// checksCompile is called for each time the extension appears in the schema.  The input object is
// the point in the schema which *contains* the extension.  The object returned from the compile
// function will be passed to the validate function.
func checksCompile(ctx jsonschema.CompilerContext, m JSONObject) (interface{}, error) {
	rawChecks, ok := m["checks"]
	if !ok {
		return nil, nil
	}

	// checks is just a map of custom error messages to jsonschema checks.  Example:
	//
	//    "checks": {
	//        "you must specify an entrypoint that references the trial class":{
	//            ... (schema which allows Native API or requires that entrypoint is set) ...
	//        },
	//        "you requested a bayesian search but hyperband is way better": {
	//            ... (schema which checks if you try searcher.name=baysian) ...
	//        }
	//    }
	checks := rawChecks.(JSONObject)

	// With this package, the way to descend into subschemas during validations is to compile them
	// here in the compile function, so we can call them later during validation.
	compiled := map[string]*jsonschema.Schema{}
	for msg, rawSchema := range checks {
		schema, err := ctx.Compile(rawSchema)
		if err != nil {
			return nil, err
		}
		compiled[msg] = schema
	}
	return compiled, nil
}

// checksValidate is called once for each time the extension appears in the schema, during each
// validation.  Each call to validate during a single validation corresponds to one of the calls
// to compile, meaning that it corresponds to a particular appearance of the extension in the
// schema, and meaning that it gets the output of that particular compile call as one of its
// inputs (the 'compiled' parameter).  The other input is the instance to be validated at this
// point in the schema.
func checksValidate(
	ctx jsonschema.ValidationContext, rawCompiled interface{}, instance JSON,
) error {
	// rawCompiled is the interface{}-typed output of our compile function.
	compiled := rawCompiled.(map[string]*jsonschema.Schema)

	// We will gather up all of the custom error messgages that we need to return in this list.
	var errors []error

	for msg, schema := range compiled {
		// Descend into a subschema, using ctx rather than the external API for validation.
		err := ctx.Validate(schema, instance)
		if err != nil {
			// Return the custom error message for this check (which is just the map key).
			errors = append(errors, ctx.Error("checks", msg))
		}
	}

	if len(errors) == 0 {
		// Woohoo, no errors.  Just return nil.
		return nil
	}

	// jsonschema's ValidationError is a nestable error.  We return a single error corresponding to
	// our extension (a "checks failed" error) with child errors corresponding to the "useful"
	// errors that we generated from checking subschemas.  Ultimately, only the leaves of this tree
	// of errors are useful at all.
	//
	// Also, jsonschema makes questionable use of object-oriented programming for generating the
	// errors, hence this odd x variable.
	var x jsonschema.ValidationError
	return x.Group(ctx.Error("checks", "checks failed"), errors...)
}

// ChecksExtension creates the metaschema and returns the full jsonschema.Extension object, gluing
// together the metaschema, the compile function, and the validate function.
func ChecksExtension() jsonschema.Extension {
	meta, err := jsonschema.CompileString("checksExtension.json", `{
		"properties" : {
			"checks": {
				"additionalProperties": { "type": "object" }
			}
		}
	}`)
	if err != nil {
		panic(err)
	}
	return jsonschema.Extension{
		Meta:     meta,
		Compile:  checksCompile,
		Validate: checksValidate,
	}
}
