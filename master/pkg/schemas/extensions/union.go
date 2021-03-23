// See determined/common/schemas/extensions.py for the explanation of this and other extensions.
// See ./checks.go for notes on implementing extensions for the santhosh-tekuri/jsonschema package.

package extensions

import (
	"fmt"
	"strings"

	"github.com/santhosh-tekuri/jsonschema/v2"
)

type unionItem struct {
	Schema    *jsonschema.Schema
	Key       string
	RawSchema JSON
}

type unionSchema struct {
	Message string
	Items   []unionItem
}

func unionCompile(ctx jsonschema.CompilerContext, m JSONObject) (interface{}, error) {
	rawUnion, ok := m["union"]
	if !ok {
		return nil, nil
	}

	union := rawUnion.(JSONObject)

	// Store the default message.
	rawDefaultMessage, ok := union["defaultMessage"]
	defaultMessage := "union failed to validate"
	if ok {
		msgString := rawDefaultMessage.(string)
		defaultMessage = msgString
	}

	// Compile the child schemas.
	var items []unionItem
	rawItems := union["items"].(JSONArray)
	for _, rawItem := range rawItems {
		schema, err := ctx.Compile(rawItem)
		if err != nil {
			return nil, err
		}

		item := rawItem.(JSONObject)
		rawKey := item["unionKey"]
		key := rawKey.(string)
		items = append(items, unionItem{schema, key, rawItem})
	}

	return unionSchema{defaultMessage, items}, nil
}

func unionValidate(
	ctx jsonschema.ValidationContext, rawUnion interface{}, instance JSON,
) error {
	union := rawUnion.(unionSchema)

	// We will only return one error message, which should be the error where the unionKey
	// evaluates to true.
	var selectedError error

	// We should only have one subschema which validates as true.
	var valid []JSON

	for _, item := range union.Items {
		err := ctx.Validate(item.Schema, instance)
		if err != nil {
			if selectedError == nil {
				// Is this the error we want to show to users?
				if evaluateUnionKey(item.Key, instance) {
					selectedError = err
				}
			}
		} else {
			valid = append(valid, item.RawSchema)
		}
	}

	if len(valid) == 1 {
		// no errors
		return nil
	}

	if len(valid) > 1 {
		return ctx.Error("union", "bug in validation! Multiple schemas matched: %v", valid)
	}

	if selectedError != nil {
		var x jsonschema.ValidationError
		return x.Group(ctx.Error("union", union.Message), selectedError)
	}

	return ctx.Error("union", union.Message)
}

// UnionExtension instantiates the union extension.
func UnionExtension() jsonschema.Extension {
	meta, err := jsonschema.CompileString("union.json", `{
		"properties" : {
			"defaultMessage": { "type": "string" },
			"union": {
				"type": "object",
				"additionalProperties": false,
				"required": ["items"],
				"properties": {
					"defaultMessage": { "type": "string" },
					"items": {
						"type": "array",
						"items": {
							"type": "object",
							"required": ["unionKey"],
							"properties": {
								"unionKey": { "type": "string" }
							}
						}
					}
				}
			}
		}
	}`)
	if err != nil {
		panic(err)
	}
	return jsonschema.Extension{
		Meta:     meta,
		Compile:  unionCompile,
		Validate: unionValidate,
	}
}

func evaluateUnionKey(key JSON, instance JSON) bool {
	switch tKey := key.(type) {
	case string:
		// Parse the string and evaluate.
		switch {
		case tKey == "always":
			return true

		case tKey == "never":
			return false

		case strings.HasPrefix(tKey, "not:"):
			return !evaluateUnionKey(tKey[len("not:"):], instance)

		case strings.HasPrefix(tKey, "const:"):
			split := strings.SplitN(tKey[len("const:"):], "=", 2)
			if len(split) != 2 {
				panic("invalid unionKey")
			}
			name := split[0]
			value := split[1]

			tInstance, ok := instance.(JSONObject)
			if !ok {
				return false
			}

			instanceValue, ok := tInstance[name]
			if !ok {
				return false
			}

			tInstanceValue, ok := instanceValue.(string)
			if !ok {
				return false
			}

			return value == tInstanceValue

		case strings.HasPrefix(tKey, "singleproperty:"):
			name := tKey[len("singleproperty:"):]

			tInstance, ok := instance.(JSONObject)
			if !ok {
				return false
			}

			if len(tInstance) != 1 {
				return false
			}

			_, ok = tInstance[name]
			return ok

		case strings.HasPrefix(tKey, "type:"):
			typ := tKey[len("type:"):]

			switch typ {
			case "array":
				_, ok := instance.(JSONArray)
				return ok
			case "object":
				_, ok := instance.(JSONObject)
				return ok
			}

		case strings.HasPrefix(tKey, "hasattr:"):
			attr := tKey[len("hasattr:"):]

			tInstance, ok := instance.(JSONObject)
			if !ok {
				return false
			}

			_, ok = tInstance[attr]
			return ok
		}
	}
	panic(fmt.Sprintf("invalid unionKey: %v", key))
}
