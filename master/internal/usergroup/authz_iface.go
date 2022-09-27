package usergroup

import (
	"github.com/determined-ai/determined/master/internal/authz"
	"github.com/determined-ai/determined/master/pkg/model"
	"github.com/determined-ai/determined/proto/pkg/groupv1"
)

// UserGroupAuthZ describes authz methods for `user` package.
type UserGroupAuthZ interface {
	// CanGetGroup checks whether a user can get a group.
	// GET /api/v1/groups/{group_id}
	CanGetGroup(curUser model.User, gid int) error

	// FilterGroupsList checks what groups a user can get.
	// POST /api/v1/groups/search
	FilterGroupsList(curUser model.User, groups []*groupv1.GroupSearchResult) (
		[]*groupv1.GroupSearchResult, error)

	// CanUpdateGroups checks if a user can create, delete, or update a group.
	// POST /api/v1/groups
	// PUT /api/v1/groups/{group_id}
	// DELETE /api/v1/groups/{group_id}
	CanUpdateGroups(curUser model.User) error
}

// AuthZProvider is the authz registry for `user` package.
var AuthZProvider authz.AuthZProviderType[UserGroupAuthZ]
