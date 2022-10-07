package authz

import (
	"fmt"
	"strings"

	"github.com/determined-ai/determined/proto/pkg/rbacv1"
)

// PermissionDeniedError represents an error that arises when a user does not have sufficient
// access privileges. RequiredPermissions can be empty for non-rbac errors.
type PermissionDeniedError struct {
	RequiredPermissions []*rbacv1.Permission
}

// Error returns an error string.
func (p PermissionDeniedError) Error() string {
	if len(p.RequiredPermissions) == 0 {
		return "access denied"
	}

	permissions := make([]string, len(p.RequiredPermissions))
	for i, perm := range p.RequiredPermissions {
		permissions[i] = perm.Name
	}
	return fmt.Sprintf("access denied; required permissions: %s", strings.Join(
		permissions, ", "))
}
