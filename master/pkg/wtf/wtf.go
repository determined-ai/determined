package wtf

// wtf is a package of things that go should defintely already have.

// Ptr returns a pointer to a copy of a value.  The copy that occurs is the normal copy that occurs
// when passing a value as an argument.  This allows you to effectively take the address of some
// non-addressible[1] values, like boolean literals.  So if a struct has a *bool field, you can do:
//
//     x := MyStruct{MyBool: &Ptr(true)}
//
// instead of:
//
//     temp := true
//     x := MyStruct{MyBool: &temp}
//
// [1] https://go.dev/ref/spec#Address_operators
func Ptr[T any](t T) *T {
	x := t
	return &x
}
