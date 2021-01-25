package schemas

import (
	"bytes"
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
// the RuntimeDefaultable interface for that object.  See ExperimentConfig for and example.
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
func FillDefaults(x interface{}) {
	// Enter the recursive default filling with no default bytes for the root object (which must
	// already exist), and starting with the name of the object type.
	name := derefType(reflect.TypeOf(x)).Name()
	fillDefaults(reflect.ValueOf(x), nil, name)
}

// fillOneDefault is the helper function to fillDefaults which actually sets values.
func fillOneDefault(obj reflect.Value, defaultBytes []byte, name string) {
	if defaultBytes == nil {
		return
	}

	// Dereference once to get the thing we are setting.
	obj = obj.Elem()

	if obj.Type().Kind() != reflect.Ptr {
		// Enforce good hygiene on structs.
		// Whoa, it's not valid to have a default value for a non-pointer value.  Bug.
		panic(
			fmt.Sprintf(
				"have defaultBytes (%v) for %v but it is not a pointer type (%T)!\n",
				string(defaultBytes),
				name,
				obj.Interface(),
			),
		)
	}

	// If the value is already set, no need to fill a default.
	if !obj.IsZero() {
		return
	}

	// If the default value would be nil, exit now so we don't allocate anything.
	if bytes.Compare(defaultBytes, []byte("null")) == 0 {
		return
	}

	// Fill any nil pointers with empty, allocated pointers.  Use the Addr() to
	// preserve the settability.
	fillable, _ := derefOutput(obj)
	fillable = fillable.Addr()
	// Unmarshal the default value into the object.  This will never override a
	// predefined user value because we only got here for nil-pointer-valued fields.
	err := json.Unmarshal(defaultBytes, fillable.Interface())
	if err != nil {
		panic(
			fmt.Sprintf(
				"failed to unmarshal defaultBytes into %T: %v",
				fillable.Interface(),
				string(defaultBytes),
			),
		)
	}
}

// fillDefaults is the recursive layer under FillDefaults.
func fillDefaults(obj reflect.Value, defaultBytes []byte, name string) {
	// Set the default value for this obj as appropriate.
	fillOneDefault(obj, defaultBytes, name)

	// Don't recurse into nil objects.  We check after the defaultBytes are processed in case the
	// defaultBytes are an empty object that can be recursed into.
	var ok bool
	obj, ok = derefInput(obj)
	if !ok {
		return
	}

	// Avoid recursing into types from external packages, which can't be Defaultable anyway.
	if !strings.Contains(obj.Type().PkgPath(), "determined/master/pkg/schemas") {
		return
	}

	// Get a source of defaults from a Defaultable or Schema interface.
	// Use PtrTo/Addr so that if the DefaultSource is defined on a struct pointer, it still works.
	var defaultSource interface{}
	if defaultable, ok := obj.Addr().Interface().(Defaultable); ok {
		fmt.Printf("%T is defaultable\n", obj.Interface())
		defaultSource = defaultable.DefaultSource()
	}else if schema, ok := obj.Addr().Interface().(Schema); ok {
		fmt.Printf("%T is schema\n", obj.Interface())
		defaultSource = schema.ParsedSchema()
	}

	switch obj.Kind() {
	case reflect.Struct:
		// Iterate through all the fields of the struct once, applying defaults.
		for i := 0; i < obj.NumField(); i++ {
			var fieldDefaultBytes []byte
			if defaultSource != nil {
				// Is there a default for this field's tag?
				fieldDefaultBytes = findDefaultInSchema(defaultSource, obj.Type().Field(i))
			}
			fieldName := fmt.Sprintf("%v.%v", name, obj.Type().Field(i).Name)
			// Recurse into the field.
			fillDefaults(obj.Field(i).Addr(), fieldDefaultBytes, fieldName)
		}

	case reflect.Slice:
		for i := 0; i < obj.Len(); i++ {
			elemName := fmt.Sprintf("%v.[%v]", name, i)
			// Recurse into the elem (there's no per-element defaults yet).
			fillDefaults(obj.Index(i).Addr(), nil, elemName)
		}

	case reflect.Map:
		for _, key := range obj.MapKeys() {
			elemName := fmt.Sprintf("%v.[%v]", name, key.Interface())
			val := obj.MapIndex(key)
			// cpy is a copy of the object that is settable.
			cpy := reflect.New(val.Type())
			cpy.Elem().Set(val)
			// Recurse into the elem (there's no per-element defaults yet).
			fillDefaults(cpy, nil, elemName)
			// Update the original value with the defaulted value.
			obj.SetMapIndex(key, cpy.Elem())
		}

	// Assert that none of the "complex" kinds are present (or Ptr, which we should have deref'ed).
	case reflect.Array,
		reflect.Chan,
		reflect.Func,
		reflect.Interface,
		reflect.UnsafePointer,
		reflect.Ptr:
		panic(fmt.Sprintf("unable to fillDefaults at %v of type %T, kind %v", name, obj.Interface(), obj.Kind()))
	}

	// AFTER the automatic defaults, we apply any runtime defaults.  This way, we've already filled
	// any nil pointers with valid objects.
	if runtimeDefaultable, ok := obj.Addr().Interface().(RuntimeDefaultable); ok {
		runtimeDefaultable.RuntimeDefaults()
	}
}

// jsonNameFromJsonTag is based on encoding/json's parseTag().
func jsonNameFromJsonTag(tag string) string {
	if idx := strings.Index(tag, ","); idx != -1 {
		return tag[:idx]
	}
	return tag
}

// findDefaultInSchema takes a json-schema and a StructField, and tries to use the json tag on the
// StructField to find a byte string that represents the json value of the deafult for the field.
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
	jsonName := jsonNameFromJsonTag(jsonTag)

	schemaObj := schema.(map[string]interface{})
	propertiesObj := schemaObj["properties"].(map[string]interface{})
	property := propertiesObj[jsonName].(map[string]interface{})

	defaultObj, ok := property["default"]
	if !ok {
		return nil
	}

	byts, err := json.Marshal(defaultObj)
	if err != nil {
		panic("somehow json failed to remarshal")
	}

	return byts
}
