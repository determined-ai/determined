package user

import (
	"github.com/determined-ai/determined/master/pkg/model"
)

// IsSSOUser checks whether user has restrictions based on being
// an SSO or SCIM user.
func IsSSOUser(_ model.User) bool {
	return false
}
