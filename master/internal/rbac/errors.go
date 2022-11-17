package rbac

import (
	"github.com/pkg/errors"
)

// ErrGlobalAssignedLocally occurs when an attempt is made to assign a role with a global-only
// permission using a non-global scope.
// nolint:lll
var ErrGlobalAssignedLocally = errors.New("a global-only permission cannot be assigned to a local scope")
