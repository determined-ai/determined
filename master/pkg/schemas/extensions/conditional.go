// See determined/common/schemas/extensions.py for the explanation of this and other extensions.
// See ./checks.go for notes on implementing extensions for the santhosh-tekuri/jsonschema package.

package extensions

import (
	"github.com/santhosh-tekuri/jsonschema/v2"
)

type conditional struct {
	// Test is either an "unless" or a "when" clause.
	Test *jsonschema.Schema
	// EnforceAfterPass will be "true" for "when" clauses or "false" for "unless" clauses.
	EnforceAfterPass bool
	// Enforce is the schema whose error will be shown to the user.
	Enforce *jsonschema.Schema
}

func conditionalCompile(ctx jsonschema.CompilerContext, m JSONObject) (interface{}, error) {
	rawConditional, ok := m["conditional"]
	if !ok {
		return nil, nil
	}
	cond := rawConditional.(JSONObject)

	var err error
	var test *jsonschema.Schema
	enforceAfterPass := true

	if rawWhen, ok := cond["when"]; ok {
		test, err = ctx.Compile(rawWhen)
		if err != nil {
			return nil, err
		}
	} else {
		test, err = ctx.Compile(cond["unless"])
		if err != nil {
			return nil, err
		}
		enforceAfterPass = false
	}

	enforce, err := ctx.Compile(cond["enforce"])
	if err != nil {
		return nil, err
	}

	return conditional{test, enforceAfterPass, enforce}, nil
}

func conditionalValidate(
	ctx jsonschema.ValidationContext, rawConditional interface{}, instance JSON,
) error {
	cond := rawConditional.(conditional)

	// Evaluate the Test clause.
	err := ctx.Validate(cond.Test, instance)
	passed := (err != nil)
	if cond.EnforceAfterPass == passed {
		return nil
	}

	// Evalutate the Enforce clause.
	err = ctx.Validate(cond.Enforce, instance)
	if err == nil {
		return nil
	}

	var x jsonschema.ValidationError
	return x.Group(ctx.Error("conditional", "conditional failed"), err)
}

// ConditionalExtension instantiates the conditional extension.
func ConditionalExtension() jsonschema.Extension {
	meta, err := jsonschema.CompileString("conditionalExtension.json", `{
		"properties" : {
			"conditional": {
				"additionalProperties": false,
				"required": ["enforce"],
				"properties": {
					"when": true,
					"unless": true,
					"enforce": true,
					"$comment": {"type": "string"}
				}
			}
		}
	}`)
	if err != nil {
		panic(err)
	}
	return jsonschema.Extension{
		Meta:     meta,
		Compile:  conditionalCompile,
		Validate: conditionalValidate,
	}
}
