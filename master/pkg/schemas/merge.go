package schemas

import (
	"fmt"
	"reflect"
)

// Mergable means an object can have custom behvaiors for schemas.Merge.
type Mergable interface {
	// Merge takes a non-nil version of itself and merges into itself.
	Merge(interface{})
}

// Merge will recurse through structs, setting empty values in obj with an non-empty values (copy
// semantics).  Both obj and src must be the same type of obj, but obj must be a pointer so that it
// is settable.
//
// The default behavior for merging maps is to merge keys from src to obj, and the default
// behavior for slices is to copy them.  This is analgous to how json.Unmarshal treats maps and
// slices.  However, the default merging behavior for an object can be overwritten by implementing
// the Mergable interface.  An example of this is BindMountsConfig.
//
// Merge is intelligent enough to handle union types automatically.  In those cases, Merge is
// recursive as log as the obj getting filled either does not have any of the union types defined
// or if it has the same union type defined as the src.  That is, a S3 checkpoint storage object
// will never be used as a src to try to fill a SharedFS checkpoint storage object.
//
// Example usage:
//
//    config, err := expconf.ParseAnyExperimentConfigYAML(bytes)
//
//    var cluster_default_checkpoint_storage expconf.CheckpointStorage = ...
//
//    // Use the cluster checkpoint storage if the user did not specify one.
//    schemas.Merge(&config.CheckpointStorage, cluster_default_checkpoint_storage)
//
func Merge(obj interface{}, src interface{}) {
	name := derefType(reflect.TypeOf(obj)).Name()
	merge(reflect.ValueOf(obj), reflect.ValueOf(src), name)
}

func mergeOne(obj reflect.Value, src reflect.Value, name string, allocated bool) {
	// If the object is Mergable, we only call Merge and nothing else.
	if mergeable, ok := obj.Addr().Interface().(Mergable); ok {
		mergeable.Merge(src.Interface())
		return
	}

	switch src.Kind() {
	case reflect.Struct:
		// Don't recurse yet; just ignore structs for now.

	case reflect.Map:
		// Maps get fused together; all input keys are written into the output map.
		for _, key := range src.MapKeys() {
			// Ensure key is not already set in obj.
			if objVal := obj.MapIndex(key); objVal.IsValid() {
				continue
			}
			elemName := fmt.Sprintf("%v.[%v]", name, key.Interface())
			val := src.MapIndex(key)

			// Since Map objects always need to be copied (we know that cpy needs to be filled),
			// we need to get a pointer to a nil pointer to the type.  Otherwise, when we recurse,
			// if the map's value type is not a pointer, the objAlreadySet check will prevent the
			// copy from happening.  That's the purpose of the PtrTo() within the New().
			cpy := reflect.New(reflect.PtrTo(val.Type()))

			merge(cpy, val, elemName)

			// Update the original value with the defaulted value.
			// Deref both layers of pointers we created.
			obj.SetMapIndex(key, cpy.Elem().Elem())
		}

	case reflect.Slice:
		// Slices get copied only if the original was nil.
		if allocated {
			for i := 0; i < src.Len(); i++ {
				val := src.Index(i)
				elemName := fmt.Sprintf("%v.[%d]", name, i)
				// Similar to map, we use an extra layer of indirection to force a copy to happen.
				cpy := reflect.New(reflect.PtrTo(val.Type()))
				merge(cpy, val, elemName)
				obj.Set(reflect.Append(obj, cpy.Elem().Elem()))
			}
		}

	// Assert that none of the "complex" kinds are present (or Ptr, which we should have deref'ed).
	case reflect.Array,
		reflect.Chan,
		reflect.Func,
		reflect.Interface,
		reflect.UnsafePointer,
		reflect.Ptr:
		panic(fmt.Sprintf("unable to fill %T with %T at %v", obj, src, name))

	default:
		// Simple types get a simple copy, but only if the original was nil.
		if allocated {
			obj.Set(src)
		}
	}
}

// merge is the recursive layer under Merge.
func merge(obj reflect.Value, src reflect.Value, name string) {
	// Stop recursing if there's nothing to copy.
	src, ok := derefInput(src)
	if !ok {
		return
	}

	// Fill any nil pointers.
	var objAllocated bool
	obj, objAllocated = derefOutput(obj)

	// Make sure that the types match.  This function does not handle cross-type operations.
	if obj.Type() != src.Type() {
		panic(
			fmt.Sprintf(
				"type mismatch in merge; can't fill %T with %T",
				obj.Interface(),
				src.Interface(),
			),
		)
	}

	mergeOne(obj, src, name, objAllocated)

	// Recurse into structs.
	if src.Kind() == reflect.Struct {
		// Detect what to do with union fields.  There are 4 important cases:
		//  1. src has a union member, obj does not -> recurse into that field.
		//  2. src has a union member, obj has the same one -> recurse into that field.
		//  3. src has a union member, obj has the different one -> do not recurse.
		//  4. src has no union member -> recursing is a noop and doesn't matter
		// Logically, this reduces to:
		//   - if obj has a union member, src does not have the same one -> don't recurse.
		//   - else -> recurse
		recurseIntoUnion := true
		for i := 0; i < src.NumField(); i++ {
			structField := src.Type().Field(i)
			if _, ok := structField.Tag.Lookup("union"); ok {
				if !obj.Field(i).IsZero() && src.Field(i).IsZero() {
					recurseIntoUnion = false
					break
				}
			}
		}
		// Recurse into each field of the struct.
		for i := 0; i < src.NumField(); i++ {
			structField := src.Type().Field(i)
			srcField := src.Field(i)
			objField := obj.Field(i)
			if _, ok := structField.Tag.Lookup("union"); ok && !recurseIntoUnion {
				continue
			}
			fieldName := fmt.Sprintf("%v.%v", name, structField.Name)
			merge(objField.Addr(), srcField, fieldName)
		}
	}
}
