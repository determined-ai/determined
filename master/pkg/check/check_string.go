package check

import (
	"regexp"
	"strings"
)

// In checks whether the first argument is in the second argument. This method returns an error
// with the provided message if the check fails.
func In(actual string, expected []string, msgAndArgs ...interface{}) error {
	for _, e := range expected {
		if actual == e {
			return nil
		}
	}
	return check(false, msgAndArgs, "%s is not in {%v}", actual, strings.Join(expected, ", "))
}

// NotEmpty checks whether the first argument is empty string. The method returns an error with the
// provided message if the string is empty.
func NotEmpty(actual string, msgAndArgs ...interface{}) error {
	return check(len(actual) > 0, msgAndArgs, "%s must be non-empty", actual)
}

// Match checks whether the first argument matches the regular expression of the second argument.
// The method returns an error with the provided message if the check fails.
func Match(actual string, regex string, msgAndArgs ...interface{}) error {
	compiled := regexp.MustCompile(regex)
	compiled.Longest()
	return check(compiled.FindString(actual) == actual, msgAndArgs,
		"%s doesn't match regex %s", actual, regex)
}

// LenBetween checks whether the length of the first argument is between the second and third arguments.
// The method returns an error with the provided message if the check fails.
func LenBetween(actual string, min, max int, msgAndArgs ...interface{}) error {
	return BetweenInclusive(len(actual), min, max, msgAndArgs...)
}

// IsValidK8sLabel checks whether the first argument is a valid Kubernetes label. The method returns
// an error with the provided message if the check fails.
func IsValidK8sLabel(actual string, msgAndArgs ...interface{}) error {
	if err := NotEmpty(actual, msgAndArgs...); err != nil {
		return err
	}
	if err := LenBetween(actual, 1, 63, msgAndArgs...); err != nil {
		return err
	}
	if err := Match(
		actual, `^[a-zA-Z0-9]([-a-zA-Z0-9_.]*[a-zA-Z0-9])?$`, msgAndArgs...,
	); err != nil {
		return err
	}
	return nil
}
