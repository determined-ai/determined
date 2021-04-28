package schemas

import (
	"fmt"
	"reflect"
)

// Mergable means an object can have custom behvaiors for schemas.Merge.
type Mergable interface {
	// Merge should take a struct and return the same struct.
	Merge(interface{}) interface{}
}

// Merge will recurse through two objects of the same type and return a merged version
// (a clean copy).
//
// The default behavior for merging maps is to include keys from both src and obj, while the default
// behavior for slices is to use one or the other.  This is analgous to how json.Unmarshal treats
// maps and slices.  However, the default merging behavior for an object can be overwritten by
// implementing the Mergable interface.  An example of this is BindMountsConfig.
//
// Example usage:
//
//    config, err := expconf.ParseAnyExperimentConfigYAML(bytes)
//
//    var cluster_default_storage expconf.CheckpointStorage = ...
//
//    // Use the cluster checkpoint storage if the user did not specify one.
//    config.RawCheckpointStorage = schemas.Merge(
//        config.RawCheckpointStorage, &cluster_default_storage
//    ).(CheckpointStorageConfig)
//
func Merge(obj interface{}, src interface{}) interface{} {
	name := fmt.Sprintf("%T", obj)

	vObj := reflect.ValueOf(obj)
	vSrc := reflect.ValueOf(src)

	// obj must have the same type as src.
	assertTypeMatch(vObj, vSrc)

	return merge(vObj, vSrc, name).Interface()
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

	// Always handle pointers first.
	if obj.Kind() == reflect.Ptr {
		if obj.IsZero() {
			return cpy(src)
		} else if src.IsZero() {
			return cpy(obj)
		}
		out := reflect.New(obj.Elem().Type())
		out.Elem().Set(merge(obj.Elem(), src.Elem(), name))
		return out
	}

	// Next handle interfaces.
	if obj.Kind() == reflect.Interface {
		if obj.IsZero() {
			return cpy(src)
		} else if src.IsZero() {
			return cpy(obj)
		}
		return merge(obj.Elem(), src.Elem(), name)
	}

	// Detect Mergables.
	if mergeable, ok := obj.Interface().(Mergable); ok {
		return reflect.ValueOf(mergeable.Merge(src.Interface()))
	}

	switch obj.Kind() {
	case reflect.Struct:
		// Recurse into each field of the struct.
		out := reflect.New(obj.Type()).Elem()
		for i := 0; i < src.NumField(); i++ {
			structField := src.Type().Field(i)
			fieldName := fmt.Sprintf("%v.%v", name, structField.Name)
			x := merge(obj.Field(i), src.Field(i), fieldName)
			out.Field(i).Set(x)
		}
		return out

	case reflect.Map:
		// Handle unallocated maps on either input.
		if src.IsZero() {
			return cpy(obj)
		} else if obj.IsZero() {
			return cpy(src)
		}
		// allocate a new map
		typ := reflect.MapOf(obj.Type().Key(), obj.Type().Elem())
		out := reflect.MakeMap(typ)
		// Iterate through keys and objects in obj.
		iter := obj.MapRange()
		for iter.Next() {
			key := iter.Key()
			objVal := iter.Value()
			if srcVal := src.MapIndex(key); srcVal.IsValid() {
				// Key present in both maps.
				out.SetMapIndex(key, merge(objVal, srcVal, name))
			} else {
				// Key is unique to obj.
				out.SetMapIndex(key, cpy(objVal))
			}
		}
		// Check for keys only present in src.
		iter = src.MapRange()
		for iter.Next() {
			key := iter.Key()
			srcVal := iter.Value()
			if objVal := obj.MapIndex(key); !objVal.IsValid() {
				// Key is unique to src.
				out.SetMapIndex(key, cpy(srcVal))
			}
		}
		return out.Convert(obj.Type())

	case reflect.Slice:
		// Slices are not merged by default.  If obj is nil we copy the src.
		if obj.IsZero() {
			return cpy(src)
		}
		return cpy(obj)

	// Assert that none of the "complex" kinds are present.
	case reflect.Array,
		reflect.Chan,
		reflect.Func,
		reflect.UnsafePointer,
		// We already handled Ptr and Interface.
		reflect.Ptr,
		reflect.Interface:
		panic(fmt.Sprintf("unable to fill %T with %T at %v", obj.Interface(), src.Interface(), name))

	default:
		// Simple kinds just get copied.
		return cpy(obj)
	}
}
