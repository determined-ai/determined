package authz

import (
	"fmt"
	"strings"

	"github.com/determined-ai/determined/proto/pkg/rbacv1"
)

// PermissionDeniedError represents an error that arises when a user does not have sufficient
// access privileges. RequiredPermissions can be empty for non-rbac errors.
type PermissionDeniedError struct {
	RequiredPermissions []rbacv1.PermissionType
	OneOf               bool

	// optional prefix error message
	Prefix string
}

// Error returns an error string.
func (p PermissionDeniedError) Error() string {
	if len(p.RequiredPermissions) == 0 {
		return strings.TrimSpace(fmt.Sprintf("%s access denied", p.Prefix))
	}

	permissions := make([]string, len(p.RequiredPermissions))
	for i, perm := range p.RequiredPermissions {
		permissions[i] = rbacv1.PermissionType_name[int32(perm)]
	}

	permStr := "access denied; required permissions:"
	if p.OneOf {
		permStr = "access denied; one of the following permissions required:"
	}

	return strings.TrimSpace(fmt.Sprintf(
		"%s %s %s",
		p.Prefix,
		permStr,
		strings.Join(permissions, ", ")))
}

// WithPrefix adds a custom prefix to error string.
func (p PermissionDeniedError) WithPrefix(prefix string) PermissionDeniedError {
	p.Prefix = prefix
	return p
}

// IsPermissionDenied checks if err is of type PermissionDeniedError.
func IsPermissionDenied(err error) bool {
	if err == nil {
		return false
	}
	if _, ok := err.(PermissionDeniedError); ok {
		return true
	}
	if strings.Contains(err.Error(), "access denied") {
		return true
	}
	return false
}

// SubIfUnauthorized substitutes an error if it is of type PermissionDeniedError.
func SubIfUnauthorized(err error, sub error) error {
	if IsPermissionDenied(err) {
		return sub
	}
	return err
}
