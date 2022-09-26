package usergroup

import (
	"github.com/determined-ai/determined/master/internal/authz"
	"github.com/determined-ai/determined/master/pkg/model"
	"github.com/determined-ai/determined/proto/pkg/groupv1"
)

// UserGroupAuthZ describes authz methods for `user` package.
type UserGroupAuthZ interface {

	// CanGetGroup checks whether a user can get a group
	// GET /api/v1/groups/{group_id}
	CanGetGroup(curUser model.User) error

	// FilterGroupsList checks what groups a user can get
	// POST /api/v1/groups/search
	// Search for one or more groups
	FilterGroupsList(curUser model.User, groups []*groupv1.GroupSearchResult) ([]*groupv1.GroupSearchResult, error)

	// POST /api/v1/groups
	// Create Groups
	CanCreateGroups(curUser model.User) error

	// PUT /api/v1/groups/{group_id}
	// Update a group
	CanUpdateGroup(curUser model.User) error

	// DELETE /api/v1/groups/{group_id}
	// Delete a group
	CanDeleteGroup(curUser model.User) error
}

// AuthZProvider is the authz registry for `user` package.
var AuthZProvider authz.AuthZProviderType[UserGroupAuthZ]

