package schemas

import (
	"encoding/json"
	"fmt"
	"reflect"
	"strings"
)

// RuntimeDefaultable means there are runtime values for filling in an object, like choosing a
// random seed based on the wall clock.
type RuntimeDefaultable interface {
	// RuntimeDefaults must apply the runtime-defined default values.
	RuntimeDefaults()
}

// Defaultable means a struct can have its defaults filled in automatically.
type Defaultable interface {
	// DefaultSource must return a parsed json-schema object in which to find defaults.
	DefaultSource() interface{}
}

// FillDefaults will recurse through structs, maps, and slices, setting default values for any
// struct fields whose struct implements the Defaultable interface.  This lets us read default
// values out of json-schema automatically.
//
// There are some forms of defaults which must be filled at runtimes, such as giving a default
// description to experiments with no description.  This can be accomplished by implementing
// the RuntimeDefaultable interface for that object.  See ExperimentConfig for an example.
//
// There are some objects which get their defaults from other objects' defaults.  This an
// unforutnate detail of our union types which have common members that appear on the root union
// object.  That's hard to reason about, and we should avoid doing that in new config objects.  But
// those objects implement DefaultSource() to customize that behavior.
//
// Example usage:
//
//    config, err := expconf.ParseAnyExperimentConfigYAML(bytes)
//
//    // Use the cluster checkpoint storage if the user did not specify one.
//    schemas.Merge(&config.CheckpointStorage, cluster_default_checkpoint_storage)
//
//    // Define any remaining undefined values.
//    schemas.FillDefaults(&config)
//
func FillDefaults(obj interface{}) {
	vObj := reflect.ValueOf(obj)
	// obj can't be a non-pointer, because it edits in-place.
	if vObj.Kind() != reflect.Ptr {
		panic("FillDefaults must be called on a pointer")
	}
	// obj can't be a nil pointer, because FillDefaults(nil) doesn't make any sense.
	if vObj.IsZero() {
		panic("FillDefaults must be called on a non-nil pointer")
	}
	// Enter the recursive default filling with no default bytes for the root object (which must
	// already exist), and starting with the name of the object type.
	name := fmt.Sprintf("%T", obj)
	vObj.Elem().Set(fillDefaults(vObj.Elem(), nil, false, name))
}

func fillDefaults(obj reflect.Value, defaultBytes []byte, inPtr bool, name string) reflect.Value {
	switch obj.Kind() {
	case reflect.Interface:
		if obj.IsZero() {
			// This doesn't make any sense; we need a type.
			panic("got a nil interface as the obj to FillDefaults into")
		}
		obj.Elem().Set(fillDefaults(obj.Elem(), defaultBytes, inPtr, name))

	case reflect.Ptr:
		if obj.IsZero() {
			if defaultBytes == nil {
				// Nil pointer with no defaultBytes means we are done recursing.
				return obj
			}
			// Otherwise, since we have default bytes, allocate the new object.
			obj = reflect.New(obj.Type().Elem())
			// Fill the object with default bytes.
			err := json.Unmarshal(defaultBytes, obj.Interface())
			if err != nil {
				panic(
					fmt.Sprintf(
						"failed to unmarshal defaultBytes into %T: %v",
						obj.Interface(),
						string(defaultBytes),
					),
				)
			}
			// We already consumed defaultBytes, so set it to nil when we recurse.
			obj.Elem().Set(fillDefaults(obj.Elem(), nil, true, name))
		} else {
			// Recurse into the element inside the pointer.
			obj.Elem().Set(fillDefaults(obj.Elem(), defaultBytes, true, name))
		}

	case reflect.Struct:
		// Get a source of defaults from a Defaultable or Schema interface.
		// Use Addr so that if the DefaultSource is defined on a struct pointer, it still works.
		var defaultSource interface{}
		if defaultable, ok := obj.Addr().Interface().(Defaultable); ok {
			defaultSource = defaultable.DefaultSource()
		} else if schema, ok := obj.Addr().Interface().(Schema); ok {
			defaultSource = schema.ParsedSchema()
		}

		// Iterate through all the fields of the struct once, applying defaults.
		for i := 0; i < obj.NumField(); i++ {
			var fieldDefaultBytes []byte
			if defaultSource != nil {
				// Is there a default for this field's tag?
				fieldDefaultBytes = findDefaultInSchema(defaultSource, obj.Type().Field(i))
			}
			fieldName := fmt.Sprintf("%v.%v", name, obj.Type().Field(i).Name)
			// Recurse into the field.
			fillDefaults(obj.Field(i).Addr(), fieldDefaultBytes, false, fieldName)
		}

	case reflect.Slice:
		for i := 0; i < obj.Len(); i++ {
			elemName := fmt.Sprintf("%v.[%v]", name, i)
			// Recurse into the elem (there's no per-element defaults yet).
			fillDefaults(obj.Index(i).Addr(), nil, false, elemName)
		}

	case reflect.Map:
		for _, key := range obj.MapKeys() {
			elemName := fmt.Sprintf("%v.[%v]", name, key.Interface())
			val := obj.MapIndex(key)
			// Recurse into the elem (there's no per-element defaults yet).
			tmp := fillDefaults(val, nil, false, elemName)
			// Update the original value with the defaulted value.
			obj.SetMapIndex(key, tmp)
		}

	// Assert that none of the "complex" kinds are present.
	case reflect.Array,
		reflect.Chan,
		reflect.Func,
		reflect.UnsafePointer:
		panic(fmt.Sprintf(
			"unable to fillDefaults at %v of type %T, kind %v", name, obj.Interface(), obj.Kind(),
		))

	default:
		// When we have reached a "simple" value (string, int, etc), if we have default bytes but
		// we are not inside a pointer, that means that there is a bug in the golang struct.
		if defaultBytes != nil && !inPtr {
			panic(
				fmt.Sprintf(
					"have defaultBytes (%v) for %v but it is not a pointer type (%T)!\n",
					string(defaultBytes),
					name,
					obj.Interface(),
				),
			)
		}
	}

	// AFTER the automatic defaults, we apply any runtime defaults.  This way, we've already filled
	// any nil pointers with valid objects.
	if runtimeDefaultable, ok := obj.Interface().(RuntimeDefaultable); ok {
		runtimeDefaultable.RuntimeDefaults()
	}

	return obj
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
//     {
//         "properties": {
//             "hello": {
//                 "type": ["string", "null"],
//                 "default": "world"
//             }
//      }
//
// and with the struct:
//
//     type X struct {
//         Hello    string `json:"hello"`
//     }
//
// then findDefaultInSchema(schema, reflect.TypeOf(x).FieldByName("Hello")) returns "world".
func findDefaultInSchema(schema interface{}, field reflect.StructField) []byte {
	// By default we'll return (nil, false), which corresponds to "no default found".
	defer func() {
		_ = recover()
	}()

	jsonTag := field.Tag.Get("json")
	jsonName := jsonNameFromJSONTag(jsonTag)

	schemaObj := schema.(map[string]interface{})
	propertiesObj := schemaObj["properties"].(map[string]interface{})
	property := propertiesObj[jsonName].(map[string]interface{})

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
