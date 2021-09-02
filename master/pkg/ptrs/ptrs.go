package ptrs

import "time"

// BoolPtr is the "&true" you always wanted.
func BoolPtr(val bool) *bool {
	tmp := val
	return &tmp
}

// IntPtr is the "&int(1)" you always wanted.
func IntPtr(val int) *int {
	tmp := val
	return &tmp
}

// Float64Ptr is the "&float64(1)" you always wanted.
func Float64Ptr(val float64) *float64 {
	tmp := val
	return &tmp
}

// StringPtr is the "&string("asdf")" you always wanted.
func StringPtr(val string) *string {
	tmp := val
	return &tmp
}

// TimePtr is the &time.Now().UTC() you always wanted.
func TimePtr(val time.Time) *time.Time {
	return &val
}
