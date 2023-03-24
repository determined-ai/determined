package schemas

import (
	"encoding/json"
	"fmt"
	"reflect"
	"strings"
)

// defaultIfDefaultable checks if obj implements our Defaultable psuedointerface and calls
// .WithDefaults if it does.
//
// The Defaultable psuedointerface is defined as:
//
//	"x.WithDefaults() returns another object with the same type as x".
//
// Defaultable is not a real go interface, it's more of a "psuedointerface".  See explanation on
// copyIfCopyable.
//
// In practice, Defaultable means an object can have custom behvaiors for schemas.WithDefaults.
// This is useful for implementing "runtime default" behaviors, like experiment seed or name.  It is
// also useful for working around types we do not own and which schemas.WithDefaults() would puke
// on.
func defaultIfDefaultable(obj reflect.Value) (reflect.Value, bool) {
	var out reflect.Value

	// Look for the .WithDefaults method.
	meth, ok := obj.Type().MethodByName("WithDefaults")
	if !ok {
		return out, false
	}

	// Verify the signature matches our Defaultable psuedointerface:
	// - one input (the receiver), and one output
	// - input type matches output type exactly (without the usual pointer receiver semantics)
	if meth.Type.NumIn() != 1 || meth.Type.NumOut() != 1 || meth.Type.In(0) != meth.Type.Out(0) {
		return out, false
	}

	// Psuedointerface matches, call the .WithDefaults method.
	out = meth.Func.Call([]reflect.Value{obj})[0]

	return out, true
}

// WithDefaults will recurse through structs, maps, and slices, setting default values for any
// struct fields whose struct implements the Defaultable pusedointerface.  This lets us read default
// values out of json-schema automatically.
//
// There are some forms of defaults which must be filled at runtimes, such as giving a default
// name to experiments with no name.  This can be accomplished by implementing
// the RuntimeDefaultable interface for that object.  See ExperimentConfig for an example.
//
// Example usage:
//
//	config, err := expconf.ParseAnyExperimentConfigYAML(bytes)
//
//	// Use the cluster checkpoint storage if the user did not specify one.
//	config.RawCheckpointStorage = schemas.Merge(
//	    config.RawCheckpointStorage, &cluster_default_storage
//	)
//
//	// Define any remaining undefined values.
//	config = schemas.WithDefaults(config)
func WithDefaults[T any](obj T) T {
	vObj := reflect.ValueOf(obj)
	name := fmt.Sprintf("%T", obj)
	return withDefaults(vObj, nil, name).Interface().(T)
}

func getDefaultSource(obj reflect.Value) interface{} {
	if schema, ok := obj.Interface().(Schema); ok {
		return schema.ParsedSchema()
	}
	return nil
}

func allocateWithDefaultBytes(typ reflect.Type, defaultBytes []byte) reflect.Value {
	// json.Unmarshal requires a pointer to work.
	ptr := reflect.New(typ)
	err := json.Unmarshal(defaultBytes, ptr.Interface())
	if err != nil {
		panic(
			fmt.Sprintf(
				"failed to unmarshal defaultBytes into %T: %q: %v",
				ptr.Interface(),
				string(defaultBytes),
				err.Error(),
			),
		)
	}
	return ptr.Elem()
}

// withDefaults is the recursive layer under WithDefaults.  withDefaults will return a clean copy
// of the original value, with defaults set.
func withDefaults(obj reflect.Value, defaultBytes []byte, name string) reflect.Value {
	// fmt.Printf("withDefaults on %v (%T)\n", name, obj.Interface())

	// Handle nil values and defaultBytes all in one place.
	if obj.Kind() == reflect.Interface ||
		obj.Kind() == reflect.Ptr ||
		obj.Kind() == reflect.Slice ||
		obj.Kind() == reflect.Map {
		if obj.IsZero() {
			if defaultBytes == nil {
				// Nil pointer with no defaultBytes means we are done recursing.
				return obj
			}
			// Use a clean copy of default bytes from obj, rather than a nil value.
			obj = allocateWithDefaultBytes(obj.Type(), defaultBytes)
		}
	}

	if out, ok := defaultIfDefaultable(obj); ok {
		return out
	}

	var out reflect.Value

	switch obj.Kind() {
	case reflect.Interface:
		out = withDefaults(obj.Elem(), defaultBytes, name)

	case reflect.Ptr:
		// Allocate the output pointer.
		out = reflect.New(obj.Type().Elem())
		// Recurse into the content of the object.
		out.Elem().Set(withDefaults(obj.Elem(), nil, name))

	case reflect.Struct:
		defaultSource := getDefaultSource(obj)
		out = reflect.New(obj.Type()).Elem()
		// Iterate through all the fields of the struct once, applying defaults.
		for i := 0; i < obj.NumField(); i++ {
			var fieldDefaultBytes []byte
			if defaultSource != nil {
				// Is there a default for this field's tag?
				fieldDefaultBytes = findDefaultInSchema(defaultSource, obj.Type().Field(i))
			}
			fieldName := fmt.Sprintf("%v.%v", name, obj.Type().Field(i).Name)
			// Recurse into the field.
			out.Field(i).Set(withDefaults(obj.Field(i), fieldDefaultBytes, fieldName))
		}

	case reflect.Slice:
		typ := reflect.SliceOf(obj.Type().Elem())
		out = reflect.MakeSlice(typ, 0, obj.Len())
		for i := 0; i < obj.Len(); i++ {
			elemName := fmt.Sprintf("%v[%v]", name, i)
			// Recurse into the elem (there's no per-element defaults yet).
			out = reflect.Append(out, withDefaults(obj.Index(i), nil, elemName))
		}

	case reflect.Map:
		typ := reflect.MapOf(obj.Type().Key(), obj.Type().Elem())
		out = reflect.MakeMap(typ)
		iter := obj.MapRange()
		for iter.Next() {
			key := iter.Key()
			val := iter.Value()
			elemName := fmt.Sprintf("%v.[%v]", name, key.Interface())
			// Recurse into the elem (there's no per-element defaults yet).
			out.SetMapIndex(key, withDefaults(val, nil, elemName))
		}

	// Assert that none of the "complex" kinds are present.
	case reflect.Array,
		reflect.Chan,
		reflect.Func,
		reflect.UnsafePointer:
		panic(fmt.Sprintf(
			"unable to withDefaults at %v of type %T, kind %v", name, obj.Interface(), obj.Kind(),
		))

	default:
		out = cpy(obj)
	}

	// fmt.Printf("withDefaults on %v (%T) returning %T\n", name, obj.Interface(), out.Interface())

	// Always return the matching type.
	return out.Convert(obj.Type())
}

// jsonNameFromJSONTag is based on encoding/json's parseTag().
func jsonNameFromJSONTag(tag string) string {
	if idx := strings.Index(tag, ","); idx != -1 {
		return tag[:idx]
	}
	return tag
}

// findDefaultInSchema takes a json-schema and a StructField, and tries to use the json tag on the
// StructField to find a byte string that represents the json value of the default for the field.
//
// It looks for defaults as `properties.JSONTAG.default`, and returns nil if none was found.  It
// also returns nil if the bytes found match `null` exactly.
//
// For example, with the schema:
//
//	{
//	    "properties": {
//	        "hello": {
//	            "type": ["string", "null"],
//	            "default": "world"
//	        }
//	     }
//	 }
//
// and with the struct:
//
//	type X struct {
//	    Hello    string `json:"hello"`
//	}
//
// then findDefaultInSchema(schema, reflect.TypeOf(x).FieldByName("Hello")) returns "world".
func findDefaultInSchema(schema interface{}, field reflect.StructField) []byte {
	jsonTag, ok := field.Tag.Lookup("json")
	if !ok {
		return nil
	}

	jsonName := jsonNameFromJSONTag(jsonTag)

	schemaObj, ok := schema.(map[string]interface{})
	if !ok {
		return nil
	}

	propertiesObj, ok := schemaObj["properties"].(map[string]interface{})
	if !ok {
		return nil
	}

	property, ok := propertiesObj[jsonName].(map[string]interface{})
	if !ok {
		return nil
	}

	defaultObj, ok := property["default"]
	if !ok {
		return nil
	}

	if defaultObj == nil {
		// Don't marshal nil values into []byte("null").
		return nil
	}

	byts, err := json.Marshal(defaultObj)
	if err != nil {
		panic("somehow json failed to remarshal")
	}

	return byts
}
