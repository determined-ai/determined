package schemas

import (
	"fmt"
	"reflect"
)

// copyIfCopyable checks if v implements our Copyable psuedointerface and calls .Copy if it does.
//
// The Copyable psuedointerface is defined as:
//
//     "x.Copy() returns another object with the same type as x".
//
// It is a "psuedointerface", not a real interface, because it can't actually be expressed in go
// generics.  If you wanted to approximate it, you would end up with something like:
//
//     // CopyReturnsT has a Copy() that returns a T.
//     type CopyReturnsT[T any] interface {
//         Copy() T
//     }
//
//     // Operates on any type T which has have a .Copy() that returns a T.
//     func Copy[T CopyReturnsT[T]](src T) T { ... }
//
// But since the resulting constraint [T CopyReturns[T]] isn't a concrete interface, we can't check
// for it in reflect code, and instead we just manually check if an object meets our definition of
// "Copyable".
//
// In practice, Copyable means an object can have custom behvaiors for schemas.Copy.  Mostly this is
// useful for working around types which we do not own and which schemas.Copy() would puke on.
func copyIfCopyable(v reflect.Value) (reflect.Value, bool) {
	var out reflect.Value

	// Look for the .Copy method.
	meth, ok := v.Type().MethodByName("Copy")
	if !ok {
		return out, false
	}

	// Verify the signature matches our Copyable psuedointerface:
	// - one input (the receiver), and one output
	// - input type matches output type exactly (without the usual pointer receiver semantics)
	if meth.Type.NumIn() != 1 || meth.Type.NumOut() != 1 || meth.Type.In(0) != meth.Type.Out(0) {
		return out, false
	}

	// Psuedointerface matches, call the .Copy method.
	out = meth.Func.Call([]reflect.Value{v})[0]

	return out, true
}

// Copy is a reflect-based deep copy.  It's only generally safe to use on schema objects.
func Copy[T any](src T) T {
	return cpy(reflect.ValueOf(src)).Interface().(T)
}

// cpy is for deep copying, but it will only work on "nice" objects, which should include our
// schema objects.  Useful to other reflect code.
func cpy(v reflect.Value) reflect.Value {
	// fmt.Printf("cpy(%T)\n", v.Interface())
	var out reflect.Value

	// Detect values which match the Copyable pseudointerface.
	var ok bool
	if out, ok = copyIfCopyable(v); ok {
		return out
	}

	switch v.Kind() {
	case reflect.Ptr:
		if v.IsZero() {
			return v
		}
		out = reflect.New(v.Elem().Type())
		out.Elem().Set(cpy(v.Elem()))

	case reflect.Interface:
		if v.IsZero() {
			return v
		}
		out = cpy(v.Elem())

	case reflect.Struct:
		out = reflect.New(v.Type()).Elem()
		// Recurse into each field of the struct.
		for i := 0; i < v.NumField(); i++ {
			out.Field(i).Set(cpy(v.Field(i)))
		}

	case reflect.Map:
		typ := reflect.MapOf(v.Type().Key(), v.Type().Elem())
		if v.IsZero() {
			// unallocated map
			out = reflect.Zero(typ)
		} else {
			out = reflect.MakeMap(typ)
			// Recurse into each key of the map.
			for _, key := range v.MapKeys() {
				val := v.MapIndex(key)
				out.SetMapIndex(key, cpy(val))
			}
		}

	case reflect.Slice:
		typ := reflect.SliceOf(v.Type().Elem())
		if v.IsZero() {
			// unallocated slice
			out = reflect.Zero(typ)
		} else {
			out = reflect.MakeSlice(typ, 0, v.Len())
			// Recurse into each element of the slice.
			for i := 0; i < v.Len(); i++ {
				val := v.Index(i)
				out = reflect.Append(out, cpy(val))
			}
		}

	// Assert that none of the "complex" kinds are present.
	case reflect.Array,
		reflect.Chan,
		reflect.Func,
		reflect.UnsafePointer:
		panic(fmt.Sprintf("unable to cpy %T of kind %v", v.Interface(), v.Kind()))

	default:
		// Simple types like string or int can be passed directly.
		return v
	}

	return out.Convert(v.Type())
}
