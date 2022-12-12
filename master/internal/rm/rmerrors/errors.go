package rmerrors

// ErrUnsupported is returned when an unsupported feature of a resource manager is used.
type ErrUnsupported string

func (e ErrUnsupported) Error() string {
	return string(e)
}
