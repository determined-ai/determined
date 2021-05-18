package schemas

import (
	"fmt"
	"reflect"
)

// Copyable means an object can have custom behvaiors for schemas.Copy.  Mostly this is useful for
// working around types which we do not own and which schemas.Copy() would puke on.
type Copyable interface {
	// Copy should return the same type.  It must not be defined as a method against a pointer of
	// the type or it will not work.
	Copy() interface{}
}

// cpy is for deep copying, but it will only work on "nice" objects, which should include our
// schema objects.  Useful to other reflect code.
func cpy(v reflect.Value) reflect.Value {
	// fmt.Printf("cpy(%T)\n", v.Interface())
	var out reflect.Value

	// Detect Copyables, but disallow pointers.  The reason is that if you have a method like:
	//    func (t Thing) Copy() interface{}
	// then Copy() will return a plain Thing object, but a pointer to a Thing will still be treated
	// as copyable.  Then, schemas.Copy(&t).(*Thing) would panic because it returns the wrong type.
	if v.Kind() != reflect.Ptr {
		if copyable, ok := v.Interface().(Copyable); ok {
			return reflect.ValueOf(copyable.Copy())
		}
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
