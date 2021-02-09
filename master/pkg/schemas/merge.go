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
	name := fmt.Sprintf("%T", obj)

	vObj := reflect.ValueOf(obj)
	vSrc := reflect.ValueOf(src)

	// obj should always be a pointer, because Merge(&x, y) will act on x in-place.
	if vObj.Kind() != reflect.Ptr {
		panic("non-pointer in merge")
	}
	// obj can't be a nil pointer, because Merge(nil, y) doesn't make any sense.
	if vObj.IsZero() {
		panic("nil pointer in merge")
	}

	// *obj must have the same type as src.
	assertTypeMatch(vObj.Elem(), vSrc)

	// Think: *obj = merge(*obj, src)
	vObj.Elem().Set(merge(vObj.Elem(), vSrc, name))
}

func assertTypeMatch(obj reflect.Value, src reflect.Value) {
	if obj.Type() == src.Type() {
		return
	}
	panic(
		fmt.Sprintf(
			"type mismatch in merge; can't merge %T into %T",
			src.Interface(),
			obj.Interface(),
		),
	)
}

// merge is the recursive layer under Merge.  obj and src must always have the same type, and the
// return type will also be the same.  The return value will never share memory with src, so it is
// safe to alter obj without affecting src after the fact.
func merge(obj reflect.Value, src reflect.Value, name string) reflect.Value {
	// fmt.Printf("merge(%T, %T, %v)\n", obj.Interface(), src.Interface(), name)
	assertTypeMatch(obj, src)

	// If src is nil, return obj unmodified.
	if src.Kind() == reflect.Ptr || src.Kind() == reflect.Interface {
		if src.IsZero() {
			return cpy(obj)
		}
	}

	// Handle nil pointers by simply copying the src.
	if obj.Kind() == reflect.Ptr && obj.IsZero() {
		return cpy(src)
	}

	// If the object is Mergable, we only call Merge on it and return it as-is.
	if mergeable, ok := obj.Addr().Interface().(Mergable); ok {
		mergeable.Merge(src.Interface())
		return obj
	}

	switch obj.Kind() {
	case reflect.Ptr:
		// We already checked for nil pointers, so just recurse on the Elem of the value.
		obj.Elem().Set(merge(obj.Elem(), src.Elem(), name))

	case reflect.Struct:
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
			if _, ok := structField.Tag.Lookup("union"); ok && !recurseIntoUnion {
				continue
			}
			fieldName := fmt.Sprintf("%v.%v", name, structField.Name)
			x := merge(obj.Field(i), src.Field(i), fieldName)
			obj.Field(i).Set(x)
		}

	case reflect.Map:
		// Maps get fused together; all input keys are written into the output map.
		for _, key := range src.MapKeys() {
			// Ensure key is not already set in obj.
			if objVal := obj.MapIndex(key); objVal.IsValid() {
				continue
			}
			val := src.MapIndex(key)
			obj.SetMapIndex(key, cpy(val))
		}

	case reflect.Slice:
		// Slices get copied only if the original was a nil pointer, which should always pass
		// through the cpy() codepath and never through here.

	// Assert that none of the "complex" kinds are present.
	case reflect.Array,
		reflect.Chan,
		reflect.Func,
		reflect.Interface,
		reflect.UnsafePointer:
		panic(fmt.Sprintf("unable to fill %T with %T at %v", obj.Interface(), src.Interface(), name))

		// Nothing to do for the simple Kinds like string or int; the only way a simple kind in the
		// src can end up being merged into the obj is if it is within a call to cpy(), like after
		// allocating a new pointer.  This is because we only merge into nil pointers.
	}

	return obj
}

// cpy is for deep copying, but it will only work on "nice" objects, which should include our
// schema objects.
func cpy(v reflect.Value) reflect.Value {
	// fmt.Printf("cpy(%T)\n", v.Interface())
	var out reflect.Value

	switch v.Kind() {
	case reflect.Ptr:
		if v.IsZero() {
			return v
		}
		out = reflect.New(v.Elem().Type())
		out.Elem().Set(cpy(v.Elem()))

	case reflect.Struct:
		out = reflect.New(v.Type()).Elem()
		// Recurse into each field of the struct.
		for i := 0; i < v.NumField(); i++ {
			out.Field(i).Set(cpy(v.Field(i)))
		}

	case reflect.Map:
		out = reflect.New(v.Type()).Elem()
		// Recurse into each key of the map.
		for _, key := range v.MapKeys() {
			val := v.MapIndex(key)
			out.SetMapIndex(key, cpy(val))
		}

	case reflect.Slice:
		out = reflect.New(v.Type()).Elem()
		// Recurse into each element of the slice.
		for i := 0; i < v.Len(); i++ {
			val := v.Index(i)
			out.Set(reflect.Append(out, cpy(val)))
		}

	// Assert that none of the "complex" kinds are present.
	case reflect.Array,
		reflect.Chan,
		reflect.Func,
		reflect.Interface,
		reflect.UnsafePointer:
		panic(fmt.Sprintf("unable to cpy %T", v.Interface()))

	default:
		// Simple types like string or int can be passed directly.
		return v
	}

	return out
}
