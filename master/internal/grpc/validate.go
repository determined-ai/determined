package grpc

import (
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

// Check returns a bool value denoting whether the check passed or failed. If the check fails, the
// string denotes the error reason.
type Check func() (bool, string)

// ValidateRequest validates that all the checks pass. If a check does not pass, an InvalidArgument
// error is returned.
func ValidateRequest(checks ...Check) error {
	for _, check := range checks {
		result, err := check()
		if !result {
			return status.Error(codes.InvalidArgument, err)
		}
	}
	return nil
}
