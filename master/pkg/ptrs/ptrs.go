package ptrs

// Ptr returns a pointer to a copy of a value.  This lets you take the address of some
// non-addressable[1] values, like boolean literals.  So if a struct has a *bool field, you can do:
//
//	x := MyStruct{MyBool: Ptr(true)}
//
// instead of:
//
//	temp := true
//	x := MyStruct{MyBool: &temp}
//
// [1] https://go.dev/ref/spec#Address_operators
func Ptr[T any](t T) *T {
	return &t
}
