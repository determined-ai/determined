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
