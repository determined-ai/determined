package usergroup

import (
	"context"

	"github.com/uptrace/bun"

	"github.com/determined-ai/determined/master/internal/authz"
	"github.com/determined-ai/determined/master/pkg/model"
)

// UserGroupAuthZ describes authz methods for `user` package.
type UserGroupAuthZ interface {
	// CanGetGroup checks whether a user can get a group.
	// GET /api/v1/groups/{group_id}
	CanGetGroup(ctx context.Context, curUser model.User, gid int) error

	// FilterGroupsList checks what groups a user can get.
	// POST /api/v1/groups/search
	FilterGroupsList(ctx context.Context, curUser model.User, query *bun.SelectQuery) (
		*bun.SelectQuery, error)

	// CanUpdateGroups checks if a user can create, delete, or update a group.
	// POST /api/v1/groups
	// PUT /api/v1/groups/{group_id}
	// DELETE /api/v1/groups/{group_id}
	CanUpdateGroups(ctx context.Context, curUser model.User) error
}

// AuthZProvider is the authz registry for `user` package.
var AuthZProvider authz.AuthZProviderType[UserGroupAuthZ]
