package schemas

import (
	"fmt"
	"reflect"
)

// mergeIfMergable checks if obj implements our Mergable psuedointerface and calls obj.Merge(src)
// if it does.
//
// The Mergable psuedointerface is defined as:
//
//	"x.Merge(src) operates on a non-pointer x, accepts src of the same type as x, and returns
//	another object of the same type as x"
//
// Note that the requirement .Merge must not operate on a pointer type is unlike most go methods.
//
// Mergable is not a real go interface, it's more of a "psuedointerface".  See explanation on
// copyIfCopyable.
//
// In practice, Mergable means an object can have custom merge behaviors.  Often this is used for
// combining lists, like bind_mounts or devices.
func mergeIfMergable(obj reflect.Value, src reflect.Value) (reflect.Value, bool) {
	var out reflect.Value

	// Look for the .WithDefaults method.
	meth, ok := obj.Type().MethodByName("Merge")
	if !ok {
		return out, false
	}

	// Verify the signature matches our Mergable psuedointerface:
	// - two inputs (the receiver), and one output
	// - input types match output type exactly (disallow the usual pointer receiver semantics)
	if meth.Type.NumIn() != 2 || meth.Type.NumOut() != 1 {
		return out, false
	}
	if meth.Type.In(0) != meth.Type.In(1) || meth.Type.In(0) != meth.Type.Out(0) {
		return out, false
	}

	// Psuedointerface matches, call the .Merge method.
	out = meth.Func.Call([]reflect.Value{obj, src})[0]

	return out, true
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
//	config, err := expconf.ParseAnyExperimentConfigYAML(bytes)
//
//	var cluster_default_storage expconf.CheckpointStorage = ...
//
//	// Use the cluster checkpoint storage if the user did not specify one.
//	config.RawCheckpointStorage = schemas.Merge(
//	    config.RawCheckpointStorage, &cluster_default_storage
//	)
func Merge[T any](obj T, src T) T {
	name := fmt.Sprintf("%T", obj)

	vObj := reflect.ValueOf(obj)
	vSrc := reflect.ValueOf(src)

	// obj must have the same type as src.
	assertTypeMatch(vObj, vSrc)

	return merge(vObj, vSrc, name).Interface().(T)
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
		return out.Convert(obj.Type())
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

	// Handle the `T Mergable[T]` pseudointerface
	if out, ok := mergeIfMergable(obj, src); ok {
		return out
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
		return out.Convert(obj.Type())

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

// UnionMerge implments the typical Merge logic for union types. The key is to merge all the common
// fields unconditionally, but to only merge the src's union member into the obj's union member if
// they are the same member, or if obj has no member.
func UnionMerge[T any](obj T, src T) T {
	name := fmt.Sprintf("%T", obj)

	vObj := reflect.ValueOf(obj)
	vSrc := reflect.ValueOf(src)

	// obj must have the same type as src.
	assertTypeMatch(vObj, vSrc)

	if vObj.Kind() != reflect.Struct {
		panic("UnionMerge must only be called on struct types")
	}

	return unionMerge(vObj, vSrc, name).Interface().(T)
}

// unionMerge is the reflect layer beneath UnionMerge.
func unionMerge(obj reflect.Value, src reflect.Value, name string) reflect.Value {
	out := reflect.New(obj.Type()).Elem()

	mergeField := func(i int) {
		structField := obj.Type().Field(i)
		fieldName := fmt.Sprintf("%v.%v", name, structField.Name)
		x := merge(obj.Field(i), src.Field(i), fieldName)
		out.Field(i).Set(x)
	}

	// Iterate through all the fields of the struct once, identifying union members and merging
	// the non-union members.
	objHasMember := -1
	srcHasMember := -1
	for i := 0; i < src.NumField(); i++ {
		if _, ok := obj.Type().Field(i).Tag.Lookup("union"); ok {
			// Union member, remember it for later.
			if !obj.Field(i).IsZero() {
				objHasMember = i
			}
			if !src.Field(i).IsZero() {
				srcHasMember = i
			}
			continue
		}
		// Non-union member, merge it normally.
		mergeField(i)
	}
	if objHasMember > -1 {
		// When obj has a union member, we can only merge that union member.
		mergeField(objHasMember)
	} else if srcHasMember > -1 {
		// Otherwise we merge whatever the src has defined.
		mergeField(srcHasMember)
	}
	return out.Convert(obj.Type())
}
