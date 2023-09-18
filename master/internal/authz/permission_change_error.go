package authz

import (
	"strings"
)

const (
	permissionChangeErrorString = "permission change detected while streaming updates"
)

// PermissionChangeError represents an error that arises when a permission change
// invalidates a streaming client permission cache.
type PermissionChangeError struct{}

// Error returns an error string.
func (p PermissionChangeError) Error() string {
	return permissionChangeErrorString
}

// IsPermissionChangeError checks if err is of type PermissionChangeError.
func IsPermissionChangeError(err error) bool {
	if err == nil {
		return false
	}
	if _, ok := err.(PermissionChangeError); ok {
		return true
	}
	if strings.Contains(err.Error(), permissionChangeErrorString) {
		return true
	}
	return false
}
