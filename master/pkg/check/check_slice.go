package check

// Contains checks whether the actual value is contained in the expected list. This method returns
// an error with the provided message if the check fails.
func Contains(actual interface{}, expected []interface{}, msgAndArgs ...interface{}) error {
	for _, value := range expected {
		if value == actual {
			return nil
		}
	}
	return check(false, msgAndArgs, "%s not in %s", actual, expected)
}
