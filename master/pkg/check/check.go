package check

import (
	"reflect"
	"strings"

	"github.com/pkg/errors"
)

func check(condition bool, msgAndArgs []interface{}, internalMsgAndArgs ...interface{}) error {
	if condition {
		return nil
	}
	msgs := make([]string, 0, 3)
	if msg := messageFromMsgAndArgs(false, msgAndArgs...); msg != "" {
		msgs = append(msgs, msg)
	}
	if msg := messageFromMsgAndArgs(true, internalMsgAndArgs...); msg != "" {
		msgs = append(msgs, msg)
	}
	return errors.New(strings.Join(msgs, ": "))
}

// Panic panics if the error is not nil.
func Panic(err error) {
	if err != nil {
		panic(err.Error())
	}
}

// True checks whether the condition is true. This method returns an error with the provided
// message if the check fails.
func True(condition bool, msgAndArgs ...interface{}) error {
	return check(condition, msgAndArgs, "expected true, got false")
}

// TrueSilent checks whether the condition is true. This method returns an error containing only the
// provided message and nothing else if the check fails.
func TrueSilent(condition bool, msgAndArgs ...interface{}) error {
	return check(condition, msgAndArgs)
}

// False checks whether the condition is false. This method returns an error with the provided
// message if the check fails.
func False(condition bool, msgAndArgs ...interface{}) error {
	return check(!condition, msgAndArgs, "expected false, got true")
}

// Equal checks whether the arguments are equal (via reflect.DeepEqual). This method returns an
// error with the provided message if the check fails.
func Equal(actual, expected interface{}, msgAndArgs ...interface{}) error {
	return check(reflect.DeepEqual(actual, expected), msgAndArgs,
		"%s does not equal %s", format(actual), format(expected))
}
