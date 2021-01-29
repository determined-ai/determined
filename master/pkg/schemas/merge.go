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
	// fmt.Printf("--------------------\n")
	name := fmt.Sprintf("%T", obj)
	merge(reflect.ValueOf(obj), reflect.ValueOf(src), false, name)
}

func assertTypeMatch(obj reflect.Value, src reflect.Value) {
	if obj.Type() == src.Type() {
		return
	}
	panic(
		fmt.Sprintf(
			"type mismatch in merge; can't fill %T with %T",
			obj.Interface(),
			src.Interface(),
		),
	)
}

// merge is the recursive layer under Merge.
func merge(obj reflect.Value, src reflect.Value, allocated bool, name string) {
	// fmt.Printf("merge(%T, %T)\n", obj.Interface(), src.Interface())
	// First deref src.  If it is ultimately nil, stop.
	for src.Kind() == reflect.Ptr || src.Kind() == reflect.Interface {
		if src.IsZero() {
			return
		}
		// Recurse with dereferenced src.
		merge(obj, src.Elem(), allocated, name)
		return
	}

	// If the object is Mergable, we only call Merge and nothing else.
	if mergeable, ok := obj.Interface().(Mergable); ok {
		mergeable.Merge(src.Interface())
		return
	}

	// obj should always be a pointer, because Merge(&x, y) will act on x in-place.
	if obj.Kind() != reflect.Ptr {
		panic("non-pointer in merge")
	}
	// obj can't be a nil pointer, because Merge(nil, y) doesn't make any sense.
	if obj.IsZero() {
		panic("nil pointer in merge")
	}
	// Now operate on what obj points to.
	obj = obj.Elem()

	switch obj.Kind() {
	case reflect.Interface:
		if obj.IsZero() {
			// This doesn't make any sense; we need a type.
			panic("got a nil interface as the obj to merge into")
		}
		// Dereference the type but not the original pointer.
		merge(obj.Elem().Addr(), src, allocated, name)

	case reflect.Ptr:
		// Note that this is a double-pointer since we already dereference the object once.
		if obj.IsZero() {
			// Allocate, then recurse with allocated = true.
			tmp := reflect.New(obj.Type().Elem())
			obj.Set(tmp)
			merge(obj, src, true, name)
		} else {
			merge(obj, src, false, name)
		}

	case reflect.Struct:
		assertTypeMatch(obj, src)
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
			merge(objField.Addr(), srcField, false, fieldName)
		}

	case reflect.Map:
		assertTypeMatch(obj, src)
		// Maps get fused together; all input keys are written into the output map.
		for _, key := range src.MapKeys() {
			// Ensure key is not already set in obj.
			if objVal := obj.MapIndex(key); objVal.IsValid() {
				continue
			}
			elemName := fmt.Sprintf("%v.[%v]", name, key.Interface())
			val := src.MapIndex(key)

			cpy := reflect.New(val.Type())
			merge(cpy, val, true, elemName)

			// Update the original value with the defaulted value.
			// Deref both layers of pointers we created.
			obj.SetMapIndex(key, cpy.Elem())
		}

	case reflect.Slice:
		assertTypeMatch(obj, src)
		if !allocated {
			// Don't override non-newly-allocated slices.
			return
		}
		// Slices get copied only if the original was nil.
		for i := 0; i < src.Len(); i++ {
			val := src.Index(i)
			elemName := fmt.Sprintf("%v.[%d]", name, i)
			cpy := reflect.New(val.Type())
			merge(cpy, val, true, elemName)
			obj.Set(reflect.Append(obj, cpy.Elem()))
		}

	// Assert that none of the "complex" kinds are present.
	case reflect.Array,
		reflect.Chan,
		reflect.Func,
		reflect.UnsafePointer:
		panic(fmt.Sprintf("unable to fill %T with %T at %v", obj, src, name))

	default:
		assertTypeMatch(obj, src)
		if !allocated {
			// Don't override non-newly-allocated values.
			return
		}
		obj.Set(src)
	}
}
