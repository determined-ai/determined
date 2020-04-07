package check

import (
	"reflect"
)

func elem(a interface{}) (interface{}, bool) {
	value := reflect.ValueOf(a)
	if value.Kind() == reflect.Ptr {
		if value.IsNil() {
			return nil, true
		}
		return value.Elem().Interface(), false
	}
	return a, false
}

func compareInt(a, b int) int {
	switch {
	case a > b:
		return 1
	case a < b:
		return -1
	default:
		return 0
	}
}

func compareInt8(a, b int8) int {
	switch {
	case a > b:
		return 1
	case a < b:
		return -1
	default:
		return 0
	}
}

func compareInt16(a, b int16) int {
	switch {
	case a > b:
		return 1
	case a < b:
		return -1
	default:
		return 0
	}
}

func compareInt32(a, b int32) int {
	switch {
	case a > b:
		return 1
	case a < b:
		return -1
	default:
		return 0
	}
}

func compareInt64(a, b int64) int {
	switch {
	case a > b:
		return 1
	case a < b:
		return -1
	default:
		return 0
	}
}

func compareUint(a, b uint) int {
	switch {
	case a > b:
		return 1
	case a < b:
		return -1
	default:
		return 0
	}
}

func compareUint8(a, b uint8) int {
	switch {
	case a > b:
		return 1
	case a < b:
		return -1
	default:
		return 0
	}
}

func compareUint16(a, b uint16) int {
	switch {
	case a > b:
		return 1
	case a < b:
		return -1
	default:
		return 0
	}
}

func compareUint32(a, b uint32) int {
	switch {
	case a > b:
		return 1
	case a < b:
		return -1
	default:
		return 0
	}
}

func compareUint64(a, b uint64) int {
	switch {
	case a > b:
		return 1
	case a < b:
		return -1
	default:
		return 0
	}
}

func compareFloat32(a, b float32) int {
	switch {
	case a > b:
		return 1
	case a < b:
		return -1
	default:
		return 0
	}
}

func compareFloat64(a, b float64) int {
	switch {
	case a > b:
		return 1
	case a < b:
		return -1
	default:
		return 0
	}
}

func compare(a, b interface{}) (int, bool) {
	switch aTyped := a.(type) {
	case int:
		bTyped, ok := b.(int)
		return compareInt(aTyped, bTyped), ok
	case int8:
		bTyped, ok := b.(int8)
		return compareInt8(aTyped, bTyped), ok
	case int16:
		bTyped, ok := b.(int16)
		return compareInt16(aTyped, bTyped), ok
	case int32:
		bTyped, ok := b.(int32)
		return compareInt32(aTyped, bTyped), ok
	case int64:
		bTyped, ok := b.(int64)
		return compareInt64(aTyped, bTyped), ok
	case uint:
		bTyped, ok := b.(uint)
		return compareUint(aTyped, bTyped), ok
	case uint8:
		bTyped, ok := b.(uint8)
		return compareUint8(aTyped, bTyped), ok
	case uint16:
		bTyped, ok := b.(uint16)
		return compareUint16(aTyped, bTyped), ok
	case uint32:
		bTyped, ok := b.(uint32)
		return compareUint32(aTyped, bTyped), ok
	case uint64:
		bTyped, ok := b.(uint64)
		return compareUint64(aTyped, bTyped), ok
	case float32:
		bTyped, ok := b.(float32)
		return compareFloat32(aTyped, bTyped), ok
	case float64:
		bTyped, ok := b.(float64)
		return compareFloat64(aTyped, bTyped), ok
	}
	return 0, false
}

func maybeCompare(
	actual, expected interface{}, comparator func(v int) bool,
	msgAndArgs []interface{}, internalMsgAndArgs ...interface{},
) error {
	actualValue, actualIsNil := elem(actual)
	expectedValue, expectedIsNil := elem(expected)
	comparison, ok := compare(actualValue, expectedValue)
	switch {
	case actualIsNil || expectedIsNil:
		return nil
	case !ok:
		return check(false, msgAndArgs, "incomparable types %s(%v) and %s(%v)",
			reflect.TypeOf(actual).Kind(), actual, reflect.TypeOf(expected).Kind(), expected)
	case !comparator(comparison):
		return check(false, msgAndArgs, internalMsgAndArgs...)
	default:
		return nil
	}
}

// GreaterThan checks whether `actual` is greater than `expected`.
func GreaterThan(actual, expected interface{}, msgAndArgs ...interface{}) error {
	return maybeCompare(actual, expected, func(comparison int) bool { return comparison > 0 },
		msgAndArgs, "%v is not greater than %v", actual, expected)
}

// GreaterThanOrEqualTo checks whether `actual` is greater than or equal to `expected`.
func GreaterThanOrEqualTo(actual, expected interface{}, msgAndArgs ...interface{}) error {
	return maybeCompare(actual, expected, func(comparison int) bool { return comparison >= 0 },
		msgAndArgs, "%v is not greater than or equal to %v", actual, expected)
}

// LessThan checks whether `actual` is less than `expected`.
func LessThan(actual, expected interface{}, msgAndArgs ...interface{}) error {
	return maybeCompare(actual, expected, func(comparison int) bool { return comparison < 0 },
		msgAndArgs, "%v is not less than %v", actual, expected)
}

// LessThanOrEqualTo checks whether `actual` is less than or equal to `expected`.
func LessThanOrEqualTo(actual, expected interface{}, msgAndArgs ...interface{}) error {
	return maybeCompare(actual, expected, func(comparison int) bool { return comparison <= 0 },
		msgAndArgs, "%v is not less than or equal to %v", actual, expected)
}
