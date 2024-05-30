package check

import (
	"regexp"
	"strconv"
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

// IsValidK8sLabel checks whether the first argument is a valid Kubernetes label. The method returns
// an error with the provided message if the check fails.
func IsValidK8sLabel(actual string, msgAndArgs ...interface{}) error {
	re := regexp.MustCompile(`^[a-zA-Z0-9]([-a-zA-Z0-9_.]*[a-zA-Z0-9])?$`)
	if !re.MatchString(actual) {
		return check(false, msgAndArgs, "%s is not a valid Kubernetes label", actual)
	}
	return check(len(actual) > 0 && len(actual) < 64, msgAndArgs,
		"%s is not between 1 and 63 chars ", actual)
}

// IsValidIPV4 checks whether the first argument is a valid IPv4 address. The method returns an error
// with the provided message if the check fails.
func IsValidIPV4(actual string, msgAndArgs ...interface{}) error {
	re := regexp.MustCompile(`^(?:[0-9]{1,3}\.){3}[0-9]{1,3}$`)
	if !re.MatchString(actual) {
		return check(false, msgAndArgs, "%s is not a valid IPv4 address", actual)
	}
	parts := strings.Split(actual, ".")
	for _, part := range parts {
		num, err := strconv.Atoi(part)
		if err != nil || num < 0 || num > 255 {
			return check(false, msgAndArgs, "%s is not a valid IPv4 address", actual)
		}
	}
	return nil
}
